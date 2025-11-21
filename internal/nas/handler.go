package nas

import (
	"encoding/hex"
	"fmt"

	"github.com/gavin/amf/internal/context"
	"github.com/gavin/amf/internal/consumer"
	"github.com/gavin/amf/internal/logger"
)

type Handler struct {
	amfContext *context.AMFContext
	ngapHandler NGAPHandler
}

type NGAPHandler interface {
	SendDownlinkNASTransport(ranUeNgapId, amfUeNgapId int64, nasPDU []byte) error
	SendPDUSessionResourceSetupRequest(ranUeNgapId, amfUeNgapId int64, pduSessionId uint8, nasPDU []byte, n2SmInfo []byte) error
}

func NewHandler(ctx *context.AMFContext) *Handler {
	return &Handler{
		amfContext: ctx,
	}
}

func (h *Handler) SetNGAPHandler(handler NGAPHandler) {
	h.ngapHandler = handler
}

func (h *Handler) HandleNASMessage(ue *context.UEContext, nasPDU []byte) error {
	logger.NasLog.Infof("Handling NAS message for UE AMF NGAP ID: %d", ue.AmfUeNgapId)

	pdu, err := DecodeNASPDU(nasPDU)
	if err != nil {
		return fmt.Errorf("failed to decode NAS PDU: %v", err)
	}

	if pdu.SecurityHeaderType != SecurityHeaderTypePlainNAS && ue.SecurityContext.Activated {
		pdu, err = DecodeSecuredNASPDU(ue, nasPDU)
		if err != nil {
			return fmt.Errorf("failed to decode secured NAS PDU: %v", err)
		}
	}

	logger.NasLog.Infof("NAS Message Type: 0x%02x", pdu.MessageType)

	switch pdu.MessageType {
	case MsgTypeRegistrationRequest:
		return h.HandleRegistrationRequest(ue, pdu.Payload)
	case MsgTypeAuthenticationResponse:
		return h.HandleAuthenticationResponse(ue, pdu.Payload)
	case MsgTypeSecurityModeComplete:
		return h.HandleSecurityModeComplete(ue, pdu.Payload)
	case MsgTypeDeregistrationRequestUEOriginating:
		return h.HandleDeregistrationRequest(ue, pdu.Payload)
	case MsgTypeServiceRequest:
		return h.HandleServiceRequest(ue, pdu.Payload)
	case MsgTypeRegistrationComplete:
		logger.NasLog.Infof("Registration Complete received for UE: %s", ue.Supi)
		return nil
	case MsgTypeULNASTransport:
		return h.HandleULNASTransport(ue, pdu.Payload)
	default:
		logger.NasLog.Warnf("Unsupported NAS message type: 0x%02x", pdu.MessageType)
		return nil
	}
}

