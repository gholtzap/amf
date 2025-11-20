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

	pdu := &NASPDU{
		ProtocolDiscriminator: ProtocolDiscriminator5GMM,
		SecurityHeaderType:    SecurityHeaderTypePlainNAS,
		MessageType:           MsgTypeServiceAccept,
		Payload:               []byte{},
	}

	nasData := EncodeNASPDU(pdu)
	return h.ngapHandler.SendDownlinkNASTransport(ue.RanUeNgapId, ue.AmfUeNgapId, nasData)
}
