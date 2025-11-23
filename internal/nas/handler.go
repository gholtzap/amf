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
	case MsgTypeAuthenticationFailure:
		return h.HandleAuthenticationFailure(ue, pdu.Payload)
	case MsgTypeIdentityResponse:
		return h.HandleIdentityResponse(ue, pdu.Payload)
	case MsgTypeSecurityModeComplete:
		return h.HandleSecurityModeComplete(ue, pdu.Payload)
	case MsgTypeDeregistrationRequestUEOriginating:
		return h.HandleDeregistrationRequest(ue, pdu.Payload)
	case MsgTypeDeregistrationAcceptUETerminating:
		return h.HandleDeregistrationAccept(ue, pdu.Payload)
	case MsgTypeServiceRequest:
		return h.HandleServiceRequest(ue, pdu.Payload)
	case MsgTypeExtendedServiceRequest:
		return h.HandleExtendedServiceRequest(ue, pdu.Payload)
	case MsgTypeRegistrationComplete:
		logger.NasLog.Infof("Registration Complete received for UE: %s", ue.Supi)
		return nil
	case MsgTypeConfigurationUpdateComplete:
		return h.HandleConfigurationUpdateComplete(ue, pdu.Payload)
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

	var existingUe *context.UEContext
	isGutiRegistration := false

	if len(regReq.MobileIdentity) > 0 {
		guti, err := DecodeGutiMobileIdentity(regReq.MobileIdentity)
		if err == nil {
			logger.NasLog.Infof("Registration Request with GUTI: %+v", guti)
			isGutiRegistration = true

			if foundUe, ok := h.amfContext.GetUEContextByGuti(guti); ok {
				existingUe = foundUe
				logger.NasLog.Infof("Found existing UE context for GUTI")

				ue.Supi = existingUe.Supi
				ue.Guti = existingUe.Guti
				ue.SecurityContext = existingUe.SecurityContext
				ue.SubscriptionData = existingUe.SubscriptionData
			} else {
				logger.NasLog.Warnf("GUTI not found in AMF context")
			}
		} else {
			suci := hex.EncodeToString(regReq.MobileIdentity)
			ue.Supi = suci
			logger.NasLog.Infof("Registration Request from SUCI: %s", suci)
		}
	}

	if len(regReq.UESecurityCapability) > 0 {
		ue.UeSecurityCapability = hex.EncodeToString(regReq.UESecurityCapability)
	}

	ue.RegistrationState = context.RegStateRegistering
	ue.RmState = context.RmRegistered

	if err := h.amfContext.PersistUEContext(ue); err != nil {
		logger.NasLog.Warnf("Failed to persist UE context: %v", err)
	}

	if isGutiRegistration && existingUe != nil &&
	   (regReq.RegistrationType == RegistrationTypePeriodicUpdate ||
	    regReq.RegistrationType == RegistrationTypeMobilityUpdate) &&
	   ue.SecurityContext != nil && ue.SecurityContext.Activated {
		logger.NasLog.Infof("Skipping authentication for periodic/mobility update with existing context")
		return h.SendRegistrationAccept(ue)
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

func (h *Handler) SendIdentityRequest(ue *context.UEContext, identityType uint8) error {
	logger.NasLog.Infof("Sending Identity Request to UE for identity type: %d", identityType)

	msg := &IdentityRequestMsg{
		IdentityType: identityType,
	}

	payload := EncodeIdentityRequest(msg)

	var nasData []byte
	var err error

	if ue.SecurityContext != nil && ue.SecurityContext.Activated {
		nasData, err = EncodeSecuredNASPDU(ue, MsgTypeIdentityRequest, payload,
			SecurityHeaderTypeIntegrityProtectedAndCiphered)
		if err != nil {
			return fmt.Errorf("failed to encode secured NAS PDU: %v", err)
		}
	} else {
		pdu := &NASPDU{
			ProtocolDiscriminator: ProtocolDiscriminator5GMM,
			SecurityHeaderType:    SecurityHeaderTypePlainNAS,
			MessageType:           MsgTypeIdentityRequest,
			Payload:               payload,
		}
		nasData = EncodeNASPDU(pdu)
	}

	return h.ngapHandler.SendDownlinkNASTransport(ue.RanUeNgapId, ue.AmfUeNgapId, nasData)
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

func (h *Handler) HandleIdentityResponse(ue *context.UEContext, payload []byte) error {
	logger.NasLog.Infof("Handle Identity Response for UE")

	idResp, err := DecodeIdentityResponse(payload)
	if err != nil {
		return fmt.Errorf("failed to decode identity response: %v", err)
	}

	if len(idResp.MobileIdentity) == 0 {
		return fmt.Errorf("no mobile identity in identity response")
	}

	identityType := idResp.MobileIdentity[0] & 0x07

	switch identityType {
	case 0x01:
		ue.Supi = hex.EncodeToString(idResp.MobileIdentity)
		logger.NasLog.Infof("Received SUPI: %s", ue.Supi)
	case 0x02:
		guti, err := DecodeGutiMobileIdentity(idResp.MobileIdentity)
		if err == nil {
			ue.Guti = guti
			logger.NasLog.Infof("Received GUTI: %+v", guti)
		} else {
			logger.NasLog.Warnf("Failed to decode GUTI: %v", err)
		}
	case 0x03:
		ue.Pei = hex.EncodeToString(idResp.MobileIdentity)
		logger.NasLog.Infof("Received IMEI: %s", ue.Pei)
	case 0x04:
		ue.Pei = hex.EncodeToString(idResp.MobileIdentity)
		logger.NasLog.Infof("Received IMEISV: %s", ue.Pei)
	default:
		logger.NasLog.Warnf("Unknown identity type: 0x%02x", identityType)
	}

	if err := h.amfContext.PersistUEContext(ue); err != nil {
		logger.NasLog.Warnf("Failed to persist UE context: %v", err)
	}

	return nil
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
		if ue.IsReAuthenticating && ue.PreviousSecurityContext != nil {
			logger.NasLog.Infof("Restoring previous security context due to re-authentication failure")
			ue.SecurityContext = ue.PreviousSecurityContext
			ue.IsReAuthenticating = false
			ue.PreviousSecurityContext = nil
		}
		return h.SendAuthenticationReject(ue)
	}

	if confirmResp.AuthResult != "AUTHENTICATION_SUCCESS" {
		logger.NasLog.Errorf("Authentication failed: %s", confirmResp.AuthResult)
		if ue.IsReAuthenticating && ue.PreviousSecurityContext != nil {
			logger.NasLog.Infof("Restoring previous security context due to re-authentication failure")
			ue.SecurityContext = ue.PreviousSecurityContext
			ue.IsReAuthenticating = false
			ue.PreviousSecurityContext = nil
		}
		return h.SendAuthenticationReject(ue)
	}

	if confirmResp.Kseaf != "" {
		kseaf, _ := hex.DecodeString(confirmResp.Kseaf)
		if err := DeriveNASKeys(ue, kseaf); err != nil {
			if ue.IsReAuthenticating && ue.PreviousSecurityContext != nil {
				logger.NasLog.Infof("Restoring previous security context due to key derivation failure")
				ue.SecurityContext = ue.PreviousSecurityContext
				ue.IsReAuthenticating = false
				ue.PreviousSecurityContext = nil
			}
			return fmt.Errorf("failed to derive NAS keys: %v", err)
		}
	}

	ue.SecurityContext.IntegrityAlgorithm = AlgorithmNIA2
	ue.SecurityContext.CipheringAlgorithm = AlgorithmNEA2

	if ue.IsReAuthenticating {
		logger.NasLog.Infof("Re-authentication successful for UE: %s", ue.Supi)
		ue.IsReAuthenticating = false
		ue.PreviousSecurityContext = nil
		ue.ULCount = 0
		ue.DLCount = 0

		if err := h.amfContext.PersistUEContext(ue); err != nil {
			logger.NasLog.Warnf("Failed to persist UE context: %v", err)
		}

		return h.SendSecurityModeCommand(ue)
	}

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

func (h *Handler) HandleAuthenticationFailure(ue *context.UEContext, payload []byte) error {
	logger.NasLog.Infof("Handle Authentication Failure for UE SUPI: %s", ue.Supi)

	authFail, err := DecodeAuthenticationFailure(payload)
	if err != nil {
		return fmt.Errorf("failed to decode authentication failure: %v", err)
	}

	logger.NasLog.Warnf("UE authentication failed with cause: 0x%02x", authFail.Cause5GMM)

	if authFail.Cause5GMM == CauseSynchFailure && len(authFail.AuthenticationFailureParameter) > 0 {
		logger.NasLog.Infof("Synchronization failure detected, AUTS received (length: %d)", len(authFail.AuthenticationFailureParameter))
	}

	return h.SendAuthenticationReject(ue)
}

func (h *Handler) InitiateReAuthentication(ue *context.UEContext) error {
	logger.NasLog.Infof("Initiating re-authentication for UE SUPI: %s", ue.Supi)

	if ue.RmState != context.RmRegistered {
		return fmt.Errorf("UE not in RM-REGISTERED state, cannot re-authenticate")
	}

	if ue.SecurityContext != nil && ue.SecurityContext.Activated {
		ue.PreviousSecurityContext = &context.SecurityContext{
			Kseaf:              make([]byte, len(ue.SecurityContext.Kseaf)),
			Kamf:               make([]byte, len(ue.SecurityContext.Kamf)),
			KnasInt:            make([]byte, len(ue.SecurityContext.KnasInt)),
			KnasEnc:            make([]byte, len(ue.SecurityContext.KnasEnc)),
			NgKsi:              ue.SecurityContext.NgKsi,
			IntegrityAlg:       ue.SecurityContext.IntegrityAlg,
			CipheringAlg:       ue.SecurityContext.CipheringAlg,
			IntegrityAlgorithm: ue.SecurityContext.IntegrityAlgorithm,
			CipheringAlgorithm: ue.SecurityContext.CipheringAlgorithm,
			Activated:          ue.SecurityContext.Activated,
		}
		copy(ue.PreviousSecurityContext.Kseaf, ue.SecurityContext.Kseaf)
		copy(ue.PreviousSecurityContext.Kamf, ue.SecurityContext.Kamf)
		copy(ue.PreviousSecurityContext.KnasInt, ue.SecurityContext.KnasInt)
		copy(ue.PreviousSecurityContext.KnasEnc, ue.SecurityContext.KnasEnc)
		logger.NasLog.Infof("Preserved existing security context for UE: %s", ue.Supi)
	}

	ue.IsReAuthenticating = true
	ue.NgKsi = (ue.NgKsi + 1) % 7

	ausfClient := consumer.NewAUSFClient(h.amfContext.AusfUri)
	servingNetwork := fmt.Sprintf("5G:mnc%s.mcc%s.3gppnetwork.org",
		h.amfContext.ServedGuami[0].PlmnId.Mnc,
		h.amfContext.ServedGuami[0].PlmnId.Mcc)

	authResp, err := ausfClient.RequestAuthentication(ue.Supi, servingNetwork)
	if err != nil {
		logger.NasLog.Errorf("Re-authentication request failed: %v", err)
		ue.IsReAuthenticating = false
		return fmt.Errorf("re-authentication request failed: %v", err)
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

		if err := h.amfContext.PersistUEContext(ue); err != nil {
			logger.NasLog.Warnf("Failed to persist UE context: %v", err)
		}

		return h.SendAuthenticationRequest(ue, rand, autn)
	}

	ue.IsReAuthenticating = false
	return fmt.Errorf("no authentication vector in response")
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

	if ue.RegistrationState == context.RegStateRegistered {
		logger.NasLog.Infof("Re-authentication completed successfully for UE: %s", ue.Supi)
		return nil
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

	if ue.Guti == nil {
		ue.Guti = h.amfContext.AllocateGuti()
		logger.NasLog.Infof("Allocated GUTI for UE: %+v", ue.Guti)
	}

	msg := &RegistrationAcceptMsg{
		RegistrationResult: 0x01,
		MobileIdentity:    EncodeGutiMobileIdentity(ue.Guti),
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

	srvReq, err := DecodeServiceRequest(payload)
	if err != nil {
		logger.NasLog.Errorf("Failed to decode service request: %v", err)
		return h.SendServiceReject(ue, CauseProtocolError)
	}

	logger.NasLog.Infof("Service Type: %d, NgKSI: %d", srvReq.ServiceType, srvReq.NgKSI)

	if ue.RmState != context.RmRegistered {
		logger.NasLog.Errorf("UE not in RM-REGISTERED state: %s", ue.RmState)
		return h.SendServiceReject(ue, Cause5GSServicesNotAllowed)
	}

	if ue.CmState != context.CmIdle {
		logger.NasLog.Warnf("UE not in CM-IDLE state: %s", ue.CmState)
	}

	if ue.SecurityContext == nil || !ue.SecurityContext.Activated {
		logger.NasLog.Errorf("Security context not activated for UE: %s", ue.Supi)
		return h.SendServiceReject(ue, CauseSecurityModeRejectedUnspecified)
	}

	if int(srvReq.NgKSI) != ue.NgKsi {
		logger.NasLog.Warnf("NgKSI mismatch - Request: %d, Expected: %d", srvReq.NgKSI, ue.NgKsi)
	}

	ue.CmState = context.CmConnected

	activePduSessions := []uint8{}
	if len(srvReq.PDUSessionStatus) > 0 {
		logger.NasLog.Infof("PDU Session Status present in Service Request")
		for i := 0; i < len(srvReq.PDUSessionStatus) && i < 16; i++ {
			statusByte := srvReq.PDUSessionStatus[i]
			for bit := 0; bit < 8; bit++ {
				if statusByte&(1<<bit) != 0 {
					pduSessionId := uint8(i*8 + bit)
					if pduSession, ok := ue.GetPduSession(int32(pduSessionId)); ok {
						if pduSession.State == context.PduSessionActive {
							activePduSessions = append(activePduSessions, pduSessionId)
							logger.NasLog.Infof("PDU Session %d is active", pduSessionId)
						}
					}
				}
			}
		}
	} else {
		for pduSessionId, pduSession := range ue.PduSessions {
			if pduSession.State == context.PduSessionActive {
				activePduSessions = append(activePduSessions, uint8(pduSessionId))
			}
		}
	}

	if err := h.amfContext.PersistUEContext(ue); err != nil {
		logger.NasLog.Warnf("Failed to persist UE context: %v", err)
	}

	return h.SendServiceAccept(ue, activePduSessions)
}

func (h *Handler) SendServiceAccept(ue *context.UEContext, activePduSessions []uint8) error {
	logger.NasLog.Infof("Sending Service Accept to UE SUPI: %s", ue.Supi)

	msg := &ServiceAcceptMsg{}

	if len(activePduSessions) > 0 {
		pduSessionStatusBytes := make([]byte, 2)
		for _, pduSessionId := range activePduSessions {
			byteIndex := pduSessionId / 8
			bitIndex := pduSessionId % 8
			if int(byteIndex) < len(pduSessionStatusBytes) {
				pduSessionStatusBytes[byteIndex] |= (1 << bitIndex)
			}
		}
		msg.PDUSessionStatus = pduSessionStatusBytes
		logger.NasLog.Infof("PDU Session Status in Service Accept: %d sessions active", len(activePduSessions))
	}

	acceptPayload := EncodeServiceAccept(msg)

	nasData, err := EncodeSecuredNASPDU(ue, MsgTypeServiceAccept, acceptPayload,
		SecurityHeaderTypeIntegrityProtectedAndCiphered)
	if err != nil {
		return fmt.Errorf("failed to encode secured NAS PDU: %v", err)
	}

	return h.ngapHandler.SendDownlinkNASTransport(ue.RanUeNgapId, ue.AmfUeNgapId, nasData)
}

func (h *Handler) SendServiceReject(ue *context.UEContext, cause uint8) error {
	logger.NasLog.Infof("Sending Service Reject to UE SUPI: %s with cause: 0x%02x", ue.Supi, cause)

	msg := &ServiceRejectMsg{
		Cause5GMM: cause,
	}

	rejectPayload := EncodeServiceReject(msg)

	nasData, err := EncodeSecuredNASPDU(ue, MsgTypeServiceReject, rejectPayload,
		SecurityHeaderTypeIntegrityProtectedAndCiphered)
	if err != nil {
		return fmt.Errorf("failed to encode secured NAS PDU: %v", err)
	}

	return h.ngapHandler.SendDownlinkNASTransport(ue.RanUeNgapId, ue.AmfUeNgapId, nasData)
}

func (h *Handler) HandleExtendedServiceRequest(ue *context.UEContext, payload []byte) error {
	logger.NasLog.Infof("Handle Extended Service Request for UE SUPI: %s", ue.Supi)

	srvReq, err := DecodeServiceRequest(payload)
	if err != nil {
		logger.NasLog.Errorf("Failed to decode extended service request: %v", err)
		return h.SendServiceReject(ue, CauseProtocolError)
	}

	logger.NasLog.Infof("Extended Service Type: %d, NgKSI: %d", srvReq.ServiceType, srvReq.NgKSI)

	if ue.RmState != context.RmRegistered {
		logger.NasLog.Errorf("UE not in RM-REGISTERED state: %s", ue.RmState)
		return h.SendServiceReject(ue, Cause5GSServicesNotAllowed)
	}

	if ue.CmState != context.CmIdle {
		logger.NasLog.Warnf("UE not in CM-IDLE state: %s", ue.CmState)
	}

	if ue.SecurityContext == nil || !ue.SecurityContext.Activated {
		logger.NasLog.Errorf("Security context not activated for UE: %s", ue.Supi)
		return h.SendServiceReject(ue, CauseSecurityModeRejectedUnspecified)
	}

	if int(srvReq.NgKSI) != ue.NgKsi {
		logger.NasLog.Warnf("NgKSI mismatch - Request: %d, Expected: %d", srvReq.NgKSI, ue.NgKsi)
	}

	ue.CmState = context.CmConnected

	activePduSessions := []uint8{}
	if len(srvReq.PDUSessionStatus) > 0 {
		logger.NasLog.Infof("PDU Session Status present in Extended Service Request")
		for i := 0; i < len(srvReq.PDUSessionStatus) && i < 16; i++ {
			statusByte := srvReq.PDUSessionStatus[i]
			for bit := 0; bit < 8; bit++ {
				if statusByte&(1<<bit) != 0 {
					pduSessionId := uint8(i*8 + bit)
					if pduSession, ok := ue.GetPduSession(int32(pduSessionId)); ok {
						if pduSession.State == context.PduSessionActive {
							activePduSessions = append(activePduSessions, pduSessionId)
							logger.NasLog.Infof("PDU Session %d is active", pduSessionId)
						}
					}
				}
			}
		}
	} else {
		for pduSessionId, pduSession := range ue.PduSessions {
			if pduSession.State == context.PduSessionActive {
				activePduSessions = append(activePduSessions, uint8(pduSessionId))
			}
		}
	}

	if err := h.amfContext.PersistUEContext(ue); err != nil {
		logger.NasLog.Warnf("Failed to persist UE context: %v", err)
	}

	return h.SendServiceAccept(ue, activePduSessions)
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
	case MsgTypePDUSessionModificationRequest:
		return h.HandlePDUSessionModificationRequest(ue, ulMsg.PDUSessionID, ulMsg.PayloadContainer)
	case MsgTypePDUSessionReleaseRequest:
		return h.HandlePDUSessionReleaseRequest(ue, ulMsg.PDUSessionID, ulMsg.PayloadContainer)
	case MsgTypeFiveGSMStatus:
		return h.HandleFiveGSMStatus(ue, ulMsg.PDUSessionID, ulMsg.PayloadContainer)
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

	alwaysOnRequested := smReq.AlwaysOnPDUSessionRequested == 1
	logger.NasLog.Infof("PDU Session Type: %d, SSC Mode: %d, AlwaysOn Requested: %v", smReq.PDUSessionType, smReq.SSCMode, alwaysOnRequested)

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
		AlwaysOn:      alwaysOnRequested,
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

	if alwaysOnRequested {
		acceptMsg.AlwaysOnPDUSessionIndication = 1
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

func (h *Handler) HandlePDUSessionModificationRequest(ue *context.UEContext, pduSessionID uint8, smMsg []byte) error {
	logger.NasLog.Infof("Handle PDU Session Modification Request for UE SUPI: %s, PDU Session ID: %d", ue.Supi, pduSessionID)

	if len(smMsg) < 3 {
		return fmt.Errorf("SM message too short")
	}

	smPayload := smMsg[3:]
	smReq, err := DecodePDUSessionModificationRequest(smPayload)
	if err != nil {
		return fmt.Errorf("failed to decode PDU session modification request: %v", err)
	}

	pduSession, ok := ue.GetPduSession(int32(pduSessionID))
	if !ok {
		logger.NasLog.Errorf("PDU Session %d not found for UE %s", pduSessionID, ue.Supi)
		return h.SendPDUSessionModificationReject(ue, pduSessionID, 0x1a)
	}

	smfClient := consumer.NewSMFClient(h.amfContext.SmfUri)

	updateData := &consumer.SmContextUpdateData{}

	if len(smReq.RequestedQoSRules) > 0 {
		logger.NasLog.Infof("QoS Rules modification requested")
	}

	if len(smReq.RequestedQoSFlowDescriptions) > 0 {
		logger.NasLog.Infof("QoS Flow Descriptions modification requested")
	}

	updateResp, err := smfClient.UpdateSMContext(pduSession.SmContextRef, updateData)
	if err != nil {
		logger.NasLog.Errorf("Failed to update SM context: %v", err)
		return h.SendPDUSessionModificationReject(ue, pduSessionID, 0x1a)
	}

	if updateResp.N1SmMsg != nil {
		logger.NasLog.Infof("Received N1 SM message from SMF")
	}

	if err := h.amfContext.PersistUEContext(ue); err != nil {
		logger.NasLog.Warnf("Failed to persist UE context: %v", err)
	}

	commandMsg := &PDUSessionModificationCommandMsg{
		QoSRules:    []byte{0x01, 0x01, 0x01, 0x01, 0x01, 0x01},
		SessionAMBR: []byte{0x3e, 0x80, 0x3e, 0x80},
	}

	smCommandPayload := EncodePDUSessionModificationCommand(commandMsg)

	smPDU := make([]byte, 0)
	smPDU = append(smPDU, ProtocolDiscriminator5GSM)
	smPDU = append(smPDU, pduSessionID)
	smPDU = append(smPDU, 0x00)
	smPDU = append(smPDU, MsgTypePDUSessionModificationCommand)
	smPDU = append(smPDU, smCommandPayload...)

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

	logger.NasLog.Infof("PDU Session %d modification completed for UE %s", pduSessionID, ue.Supi)

	return h.ngapHandler.SendDownlinkNASTransport(ue.RanUeNgapId, ue.AmfUeNgapId, nasData)
}

func (h *Handler) SendPDUSessionModificationReject(ue *context.UEContext, pduSessionID uint8, cause uint8) error {
	logger.NasLog.Infof("Sending PDU Session Modification Reject to UE for PDU Session ID: %d", pduSessionID)

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

func (h *Handler) HandlePDUSessionReleaseRequest(ue *context.UEContext, pduSessionID uint8, smMsg []byte) error {
	logger.NasLog.Infof("Handle PDU Session Release Request for UE SUPI: %s, PDU Session ID: %d", ue.Supi, pduSessionID)

	if len(smMsg) < 3 {
		return fmt.Errorf("SM message too short")
	}

	smPayload := smMsg[3:]
	smReq, err := DecodePDUSessionReleaseRequest(smPayload)
	if err != nil {
		return fmt.Errorf("failed to decode PDU session release request: %v", err)
	}

	if smReq.Cause5GSM > 0 {
		logger.NasLog.Infof("UE provided release cause: 0x%02x", smReq.Cause5GSM)
	}

	pduSession, ok := ue.GetPduSession(int32(pduSessionID))
	if !ok {
		logger.NasLog.Errorf("PDU Session %d not found for UE %s", pduSessionID, ue.Supi)
		return nil
	}

	smfClient := consumer.NewSMFClient(h.amfContext.SmfUri)

	releaseData := &consumer.SmContextReleaseData{
		Cause: "REL_DUE_TO_UE_REQUEST",
	}

	_, err = smfClient.ReleaseSMContext(pduSession.SmContextRef, releaseData)
	if err != nil {
		logger.NasLog.Warnf("Failed to release SM context: %v", err)
	}

	if !ue.DeletePduSession(int32(pduSessionID)) {
		logger.NasLog.Warnf("Failed to delete PDU Session %d from UE context", pduSessionID)
	}

	logger.NasLog.Infof("PDU Session %d released for UE %s", pduSessionID, ue.Supi)

	if err := h.amfContext.PersistUEContext(ue); err != nil {
		logger.NasLog.Warnf("Failed to persist UE context: %v", err)
	}

	completeMsg := &PDUSessionReleaseCompleteMsg{}

	smCompletePayload := EncodePDUSessionReleaseComplete(completeMsg)

	smPDU := make([]byte, 0)
	smPDU = append(smPDU, ProtocolDiscriminator5GSM)
	smPDU = append(smPDU, pduSessionID)
	smPDU = append(smPDU, 0x00)
	smPDU = append(smPDU, MsgTypePDUSessionReleaseComplete)
	smPDU = append(smPDU, smCompletePayload...)

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

func (h *Handler) SendConfigurationUpdateCommand(ue *context.UEContext, newGuti *context.Guti) error {
	logger.NasLog.Infof("Sending Configuration Update Command to UE")

	if newGuti == nil {
		newGuti = h.amfContext.AllocateGuti()
	}

	msg := &ConfigurationUpdateCommandMsg{
		ConfigurationUpdateIndication: 0x01,
		Guti:                          EncodeGutiMobileIdentity(newGuti),
	}

	payload := EncodeConfigurationUpdateCommand(msg)

	nasData, err := EncodeSecuredNASPDU(ue, MsgTypeConfigurationUpdateCommand, payload,
		SecurityHeaderTypeIntegrityProtectedAndCiphered)
	if err != nil {
		return fmt.Errorf("failed to encode secured NAS PDU: %v", err)
	}

	ue.Guti = newGuti
	logger.NasLog.Infof("Allocated new GUTI for UE: %+v", newGuti)

	if err := h.amfContext.PersistUEContext(ue); err != nil {
		logger.NasLog.Warnf("Failed to persist UE context: %v", err)
	}

	return h.ngapHandler.SendDownlinkNASTransport(ue.RanUeNgapId, ue.AmfUeNgapId, nasData)
}

func (h *Handler) HandleConfigurationUpdateComplete(ue *context.UEContext, payload []byte) error {
	logger.NasLog.Infof("Handle Configuration Update Complete for UE SUPI: %s", ue.Supi)

	_, err := DecodeConfigurationUpdateComplete(payload)
	if err != nil {
		logger.NasLog.Errorf("Failed to decode Configuration Update Complete: %v", err)
		return err
	}

	logger.NasLog.Infof("Configuration Update Complete processed successfully for UE: %s", ue.Supi)

	if err := h.amfContext.PersistUEContext(ue); err != nil {
		logger.NasLog.Warnf("Failed to persist UE context: %v", err)
	}

	return nil
}

func (h *Handler) SendDeregistrationRequest(ue *context.UEContext, deregType uint8, cause uint8, reregistrationRequired bool) error {
	logger.NasLog.Infof("Sending Network-Initiated Deregistration Request to UE SUPI: %s", ue.Supi)

	deregTypeValue := deregType
	if reregistrationRequired {
		deregTypeValue |= DeregistrationReRegistrationRequired
		logger.NasLog.Infof("Re-registration required flag set for deregistration")
	}

	msg := &DeregistrationRequestMsg{
		DeregistrationType: deregTypeValue,
		Cause5GMM:          cause,
	}

	payload := EncodeDeregistrationRequest(msg)

	var nasData []byte
	var err error

	if ue.SecurityContext != nil && ue.SecurityContext.Activated {
		nasData, err = EncodeSecuredNASPDU(ue, MsgTypeDeregistrationRequestUETerminating, payload,
			SecurityHeaderTypeIntegrityProtectedAndCiphered)
		if err != nil {
			return fmt.Errorf("failed to encode secured NAS PDU: %v", err)
		}
	} else {
		pdu := &NASPDU{
			ProtocolDiscriminator: ProtocolDiscriminator5GMM,
			SecurityHeaderType:    SecurityHeaderTypePlainNAS,
			MessageType:           MsgTypeDeregistrationRequestUETerminating,
			Payload:               payload,
		}
		nasData = EncodeNASPDU(pdu)
	}

	ue.RegistrationState = context.RegStateDeregistered
	ue.RmState = context.RmDeregistered

	if err := h.amfContext.PersistUEContext(ue); err != nil {
		logger.NasLog.Warnf("Failed to persist UE context: %v", err)
	}

	return h.ngapHandler.SendDownlinkNASTransport(ue.RanUeNgapId, ue.AmfUeNgapId, nasData)
}

func (h *Handler) HandleDeregistrationAccept(ue *context.UEContext, payload []byte) error {
	logger.NasLog.Infof("Handle Deregistration Accept (UE Terminating) for UE SUPI: %s", ue.Supi)

	logger.NasLog.Infof("Network-initiated deregistration completed successfully for UE: %s", ue.Supi)

	ue.RegistrationState = context.RegStateDeregistered
	ue.RmState = context.RmDeregistered

	if err := h.amfContext.PersistUEContext(ue); err != nil {
		logger.NasLog.Warnf("Failed to persist UE context: %v", err)
	}

	h.amfContext.DeleteUEContext(ue.AmfUeNgapId)
	logger.NasLog.Infof("UE context deleted for AMF UE NGAP ID: %d", ue.AmfUeNgapId)

	return nil
}

func (h *Handler) HandleFiveGSMStatus(ue *context.UEContext, pduSessionID uint8, smMsg []byte) error {
	logger.NasLog.Infof("Handle 5GSM Status for UE SUPI: %s, PDU Session ID: %d", ue.Supi, pduSessionID)

	if len(smMsg) < 3 {
		return fmt.Errorf("SM message too short")
	}

	smPayload := smMsg[3:]
	statusMsg, err := DecodeFiveGSMStatus(smPayload)
	if err != nil {
		return fmt.Errorf("failed to decode 5GSM status: %v", err)
	}

	logger.NasLog.Infof("Received 5GSM Status from UE %s for PDU Session %d with cause: 0x%02x",
		ue.Supi, pduSessionID, statusMsg.Cause5GSM)

	_, ok := ue.GetPduSession(int32(pduSessionID))
	if !ok {
		logger.NasLog.Warnf("PDU Session %d not found for UE %s", pduSessionID, ue.Supi)
	}

	return nil
}