func (h *Handler) HandleRegistrationRequest(ue *context.UEContext, payload []byte) error {
	logger.NasLog.Infof("Handle Registration Request for UE")

	regReq, err := DecodeRegistrationRequest(payload)
	if err != nil {
		return fmt.Errorf("failed to decode registration request: %v", err)
	}

	ue.RegistrationType = regReq.RegistrationType
	ue.NgKsi = int(regReq.NgKSI)

	if len(regReq.MobileIdentity) > 0 {
		suci := hex.EncodeToString(regReq.MobileIdentity)
		ue.Supi = suci
		logger.NasLog.Infof("Registration Request from SUCI: %s", suci)
	}

	if len(regReq.UESecurityCapability) > 0 {
		ue.UeSecurityCapability = hex.EncodeToString(regReq.UESecurityCapability)
	}

	ue.RegistrationState = context.RegStateRegistering
	ue.RmState = context.RmRegistered

	if err := h.amfContext.PersistUEContext(ue); err != nil {
		logger.NasLog.Warnf("Failed to persist UE context: %v", err)
	}

	ausfClient := consumer.NewAUSFClient(h.amfContext.AusfUri)
	servingNetwork := fmt.Sprintf("5G:mnc%s.mcc%s.3gppnetwork.org",
		h.amfContext.ServedGuami[0].PlmnId.Mnc,
		h.amfContext.ServedGuami[0].PlmnId.Mcc)

	authResp, err := ausfClient.RequestAuthentication(ue.Supi, servingNetwork)
	if err != nil {
		logger.NasLog.Errorf("Authentication request failed: %v", err)
		return h.SendRegistrationReject(ue, CauseProtocolError)
	}

	if authResp.Links != nil {
		if authCtxId, ok := authResp.Links["5g-aka"]; ok {
			ue.AuthenticationCtxId = authCtxId
		}
	}

	if av, ok := authResp.AuthenticationVector.(map[string]interface{}); ok {
		var rand, autn []byte

		if randStr, ok := av["rand"].(string); ok {
			rand, _ = hex.DecodeString(randStr)
		}
		if autnStr, ok := av["autn"].(string); ok {
			autn, _ = hex.DecodeString(autnStr)
		}

		return h.SendAuthenticationRequest(ue, rand, autn)
	}

	return fmt.Errorf("no authentication vector in response")
}

func (h *Handler) SendAuthenticationRequest(ue *context.UEContext, rand, autn []byte) error {
	logger.NasLog.Infof("Sending Authentication Request to UE")

	msg := &AuthenticationRequestMsg{
		NgKSI: uint8(ue.NgKsi),
		RAND:  rand,
		AUTN:  autn,
		ABBA:  []byte{0x00, 0x00},
	}

	payload := EncodeAuthenticationRequest(msg)
	pdu := &NASPDU{
		ProtocolDiscriminator: ProtocolDiscriminator5GMM,
		SecurityHeaderType:    SecurityHeaderTypePlainNAS,
		MessageType:           MsgTypeAuthenticationRequest,
		Payload:               payload,
	}

	nasData := EncodeNASPDU(pdu)
	return h.ngapHandler.SendDownlinkNASTransport(ue.RanUeNgapId, ue.AmfUeNgapId, nasData)
}

func (h *Handler) HandleAuthenticationResponse(ue *context.UEContext, payload []byte) error {
	logger.NasLog.Infof("Handle Authentication Response for UE SUPI: %s", ue.Supi)

	authResp, err := DecodeAuthenticationResponse(payload)
	if err != nil {
		return fmt.Errorf("failed to decode authentication response: %v", err)
	}

	if len(authResp.RES) == 0 {
		return fmt.Errorf("no RES in authentication response")
	}

	ausfClient := consumer.NewAUSFClient(h.amfContext.AusfUri)
	resHex := hex.EncodeToString(authResp.RES)

	confirmResp, err := ausfClient.ConfirmAuthentication(ue.AuthenticationCtxId, resHex)
	if err != nil {
		logger.NasLog.Errorf("Authentication confirmation failed: %v", err)
		return h.SendAuthenticationReject(ue)
	}

	if confirmResp.AuthResult != "AUTHENTICATION_SUCCESS" {
		logger.NasLog.Errorf("Authentication failed: %s", confirmResp.AuthResult)
		return h.SendAuthenticationReject(ue)
	}

	if confirmResp.Kseaf != "" {
		kseaf, _ := hex.DecodeString(confirmResp.Kseaf)
		if err := DeriveNASKeys(ue, kseaf); err != nil {
			return fmt.Errorf("failed to derive NAS keys: %v", err)
		}
	}

	ue.SecurityContext.IntegrityAlgorithm = AlgorithmNIA2
	ue.SecurityContext.CipheringAlgorithm = AlgorithmNEA2

	if err := h.amfContext.PersistUEContext(ue); err != nil {
		logger.NasLog.Warnf("Failed to persist UE context: %v", err)
	}

	return h.SendSecurityModeCommand(ue)
}

func (h *Handler) SendAuthenticationReject(ue *context.UEContext) error {
	logger.NasLog.Infof("Sending Authentication Reject to UE")

	pdu := &NASPDU{
		ProtocolDiscriminator: ProtocolDiscriminator5GMM,
		SecurityHeaderType:    SecurityHeaderTypePlainNAS,
		MessageType:           MsgTypeAuthenticationReject,
		Payload:               []byte{},
	}

	nasData := EncodeNASPDU(pdu)
	return h.ngapHandler.SendDownlinkNASTransport(ue.RanUeNgapId, ue.AmfUeNgapId, nasData)
}

func (h *Handler) SendSecurityModeCommand(ue *context.UEContext) error {
	logger.NasLog.Infof("Sending Security Mode Command to UE")

	nasSecAlg := uint8(ue.SecurityContext.CipheringAlgorithm<<4 | ue.SecurityContext.IntegrityAlgorithm)

	msg := &SecurityModeCommandMsg{
		SelectedNASSecurityAlgorithms: nasSecAlg,
		NgKSI:                         uint8(ue.NgKsi),
		ReplayedUESecurityCapabilities: []byte{0x00, 0x00, 0x00, 0x00},
		IMEISVRequest:                 1,
	}

	payload := EncodeSecurityModeCommand(msg)

	nasData, err := EncodeSecuredNASPDU(ue, MsgTypeSecurityModeCommand, payload,
		SecurityHeaderTypeIntegrityProtectedWithNewContext)
	if err != nil {
		return fmt.Errorf("failed to encode secured NAS PDU: %v", err)
	}

	return h.ngapHandler.SendDownlinkNASTransport(ue.RanUeNgapId, ue.AmfUeNgapId, nasData)
}

func (h *Handler) HandleSecurityModeComplete(ue *context.UEContext, payload []byte) error {
	logger.NasLog.Infof("Handle Security Mode Complete for UE SUPI: %s", ue.Supi)

	smcResp, err := DecodeSecurityModeComplete(payload)
	if err != nil {
		return fmt.Errorf("failed to decode security mode complete: %v", err)
	}

	if len(smcResp.IMEISV) > 0 {
		ue.Pei = hex.EncodeToString(smcResp.IMEISV)
		logger.NasLog.Infof("Received IMEISV: %s", ue.Pei)
	}

	ue.SecurityContext.Activated = true

	if err := h.amfContext.PersistUEContext(ue); err != nil {
		logger.NasLog.Warnf("Failed to persist UE context: %v", err)
	}

	udmClient := consumer.NewUDMClient(h.amfContext.UdmUri)
	plmnId := h.amfContext.ServedGuami[0].PlmnId.Mcc + h.amfContext.ServedGuami[0].PlmnId.Mnc

	amData, err := udmClient.GetAccessAndMobilitySubscriptionData(ue.Supi, plmnId)
	if err != nil {
		logger.NasLog.Warnf("Failed to get AM subscription data: %v", err)
	} else {
		if amData.Nssai != nil && len(amData.Nssai.DefaultSingleNssais) > 0 {
			logger.NasLog.Infof("Retrieved subscription data with %d default S-NSSAIs",
				len(amData.Nssai.DefaultSingleNssais))
		}
	}

	if err := udmClient.RegisterAMF(ue.Supi, h.amfContext.Name,
		h.amfContext.ServedGuami[0].PlmnId.Mcc); err != nil {
		logger.NasLog.Warnf("Failed to register AMF at UDM: %v", err)
	}

	return h.SendRegistrationAccept(ue)
}

func (h *Handler) SendRegistrationAccept(ue *context.UEContext) error {
	logger.NasLog.Infof("Sending Registration Accept to UE")

	msg := &RegistrationAcceptMsg{
		RegistrationResult: 0x01,
		AllowedNSSAI:      []byte{0x01, 0x01, 0x01},
		T3512Value:        []byte{0x5e, 0x01, 0x3c},
	}

	payload := EncodeRegistrationAccept(msg)

	nasData, err := EncodeSecuredNASPDU(ue, MsgTypeRegistrationAccept, payload,
		SecurityHeaderTypeIntegrityProtectedAndCiphered)
	if err != nil {
		return fmt.Errorf("failed to encode secured NAS PDU: %v", err)
	}

	ue.RegistrationState = context.RegStateRegistered
	logger.NasLog.Infof("UE %s successfully registered", ue.Supi)

	if err := h.amfContext.PersistUEContext(ue); err != nil {
		logger.NasLog.Warnf("Failed to persist UE context: %v", err)
	}

	return h.ngapHandler.SendDownlinkNASTransport(ue.RanUeNgapId, ue.AmfUeNgapId, nasData)
}

func (h *Handler) SendRegistrationReject(ue *context.UEContext, cause uint8) error {
	logger.NasLog.Infof("Sending Registration Reject to UE with cause: 0x%02x", cause)

	pdu := &NASPDU{
		ProtocolDiscriminator: ProtocolDiscriminator5GMM,
		SecurityHeaderType:    SecurityHeaderTypePlainNAS,
		MessageType:           MsgTypeRegistrationReject,
		Payload:               []byte{cause},
	}

	nasData := EncodeNASPDU(pdu)
	return h.ngapHandler.SendDownlinkNASTransport(ue.RanUeNgapId, ue.AmfUeNgapId, nasData)
}

func (h *Handler) HandleDeregistrationRequest(ue *context.UEContext, payload []byte) error {
	logger.NasLog.Infof("Handle Deregistration Request for UE SUPI: %s", ue.Supi)

	ue.RegistrationState = context.RegStateDeregistered
	ue.RmState = context.RmDeregistered

	if err := h.amfContext.PersistUEContext(ue); err != nil {
		logger.NasLog.Warnf("Failed to persist UE context: %v", err)
	}

	pdu := &NASPDU{
		ProtocolDiscriminator: ProtocolDiscriminator5GMM,
		SecurityHeaderType:    SecurityHeaderTypePlainNAS,
		MessageType:           MsgTypeDeregistrationAcceptUEOriginating,
		Payload:               []byte{},
	}

	nasData := EncodeNASPDU(pdu)
	return h.ngapHandler.SendDownlinkNASTransport(ue.RanUeNgapId, ue.AmfUeNgapId, nasData)
}

func (h *Handler) HandleServiceRequest(ue *context.UEContext, payload []byte) error {
	logger.NasLog.Infof("Handle Service Request for UE SUPI: %s", ue.Supi)

	ue.CmState = context.CmConnected

	if err := h.amfContext.PersistUEContext(ue); err != nil {
		logger.NasLog.Warnf("Failed to persist UE context: %v", err)
	}

	pdu := &NASPDU{
		ProtocolDiscriminator: ProtocolDiscriminator5GMM,
		SecurityHeaderType:    SecurityHeaderTypePlainNAS,
		MessageType:           MsgTypeServiceAccept,
		Payload:               []byte{},
	}

	nasData := EncodeNASPDU(pdu)
	return h.ngapHandler.SendDownlinkNASTransport(ue.RanUeNgapId, ue.AmfUeNgapId, nasData)
}

func (h *Handler) HandleULNASTransport(ue *context.UEContext, payload []byte) error {
	logger.NasLog.Infof("Handle UL NAS Transport for UE SUPI: %s", ue.Supi)

	ulMsg, err := DecodeULNASTransport(payload)
	if err != nil {
		return fmt.Errorf("failed to decode UL NAS transport: %v", err)
	}

	logger.NasLog.Infof("Payload Container Type: 0x%02x, PDU Session ID: %d",
		ulMsg.PayloadContainerType, ulMsg.PDUSessionID)

	if ulMsg.PayloadContainerType != PayloadContainerTypeN1SMInfo {
		logger.NasLog.Warnf("Unsupported payload container type: 0x%02x", ulMsg.PayloadContainerType)
		return nil
	}

	if len(ulMsg.PayloadContainer) < 3 {
		return fmt.Errorf("payload container too short")
	}

	smPdu := ulMsg.PayloadContainer
	smPD := (smPdu[0] >> 4) & 0x0f
	smMsgType := smPdu[2]

	logger.NasLog.Infof("SM Protocol Discriminator: 0x%x, SM Message Type: 0x%02x", smPD, smMsgType)

	if smPD != ProtocolDiscriminator5GSM {
		logger.NasLog.Warnf("Invalid SM protocol discriminator: 0x%x", smPD)
		return nil
	}

	switch smMsgType {
	case MsgTypePDUSessionEstablishmentRequest:
		return h.HandlePDUSessionEstablishmentRequest(ue, ulMsg.PDUSessionID, ulMsg.PayloadContainer, ulMsg.DNN, ulMsg.SNSSAI)
	case MsgTypePDUSessionReleaseRequest:
		logger.NasLog.Infof("PDU Session Release Request received for PDU Session ID: %d", ulMsg.PDUSessionID)
		return nil
	default:
		logger.NasLog.Warnf("Unsupported SM message type: 0x%02x", smMsgType)
		return nil
	}
}

func (h *Handler) HandlePDUSessionEstablishmentRequest(ue *context.UEContext, pduSessionID uint8, smMsg []byte, dnn []byte, snssai []byte) error {
	logger.NasLog.Infof("Handle PDU Session Establishment Request for UE SUPI: %s, PDU Session ID: %d", ue.Supi, pduSessionID)

	if len(smMsg) < 3 {
		return fmt.Errorf("SM message too short")
	}

	smPayload := smMsg[3:]
	smReq, err := DecodePDUSessionEstablishmentRequest(smPayload)
	if err != nil {
		return fmt.Errorf("failed to decode PDU session establishment request: %v", err)
	}

	logger.NasLog.Infof("PDU Session Type: %d, SSC Mode: %d", smReq.PDUSessionType, smReq.SSCMode)

	dnnStr := "internet"
	if len(dnn) > 0 {
		dnnStr = string(dnn)
	}

	smfClient := consumer.NewSMFClient(h.amfContext.SmfUri)

	createData := &consumer.SmContextCreateData{
		Supi:         ue.Supi,
		Dnn:          dnnStr,
		PduSessionId: int32(pduSessionID),
		ServingNfId:  h.amfContext.Name,
		RequestType:  "INITIAL_REQUEST",
		AnType:       "3GPP_ACCESS",
		RatType:      "NR",
	}

	if len(snssai) > 0 && len(snssai) >= 4 {
		createData.SNssai = &consumer.SNssai{
			Sst: int(snssai[1]),
		}
		if len(snssai) > 4 {
			sd := fmt.Sprintf("%02x%02x%02x", snssai[2], snssai[3], snssai[4])
			createData.SNssai.Sd = sd
		}
	}

	createResp, err := smfClient.CreateSMContext(createData)
	if err != nil {
		logger.NasLog.Errorf("Failed to create SM context: %v", err)
		return h.SendPDUSessionEstablishmentReject(ue, pduSessionID, 0x1a)
	}

	if createResp.N1SmMsg != nil {
		logger.NasLog.Infof("Received N1 SM message from SMF")
	}

	pduSessionCtx := &context.PduSessionContext{
		PduSessionId:  int32(pduSessionID),
		SmContextRef:  createResp.SmContextRef,
		SmContextId:   createResp.SmContextId,
		Dnn:           dnnStr,
		SessionAmbr:   &context.Ambr{Uplink: "100 Mbps", Downlink: "100 Mbps"},
		State:         context.PduSessionActive,
	}

	ue.PduSessions[int32(pduSessionID)] = pduSessionCtx
	logger.NasLog.Infof("PDU Session %d created for UE %s", pduSessionID, ue.Supi)

	if err := h.amfContext.PersistUEContext(ue); err != nil {
		logger.NasLog.Warnf("Failed to persist UE context: %v", err)
	}

	acceptMsg := &PDUSessionEstablishmentAcceptMsg{
		PDUSessionType: 1,
		SSCMode:        1,
		SessionAMBR:    []byte{0x3e, 0x80, 0x3e, 0x80},
		QoSRules:       []byte{0x01, 0x01, 0x01, 0x01, 0x01, 0x01},
		PDUAddress:     []byte{0x01, 192, 168, 100, 10},
	}

	if len(dnn) > 0 {
		acceptMsg.DNN = dnn
	}
	if len(snssai) > 0 {
		acceptMsg.SNSSAl = snssai
	}

	smAcceptPayload := EncodePDUSessionEstablishmentAccept(acceptMsg)

	smPDU := make([]byte, 0)
	smPDU = append(smPDU, ProtocolDiscriminator5GSM)
	smPDU = append(smPDU, pduSessionID)
	smPDU = append(smPDU, 0x00)
	smPDU = append(smPDU, MsgTypePDUSessionEstablishmentAccept)
	smPDU = append(smPDU, smAcceptPayload...)

	dlMsg := &DLNASTransportMsg{
		PayloadContainerType: PayloadContainerTypeN1SMInfo,
		PayloadContainer:     smPDU,
		PDUSessionID:         pduSessionID,
	}

	dlPayload := EncodeDLNASTransport(dlMsg)

	nasData, err := EncodeSecuredNASPDU(ue, MsgTypeDLNASTransport, dlPayload,
		SecurityHeaderTypeIntegrityProtectedAndCiphered)
	if err != nil {
		return fmt.Errorf("failed to encode secured NAS PDU: %v", err)
	}

	n2SmInfo := []byte{}
	if createResp.N2SmInfo != nil {
		logger.NasLog.Infof("N2 SM Info available from SMF")
	}

	if h.ngapHandler != nil {
		return h.ngapHandler.SendPDUSessionResourceSetupRequest(
			ue.RanUeNgapId, ue.AmfUeNgapId, pduSessionID, nasData, n2SmInfo)
	}

	return nil
}

func (h *Handler) SendPDUSessionEstablishmentReject(ue *context.UEContext, pduSessionID uint8, cause uint8) error {
	logger.NasLog.Infof("Sending PDU Session Establishment Reject to UE for PDU Session ID: %d", pduSessionID)

	rejectMsg := &PDUSessionEstablishmentRejectMsg{
		Cause5GSM: cause,
	}

	smRejectPayload := EncodePDUSessionEstablishmentReject(rejectMsg)

	smPDU := make([]byte, 0)
	smPDU = append(smPDU, ProtocolDiscriminator5GSM)
	smPDU = append(smPDU, pduSessionID)
	smPDU = append(smPDU, 0x00)
	smPDU = append(smPDU, MsgTypePDUSessionEstablishmentReject)
	smPDU = append(smPDU, smRejectPayload...)

	dlMsg := &DLNASTransportMsg{
		PayloadContainerType: PayloadContainerTypeN1SMInfo,
		PayloadContainer:     smPDU,
		PDUSessionID:         pduSessionID,
	}

	dlPayload := EncodeDLNASTransport(dlMsg)

	nasData, err := EncodeSecuredNASPDU(ue, MsgTypeDLNASTransport, dlPayload,
		SecurityHeaderTypeIntegrityProtectedAndCiphered)
	if err != nil {
		return fmt.Errorf("failed to encode secured NAS PDU: %v", err)
	}

	return h.ngapHandler.SendDownlinkNASTransport(ue.RanUeNgapId, ue.AmfUeNgapId, nasData)
}
