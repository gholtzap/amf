package nas

import (
	"encoding/hex"
	"fmt"
	"time"

	"github.com/gavin/amf/internal/context"
	"github.com/gavin/amf/internal/consumer"
	"github.com/gavin/amf/internal/logger"
	"github.com/gavin/amf/pkg/factory"
)

type Handler struct {
	amfContext *context.AMFContext
	ngapHandler NGAPHandler
}

type NGAPHandler interface {
	SendDownlinkNASTransport(ranUeNgapId, amfUeNgapId int64, nasPDU []byte) error
	SendPDUSessionResourceSetupRequest(ranUeNgapId, amfUeNgapId int64, pduSessionId uint8, nasPDU []byte, n2SmInfo []byte) error
	NotifyCommunicationFailure(ue *context.UEContext, failureType string)
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
	case MsgTypeControlPlaneServiceRequest:
		return h.HandleControlPlaneServiceRequest(ue, pdu.Payload)
	case MsgTypeExtendedServiceRequest:
		return h.HandleExtendedServiceRequest(ue, pdu.Payload)
	case MsgTypeRegistrationComplete:
		logger.NasLog.Infof("Registration Complete received for UE: %s", ue.Supi)
		ue.StopT3550()
		h.startT3512(ue)
		return nil
	case MsgTypeConfigurationUpdateComplete:
		return h.HandleConfigurationUpdateComplete(ue, pdu.Payload)
	case MsgTypeGenericUEConfigurationUpdateComplete:
		return h.HandleGenericUEConfigurationUpdateComplete(ue, pdu.Payload)
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
	ue.IsEmergencyRegistration = (regReq.RegistrationType == RegistrationTypeEmergency)
	ue.MicoMode = regReq.MicoIndication

	if ue.MicoMode {
		logger.NasLog.Infof("UE requested MICO mode")
	}

	if len(regReq.RequestedEDrxParameters) > 0 {
		eDrxParams := &context.EDrxParameters{
			Enabled:          true,
			EDrxValue:        regReq.RequestedEDrxParameters[0],
			PagingTimeWindow: 0,
		}
		if len(regReq.RequestedEDrxParameters) > 1 {
			eDrxParams.PagingTimeWindow = regReq.RequestedEDrxParameters[1]
		}
		ue.EDrxParameters = eDrxParams
		logger.NasLog.Infof("UE requested eDRX: value=0x%02x, PTW=0x%02x",
			eDrxParams.EDrxValue, eDrxParams.PagingTimeWindow)
	}

	if len(regReq.RequestedT3324Value) > 0 || len(regReq.RequestedT3412ExtendedValue) > 0 {
		psmParams := &context.PSMParameters{
			Enabled: true,
		}
		if len(regReq.RequestedT3324Value) > 0 {
			psmParams.T3324Value = regReq.RequestedT3324Value[0]
		}
		if len(regReq.RequestedT3412ExtendedValue) > 0 {
			psmParams.T3412ExtendedValue = regReq.RequestedT3412ExtendedValue[0]
		}
		ue.PSMParameters = psmParams
		logger.NasLog.Infof("UE requested PSM: T3324=0x%02x, T3412ext=0x%02x",
			psmParams.T3324Value, psmParams.T3412ExtendedValue)
	}

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
		if ue.SecurityContext == nil {
			ue.SecurityContext = &context.SecurityContext{}
		}
		ue.SecurityContext.SecurityCapability = ParseUESecurityCapabilities(regReq.UESecurityCapability)
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

		if regReq.RegistrationType == RegistrationTypeMobilityUpdate {
			logger.NasLog.Infof("Processing Mobility Registration Update (Tracking Area Update) for UE %s", ue.Supi)
			logger.NasLog.Infof("Current TAI: %+v", ue.Tai)

			if ue.IsTaiForbidden(ue.Tai) {
				logger.NasLog.Warnf("UE in forbidden TAI: %+v - rejecting registration", ue.Tai)
				return h.SendRegistrationReject(ue, CauseTrackingAreaNotAllowed)
			}

			if ue.IsServiceAreaRestricted(ue.Tai) {
				logger.NasLog.Warnf("UE in restricted service area: TAC=%s - rejecting mobility update", ue.Tai.Tac)
				return h.SendRegistrationReject(ue, CauseTrackingAreaNotAllowed)
			}

			if !ue.IsTaiInList(ue.Tai) {
				logger.NasLog.Infof("UE moved to new TAI outside allowed list - will provide updated TAI list")
			} else {
				logger.NasLog.Infof("UE TAI is in allowed list - confirming registration")
			}
		} else {
			logger.NasLog.Infof("Processing Periodic Registration Update for UE %s", ue.Supi)
		}

		logger.NasLog.Infof("Skipping authentication for periodic/mobility update with existing context")
		return h.SendRegistrationAccept(ue)
	}

	if ue.IsEmergencyRegistration {
		logger.NasLog.Infof("Emergency registration detected for UE")
		if ue.Supi == "" {
			ue.Supi = "emergency-" + fmt.Sprintf("%d", ue.AmfUeNgapId)
			logger.NasLog.Infof("Assigned emergency SUPI: %s", ue.Supi)
		}
		return h.SendEmergencyRegistrationAccept(ue)
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

	ue.RequestedIdentityType = identityType

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

	h.startT3570(ue, identityType)

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

	h.startT3560(ue, rand, autn)

	return h.ngapHandler.SendDownlinkNASTransport(ue.RanUeNgapId, ue.AmfUeNgapId, nasData)
}

func (h *Handler) HandleIdentityResponse(ue *context.UEContext, payload []byte) error {
	logger.NasLog.Infof("Handle Identity Response for UE")

	ue.StopT3570()

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

	ue.StopT3560()

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

	amfSupportedEA := []int{AlgorithmNEA0, AlgorithmNEA2}
	amfSupportedIA := []int{AlgorithmNIA0, AlgorithmNIA2}

	if ue.SecurityContext.SecurityCapability != nil {
		cipheringAlg, integrityAlg := SelectSecurityAlgorithms(
			ue.SecurityContext.SecurityCapability,
			amfSupportedEA,
			amfSupportedIA,
		)
		ue.SecurityContext.CipheringAlgorithm = cipheringAlg
		ue.SecurityContext.IntegrityAlgorithm = integrityAlg
		logger.NasLog.Infof("Selected algorithms - Ciphering: NEA%d, Integrity: NIA%d",
			cipheringAlg, integrityAlg)
	} else {
		ue.SecurityContext.IntegrityAlgorithm = AlgorithmNIA2
		ue.SecurityContext.CipheringAlgorithm = AlgorithmNEA2
		logger.NasLog.Warnf("No UE security capabilities, using default NIA2/NEA2")
	}

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

	h.startT3565(ue)

	return h.ngapHandler.SendDownlinkNASTransport(ue.RanUeNgapId, ue.AmfUeNgapId, nasData)
}

func (h *Handler) HandleSecurityModeComplete(ue *context.UEContext, payload []byte) error {
	logger.NasLog.Infof("Handle Security Mode Complete for UE SUPI: %s", ue.Supi)

	ue.StopT3565()

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

		if amData.ServiceAreaRestriction != nil {
			ue.ServiceAreaRestriction = &context.ServiceAreaRestriction{
				RestrictionType: amData.ServiceAreaRestriction.RestrictionType,
				MaxNumOfTAs:     amData.ServiceAreaRestriction.MaxNumOfTAs,
			}
			for _, area := range amData.ServiceAreaRestriction.Areas {
				ue.ServiceAreaRestriction.Areas = append(ue.ServiceAreaRestriction.Areas,
					context.AreaRestriction{Tacs: area.Tacs})
			}
			logger.NasLog.Infof("Service area restriction applied: %s", ue.ServiceAreaRestriction.RestrictionType)
		}

		if len(amData.ForbiddenAreas) > 0 {
			ue.ForbiddenTaiList = make([]context.Tai, 0)
			for _, area := range amData.ForbiddenAreas {
				for _, tac := range area.Tacs {
					forbiddenTai := context.Tai{
						PlmnId: h.amfContext.ServedGuami[0].PlmnId,
						Tac:    tac,
					}
					ue.ForbiddenTaiList = append(ue.ForbiddenTaiList, forbiddenTai)
				}
			}
			logger.NasLog.Infof("Forbidden areas configured: %d TACs", len(ue.ForbiddenTaiList))
		}
	}

	if err := udmClient.RegisterAMF(ue.Supi, h.amfContext.Name,
		h.amfContext.ServedGuami[0].PlmnId.Mcc); err != nil {
		logger.NasLog.Warnf("Failed to register AMF at UDM: %v", err)
	}

	return h.SendRegistrationAccept(ue)
}

func (h *Handler) buildTAIListForUE(ue *context.UEContext) []context.Tai {
	taiList := make([]context.Tai, 0)

	if ue.RanContext == nil {
		return taiList
	}

	for _, supportedTai := range ue.RanContext.SupportedTAList {
		taiList = append(taiList, supportedTai.Tai)
	}

	if len(taiList) == 0 && ue.Tai.Tac != "" {
		taiList = append(taiList, ue.Tai)
	}

	if len(taiList) > 16 {
		taiList = taiList[:16]
	}

	return taiList
}

func (h *Handler) SendRegistrationAccept(ue *context.UEContext) error {
	logger.NasLog.Infof("Sending Registration Accept to UE")

	if ue.Guti == nil {
		ue.Guti = h.amfContext.AllocateGuti()
		logger.NasLog.Infof("Allocated GUTI for UE: %+v", ue.Guti)
	}

	if ue.IsTaiForbidden(ue.Tai) {
		logger.NasLog.Warnf("UE in forbidden TAI: %+v - rejecting registration", ue.Tai)
		return h.SendRegistrationReject(ue, CauseTrackingAreaNotAllowed)
	}

	if ue.IsServiceAreaRestricted(ue.Tai) {
		logger.NasLog.Warnf("UE in restricted service area: TAC=%s - rejecting registration", ue.Tai.Tac)
		return h.SendRegistrationReject(ue, CauseTrackingAreaNotAllowed)
	}

	taiList := h.buildTAIListForUE(ue)
	if len(taiList) > 0 {
		ue.TaiList = taiList
		logger.NasLog.Infof("Assigned TAI list with %d TAIs to UE", len(taiList))
	}

	msg := &RegistrationAcceptMsg{
		RegistrationResult: 0x01,
		MobileIdentity:    EncodeGutiMobileIdentity(ue.Guti),
		TAIList:           EncodeServiceAreaList(taiList),
		AllowedNSSAI:      []byte{0x01, 0x01, 0x01},
		T3512Value:        []byte{0x5e, 0x01, 0x3c},
		MicoIndication:    ue.MicoMode,
	}

	if ue.EDrxParameters != nil && ue.EDrxParameters.Enabled {
		msg.NegotiatedEDrxParameters = []byte{
			ue.EDrxParameters.EDrxValue,
			ue.EDrxParameters.PagingTimeWindow,
		}
		logger.NasLog.Infof("Sending negotiated eDRX parameters: value=0x%02x, PTW=0x%02x",
			ue.EDrxParameters.EDrxValue, ue.EDrxParameters.PagingTimeWindow)
	}

	if ue.PSMParameters != nil && ue.PSMParameters.Enabled {
		if ue.PSMParameters.T3324Value > 0 {
			msg.T3324Value = []byte{ue.PSMParameters.T3324Value}
		}
		if ue.PSMParameters.T3412ExtendedValue > 0 {
			msg.T3412ExtendedValue = []byte{ue.PSMParameters.T3412ExtendedValue}
		}
		logger.NasLog.Infof("Sending PSM parameters: T3324=0x%02x, T3412ext=0x%02x",
			ue.PSMParameters.T3324Value, ue.PSMParameters.T3412ExtendedValue)
	}

	payload := EncodeRegistrationAccept(msg)

	nasData, err := EncodeSecuredNASPDU(ue, MsgTypeRegistrationAccept, payload,
		SecurityHeaderTypeIntegrityProtectedAndCiphered)
	if err != nil {
		return fmt.Errorf("failed to encode secured NAS PDU: %v", err)
	}

	ue.RegistrationState = context.RegStateRegistered
	ue.RmState = context.RmRegistered
	logger.NasLog.Infof("UE %s successfully registered", ue.Supi)

	if err := h.amfContext.PersistUEContext(ue); err != nil {
		logger.NasLog.Warnf("Failed to persist UE context: %v", err)
	}

	h.notifyRegistrationStateChange(ue)

	h.startT3550(ue)

	return h.ngapHandler.SendDownlinkNASTransport(ue.RanUeNgapId, ue.AmfUeNgapId, nasData)
}

func (h *Handler) SendEmergencyRegistrationAccept(ue *context.UEContext) error {
	logger.NasLog.Infof("Sending Emergency Registration Accept to UE")

	if ue.Guti == nil {
		ue.Guti = h.amfContext.AllocateGuti()
		logger.NasLog.Infof("Allocated GUTI for emergency UE: %+v", ue.Guti)
	}

	msg := &RegistrationAcceptMsg{
		RegistrationResult: 0x11,
		MobileIdentity:    EncodeGutiMobileIdentity(ue.Guti),
		AllowedNSSAI:      []byte{0x01, 0x01, 0x01},
	}

	payload := EncodeRegistrationAccept(msg)

	pdu := &NASPDU{
		ProtocolDiscriminator: ProtocolDiscriminator5GMM,
		SecurityHeaderType:    SecurityHeaderTypePlainNAS,
		MessageType:           MsgTypeRegistrationAccept,
		Payload:               payload,
	}

	nasData := EncodeNASPDU(pdu)

	ue.RegistrationState = context.RegStateRegistered
	ue.RmState = context.RmRegistered
	logger.NasLog.Infof("Emergency UE %s successfully registered", ue.Supi)

	if err := h.amfContext.PersistUEContext(ue); err != nil {
		logger.NasLog.Warnf("Failed to persist UE context: %v", err)
	}

	return h.ngapHandler.SendDownlinkNASTransport(ue.RanUeNgapId, ue.AmfUeNgapId, nasData)
}

func (h *Handler) SendRegistrationReject(ue *context.UEContext, cause uint8) error {
	logger.NasLog.Infof("Sending Registration Reject to UE with cause: 0x%02x", cause)

	t3502Seconds := getT3502Value()
	msg := &RegistrationRejectMsg{
		Cause5GMM:  cause,
		T3502Value: EncodeGPRSTimer2(t3502Seconds),
	}

	payload := EncodeRegistrationReject(msg)

	pdu := &NASPDU{
		ProtocolDiscriminator: ProtocolDiscriminator5GMM,
		SecurityHeaderType:    SecurityHeaderTypePlainNAS,
		MessageType:           MsgTypeRegistrationReject,
		Payload:               payload,
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

	h.notifyRegistrationStateChange(ue)

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
	ue.StopT3513()

	h.notifyConnectivityStateChange(ue)

	if err := h.DeliverPendingMessages(ue); err != nil {
		logger.NasLog.Warnf("Failed to deliver pending messages: %v", err)
	}

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

	ue.ActivePduSessions = activePduSessions

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

	if err := h.ngapHandler.SendDownlinkNASTransport(ue.RanUeNgapId, ue.AmfUeNgapId, nasData); err != nil {
		return err
	}

	h.startT3517(ue)
	return nil
}

func (h *Handler) SendControlPlaneServiceAccept(ue *context.UEContext) error {
	logger.NasLog.Infof("Sending Control Plane Service Accept to UE SUPI: %s", ue.Supi)

	msg := &ControlPlaneServiceAccept{}

	acceptPayload := EncodeControlPlaneServiceAccept(msg)

	nasData, err := EncodeSecuredNASPDU(ue, MsgTypeServiceAccept, acceptPayload,
		SecurityHeaderTypeIntegrityProtectedAndCiphered)
	if err != nil {
		return fmt.Errorf("failed to encode secured NAS PDU: %v", err)
	}

	if err := h.ngapHandler.SendDownlinkNASTransport(ue.RanUeNgapId, ue.AmfUeNgapId, nasData); err != nil {
		return err
	}

	return nil
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

func (h *Handler) SendFiveGMMStatus(ue *context.UEContext, cause uint8) error {
	logger.NasLog.Infof("Sending 5GMM Status to UE SUPI: %s with cause: 0x%02x", ue.Supi, cause)

	msg := &FiveGMMStatusMsg{
		Cause5GMM: cause,
	}

	statusPayload := EncodeFiveGMMStatus(msg)

	nasData, err := EncodeSecuredNASPDU(ue, MsgTypeFiveGMMStatus, statusPayload,
		SecurityHeaderTypeIntegrityProtectedAndCiphered)
	if err != nil {
		return fmt.Errorf("failed to encode secured NAS PDU: %v", err)
	}

	return h.ngapHandler.SendDownlinkNASTransport(ue.RanUeNgapId, ue.AmfUeNgapId, nasData)
}

func (h *Handler) DeliverPendingMessages(ue *context.UEContext) error {
	if len(ue.PendingMessages) == 0 {
		return nil
	}

	logger.NasLog.Infof("Delivering %d pending messages for UE SUPI: %s", len(ue.PendingMessages), ue.Supi)

	for _, pendingMsg := range ue.PendingMessages {
		logger.NasLog.Infof("Delivering pending message for PDU Session ID: %d", pendingMsg.PduSessionId)

		if pendingMsg.N1MessageContent != nil {
			if err := h.ngapHandler.SendDownlinkNASTransport(ue.RanUeNgapId, ue.AmfUeNgapId, pendingMsg.N1MessageContent); err != nil {
				logger.NasLog.Errorf("Failed to deliver pending N1 message: %v", err)
				continue
			}
		}

		if pendingMsg.N2SmInfo != nil && pendingMsg.PduSessionId > 0 {
			nasPdu := []byte{}
			if err := h.ngapHandler.SendPDUSessionResourceSetupRequest(ue.RanUeNgapId, ue.AmfUeNgapId, uint8(pendingMsg.PduSessionId), nasPdu, pendingMsg.N2SmInfo); err != nil {
				logger.NasLog.Errorf("Failed to deliver pending N2 message: %v", err)
				continue
			}
		}
	}

	ue.PendingMessages = nil
	logger.NasLog.Infof("All pending messages delivered for UE SUPI: %s", ue.Supi)

	if err := h.amfContext.PersistUEContext(ue); err != nil {
		logger.NasLog.Warnf("Failed to persist UE context after clearing pending messages: %v", err)
	}

	return nil
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
	ue.StopT3513()

	h.notifyConnectivityStateChange(ue)

	if err := h.DeliverPendingMessages(ue); err != nil {
		logger.NasLog.Warnf("Failed to deliver pending messages: %v", err)
	}

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

func (h *Handler) HandleControlPlaneServiceRequest(ue *context.UEContext, payload []byte) error {
	logger.NasLog.Infof("Handle Control Plane Service Request for UE SUPI: %s", ue.Supi)

	cpSrvReq, err := DecodeControlPlaneServiceRequest(payload)
	if err != nil {
		logger.NasLog.Errorf("Failed to decode control plane service request: %v", err)
		return h.SendServiceReject(ue, CauseProtocolError)
	}

	logger.NasLog.Infof("CP Service Type: %d, NgKSI: %d", cpSrvReq.ServiceType, cpSrvReq.NgKSI)

	if ue.RmState != context.RmRegistered {
		logger.NasLog.Errorf("UE not in RM-REGISTERED state: %s", ue.RmState)
		return h.SendServiceReject(ue, Cause5GSServicesNotAllowed)
	}

	if ue.SecurityContext == nil || !ue.SecurityContext.Activated {
		logger.NasLog.Errorf("Security context not activated for UE: %s", ue.Supi)
		return h.SendServiceReject(ue, CauseSecurityModeRejectedUnspecified)
	}

	if int(cpSrvReq.NgKSI) != ue.NgKsi {
		logger.NasLog.Warnf("NgKSI mismatch - Request: %d, Expected: %d", cpSrvReq.NgKSI, ue.NgKsi)
	}

	if cpSrvReq.ServiceType == ControlPlaneServiceTypeRNAUpdate {
		logger.NasLog.Infof("RNA Update requested for UE: %s", ue.Supi)
		return h.HandleRNAUpdate(ue, cpSrvReq)
	}

	ue.CmState = context.CmConnected
	ue.StopT3513()

	h.notifyConnectivityStateChange(ue)

	if err := h.DeliverPendingMessages(ue); err != nil {
		logger.NasLog.Warnf("Failed to deliver pending messages: %v", err)
	}

	if err := h.amfContext.PersistUEContext(ue); err != nil {
		logger.NasLog.Warnf("Failed to persist UE context: %v", err)
	}

	return h.SendServiceAccept(ue, []uint8{})
}

func (h *Handler) HandleRNAUpdate(ue *context.UEContext, cpSrvReq *ControlPlaneServiceRequest) error {
	logger.NasLog.Infof("Processing RNA Update for UE: %s", ue.Supi)

	if cpSrvReq.AllowedPduSessionStatus != nil && len(cpSrvReq.AllowedPduSessionStatus) > 0 {
		logger.NasLog.Infof("Allowed PDU Session Status present in RNA Update")
		for i := 0; i < len(cpSrvReq.AllowedPduSessionStatus) && i < 16; i++ {
			statusByte := cpSrvReq.AllowedPduSessionStatus[i]
			for bit := 0; bit < 8; bit++ {
				if statusByte&(1<<bit) != 0 {
					pduSessionId := int32(i*8 + bit)
					if pduSession, ok := ue.GetPduSession(pduSessionId); ok {
						logger.NasLog.Infof("PDU Session %d is allowed in RNA Update", pduSessionId)
						if pduSession.State == context.PduSessionActive {
							logger.NasLog.Infof("PDU Session %d remains active", pduSessionId)
						}
					}
				}
			}
		}
	}

	if cpSrvReq.RequestedNSSAI != nil {
		logger.NasLog.Infof("Requested NSSAI present in RNA Update")
	}

	if cpSrvReq.OldPduSessionId != 0 {
		logger.NasLog.Infof("Old PDU Session ID: %d", cpSrvReq.OldPduSessionId)
	}

	return h.SendControlPlaneServiceAccept(ue)
}

func (h *Handler) HandleULNASTransport(ue *context.UEContext, payload []byte) error {
	logger.NasLog.Infof("Handle UL NAS Transport for UE SUPI: %s", ue.Supi)

	ulMsg, err := DecodeULNASTransport(payload)
	if err != nil {
		logger.NasLog.Errorf("Failed to decode UL NAS transport: %v", err)
		return h.SendFiveGMMStatus(ue, CauseSemanticallyIncorrectMessage)
	}

	logger.NasLog.Infof("Payload Container Type: 0x%02x, PDU Session ID: %d",
		ulMsg.PayloadContainerType, ulMsg.PDUSessionID)

	if ulMsg.PayloadContainerType == PayloadContainerTypeUEParameterUpdate {
		logger.NasLog.Infof("Received UE Parameter Update response from UE")
		return h.HandleUEParameterUpdateResponse(ue, ulMsg.PayloadContainer)
	}

	if ulMsg.PayloadContainerType != PayloadContainerTypeN1SMInfo {
		logger.NasLog.Warnf("Unsupported payload container type: 0x%02x", ulMsg.PayloadContainerType)
		return h.SendFiveGMMStatus(ue, CauseMessageTypeNotCompatible)
	}

	if len(ulMsg.PayloadContainer) < 3 {
		logger.NasLog.Errorf("Payload container too short: %d bytes", len(ulMsg.PayloadContainer))
		return h.SendFiveGMMStatus(ue, CauseSemanticallyIncorrectMessage)
	}

	smPdu := ulMsg.PayloadContainer
	smPD := (smPdu[0] >> 4) & 0x0f
	smMsgType := smPdu[2]

	logger.NasLog.Infof("SM Protocol Discriminator: 0x%x, SM Message Type: 0x%02x", smPD, smMsgType)

	if smPD != ProtocolDiscriminator5GSM {
		logger.NasLog.Warnf("Invalid SM protocol discriminator: 0x%x", smPD)
		return h.SendFiveGMMStatus(ue, CauseProtocolError)
	}

	switch smMsgType {
	case MsgTypePDUSessionEstablishmentRequest:
		return h.HandlePDUSessionEstablishmentRequest(ue, ulMsg.PDUSessionID, ulMsg.PayloadContainer, ulMsg.DNN, ulMsg.SNSSAI)
	case MsgTypePDUSessionAuthenticationComplete:
		return h.HandlePDUSessionAuthenticationComplete(ue, ulMsg.PDUSessionID, ulMsg.PayloadContainer)
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
	mptcpRequested := smReq.MPTCP > 0
	logger.NasLog.Infof("PDU Session Type: %d, SSC Mode: %d, AlwaysOn Requested: %v, MPTCP Requested: %v", smReq.PDUSessionType, smReq.SSCMode, alwaysOnRequested, mptcpRequested)

	dnnStr := "internet"
	if len(dnn) > 0 {
		dnnStr = string(dnn)
	}

	requireEap := h.requireEapAuthentication(dnnStr)
	if requireEap {
		logger.NasLog.Infof("EAP authentication required for DNN: %s", dnnStr)
		return h.initiateEapAuthentication(ue, pduSessionID, smReq, dnn, snssai, dnnStr)
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

	selectedSscMode := h.selectSscMode(smReq.SSCMode, dnnStr)

	pduSessionCtx := &context.PduSessionContext{
		PduSessionId:  int32(pduSessionID),
		SmContextRef:  createResp.SmContextRef,
		SmContextId:   createResp.SmContextId,
		Dnn:           dnnStr,
		SessionAmbr:   &context.Ambr{Uplink: "100 Mbps", Downlink: "100 Mbps"},
		State:         context.PduSessionActive,
		AlwaysOn:      alwaysOnRequested,
		SscMode:       selectedSscMode,
		PduSessionType: smReq.PDUSessionType,
		MptcpRequested: mptcpRequested,
		MptcpIndication: smReq.MPTCP,
	}

	ue.PduSessions[int32(pduSessionID)] = pduSessionCtx
	logger.NasLog.Infof("PDU Session %d created for UE %s (SSC Mode: %d, PDU Session Type: %d)", pduSessionID, ue.Supi, selectedSscMode, smReq.PDUSessionType)

	if err := h.amfContext.PersistUEContext(ue); err != nil {
		logger.NasLog.Warnf("Failed to persist UE context: %v", err)
	}

	acceptMsg := &PDUSessionEstablishmentAcceptMsg{
		PDUSessionType: smReq.PDUSessionType,
		SSCMode:        selectedSscMode,
		SessionAMBR:    []byte{0x3e, 0x80, 0x3e, 0x80},
		QoSRules:       []byte{0x01, 0x01, 0x01, 0x01, 0x01, 0x01},
		PDUAddress:     []byte{0x01, 192, 168, 100, 10},
	}

	if alwaysOnRequested {
		acceptMsg.AlwaysOnPDUSessionIndication = 1
	}

	if mptcpRequested {
		acceptMsg.MPTCP = smReq.MPTCP
		logger.NasLog.Infof("MPTCP indication included in PDU Session Accept: 0x%02x", acceptMsg.MPTCP)
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

func (h *Handler) selectSscMode(requestedMode uint8, dnn string) uint8 {
	if requestedMode == 0 {
		return 1
	}

	supportedModes := map[string][]uint8{
		"internet": {1, 2, 3},
		"ims":      {1},
		"default":  {1, 2, 3},
	}

	allowedModes, exists := supportedModes[dnn]
	if !exists {
		allowedModes = supportedModes["default"]
	}

	for _, mode := range allowedModes {
		if mode == requestedMode {
			logger.NasLog.Infof("SSC Mode %d selected for DNN %s", requestedMode, dnn)
			return requestedMode
		}
	}

	logger.NasLog.Infof("Requested SSC Mode %d not supported for DNN %s, selecting mode 1", requestedMode, dnn)
	return 1
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

	var qosFlowDescs []QoSFlowDescription
	var qosRules []QoSRule

	if len(smReq.RequestedQoSRules) > 0 {
		logger.NasLog.Infof("QoS Rules modification requested")
		parsedRules, err := ParseQoSRules(smReq.RequestedQoSRules)
		if err != nil {
			logger.NasLog.Warnf("Failed to parse QoS rules: %v", err)
		} else {
			qosRules = parsedRules
			logger.NasLog.Infof("Parsed %d QoS rules", len(qosRules))
		}
	}

	if len(smReq.RequestedQoSFlowDescriptions) > 0 {
		logger.NasLog.Infof("QoS Flow Descriptions modification requested")
		parsedFlows, err := ParseQoSFlowDescriptions(smReq.RequestedQoSFlowDescriptions)
		if err != nil {
			logger.NasLog.Warnf("Failed to parse QoS flow descriptions: %v", err)
		} else {
			qosFlowDescs = parsedFlows
			logger.NasLog.Infof("Parsed %d QoS flow descriptions", len(qosFlowDescs))

			for _, flow := range qosFlowDescs {
				switch flow.OperationCode {
				case QoSFlowOperationCodeCreate:
					logger.NasLog.Infof("Creating QoS flow QFI=%d, 5QI=%d", flow.QFI, flow.FiveQI)
					qosFlow := pduSession.AddQosFlow(int(flow.QFI), int(flow.FiveQI))
					if flow.GFBR_Uplink > 0 || flow.GFBR_Downlink > 0 {
						if qosFlow.QosParameters == nil {
							qosFlow.QosParameters = &context.QosParameters{}
						}
					}

				case QoSFlowOperationCodeModify:
					logger.NasLog.Infof("Modifying QoS flow QFI=%d, 5QI=%d", flow.QFI, flow.FiveQI)
					params := &context.QosParameters{}
					pduSession.ModifyQosFlow(int(flow.QFI), int(flow.FiveQI), params)

				case QoSFlowOperationCodeDelete:
					logger.NasLog.Infof("Deleting QoS flow QFI=%d", flow.QFI)
					pduSession.DeleteQosFlow(int(flow.QFI))
				}
			}
		}
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

	if len(qosFlowDescs) > 0 {
		commandMsg.QoSFlowDescriptions = BuildQoSFlowDescriptions(qosFlowDescs)
	}

	if len(qosRules) > 0 {
		commandMsg.QoSRules = BuildQoSRules(qosRules)
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

func (h *Handler) SendNetworkInitiatedQoSModification(ue *context.UEContext, pduSessionID uint8, qosFlows []QoSFlowDescription, qosRules []QoSRule) error {
	logger.NasLog.Infof("Sending network-initiated QoS modification for PDU Session ID: %d", pduSessionID)

	pduSession, ok := ue.GetPduSession(int32(pduSessionID))
	if !ok {
		logger.NasLog.Errorf("PDU Session %d not found for UE %s", pduSessionID, ue.Supi)
		return fmt.Errorf("PDU session not found")
	}

	for _, flow := range qosFlows {
		switch flow.OperationCode {
		case QoSFlowOperationCodeCreate:
			logger.NasLog.Infof("Network creating QoS flow QFI=%d, 5QI=%d", flow.QFI, flow.FiveQI)
			qosFlow := pduSession.AddQosFlow(int(flow.QFI), int(flow.FiveQI))
			if flow.GFBR_Uplink > 0 || flow.GFBR_Downlink > 0 {
				if qosFlow.QosParameters == nil {
					qosFlow.QosParameters = &context.QosParameters{}
				}
			}

		case QoSFlowOperationCodeModify:
			logger.NasLog.Infof("Network modifying QoS flow QFI=%d, 5QI=%d", flow.QFI, flow.FiveQI)
			params := &context.QosParameters{}
			pduSession.ModifyQosFlow(int(flow.QFI), int(flow.FiveQI), params)

		case QoSFlowOperationCodeDelete:
			logger.NasLog.Infof("Network deleting QoS flow QFI=%d", flow.QFI)
			pduSession.DeleteQosFlow(int(flow.QFI))
		}
	}

	commandMsg := &PDUSessionModificationCommandMsg{}

	if len(qosFlows) > 0 {
		commandMsg.QoSFlowDescriptions = BuildQoSFlowDescriptions(qosFlows)
	}

	if len(qosRules) > 0 {
		commandMsg.QoSRules = BuildQoSRules(qosRules)
	} else {
		commandMsg.QoSRules = []byte{0x01, 0x01, 0x01, 0x01, 0x01, 0x01}
	}

	commandMsg.SessionAMBR = []byte{0x3e, 0x80, 0x3e, 0x80}

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

	if err := h.amfContext.PersistUEContext(ue); err != nil {
		logger.NasLog.Warnf("Failed to persist UE context: %v", err)
	}

	logger.NasLog.Infof("Network-initiated QoS modification sent for PDU Session ID: %d", pduSessionID)

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

func (h *Handler) HandlePDUSessionAuthenticationComplete(ue *context.UEContext, pduSessionID uint8, smMsg []byte) error {
	logger.NasLog.Infof("Handle PDU Session Authentication Complete for UE SUPI: %s, PDU Session ID: %d", ue.Supi, pduSessionID)

	if len(smMsg) < 3 {
		return fmt.Errorf("SM message too short")
	}

	smPayload := smMsg[3:]
	authComplete, err := DecodePDUSessionAuthenticationComplete(smPayload)
	if err != nil {
		return fmt.Errorf("failed to decode PDU session authentication complete: %v", err)
	}

	pduSession, ok := ue.GetPduSession(int32(pduSessionID))
	if !ok {
		logger.NasLog.Errorf("PDU Session %d not found for UE %s", pduSessionID, ue.Supi)
		return h.SendPDUSessionEstablishmentReject(ue, pduSessionID, 0x36)
	}

	if pduSession.EapState != context.EapStateInProgress {
		logger.NasLog.Warnf("PDU Session %d not in authentication state", pduSessionID)
		return h.SendPDUSessionEstablishmentReject(ue, pduSessionID, 0x36)
	}

	if len(authComplete.EAPMessage) == 0 {
		logger.NasLog.Errorf("No EAP message in authentication complete")
		return h.SendPDUSessionEstablishmentReject(ue, pduSessionID, 0x36)
	}

	valid, res, err := VerifyEAPResponse(authComplete.EAPMessage, pduSession.EapRand)
	if err != nil {
		logger.NasLog.Errorf("Failed to verify EAP response: %v", err)
		pduSession.EapState = context.EapStateFailed
		return h.SendPDUSessionEstablishmentReject(ue, pduSessionID, 0x1d)
	}

	if !valid || len(res) == 0 {
		logger.NasLog.Errorf("EAP authentication failed for PDU Session %d", pduSessionID)
		pduSession.EapState = context.EapStateFailed
		return h.SendPDUSessionEstablishmentReject(ue, pduSessionID, 0x1d)
	}

	logger.NasLog.Infof("PDU Session authentication successful for PDU Session %d", pduSessionID)
	pduSession.EapState = context.EapStateSuccess

	if len(pduSession.RequestMsg) == 0 {
		logger.NasLog.Errorf("No saved establishment request for PDU Session %d", pduSessionID)
		return h.SendPDUSessionEstablishmentReject(ue, pduSessionID, 0x1a)
	}

	logger.NasLog.Infof("Resuming PDU session establishment after successful authentication")
	return h.completePDUSessionEstablishment(ue, pduSessionID, pduSession)
}

func (h *Handler) SendPDUSessionAuthenticationCommand(ue *context.UEContext, pduSessionID uint8, rand []byte, autn []byte, identifier uint8) error {
	logger.NasLog.Infof("Sending PDU Session Authentication Command for PDU Session ID: %d", pduSessionID)

	eapRequest := GenerateEAPRequest(identifier, rand, autn)

	authCmd := &PDUSessionAuthenticationCommandMsg{
		EAPMessage: eapRequest,
	}

	smAuthPayload := EncodePDUSessionAuthenticationCommand(authCmd)

	smPDU := make([]byte, 0)
	smPDU = append(smPDU, ProtocolDiscriminator5GSM)
	smPDU = append(smPDU, pduSessionID)
	smPDU = append(smPDU, 0x00)
	smPDU = append(smPDU, MsgTypePDUSessionAuthenticationCommand)
	smPDU = append(smPDU, smAuthPayload...)

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

func (h *Handler) completePDUSessionEstablishment(ue *context.UEContext, pduSessionID uint8, pduSession *context.PduSessionContext) error {
	logger.NasLog.Infof("Completing PDU Session Establishment for PDU Session ID: %d", pduSessionID)

	if pduSession.PduSessionType == 0 {
		pduSession.PduSessionType = 0x01
	}

	acceptMsg := &PDUSessionEstablishmentAcceptMsg{
		PDUSessionType: pduSession.PduSessionType,
		SSCMode:        pduSession.SscMode,
		SessionAMBR:    []byte{0x3e, 0x80, 0x3e, 0x80},
		QoSRules:       []byte{0x01, 0x01, 0x01, 0x01, 0x01, 0x01},
		PDUAddress:     []byte{0x01, 192, 168, 100, 10},
	}

	if pduSession.AlwaysOn {
		acceptMsg.AlwaysOnPDUSessionIndication = 1
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

	if h.ngapHandler != nil {
		return h.ngapHandler.SendPDUSessionResourceSetupRequest(
			ue.RanUeNgapId, ue.AmfUeNgapId, pduSessionID, nasData, n2SmInfo)
	}

	return nil
}

func (h *Handler) requireEapAuthentication(dnn string) bool {
	return false
}

func (h *Handler) initiateEapAuthentication(ue *context.UEContext, pduSessionID uint8, smReq *PDUSessionEstablishmentRequestMsg, dnn []byte, snssai []byte, dnnStr string) error {
	logger.NasLog.Infof("Initiating EAP authentication for PDU Session %d", pduSessionID)

	rand := make([]byte, 16)
	for i := range rand {
		rand[i] = byte(i + 1)
	}

	autn := make([]byte, 16)
	for i := range autn {
		autn[i] = byte(i + 0x10)
	}

	identifier := uint8(1)

	selectedSscMode := h.selectSscMode(smReq.SSCMode, dnnStr)

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

	pduSessionCtx := &context.PduSessionContext{
		PduSessionId:   int32(pduSessionID),
		SmContextRef:   createResp.SmContextRef,
		SmContextId:    createResp.SmContextId,
		Dnn:            dnnStr,
		SessionAmbr:    &context.Ambr{Uplink: "100 Mbps", Downlink: "100 Mbps"},
		State:          context.PduSessionInactive,
		AlwaysOn:       smReq.AlwaysOnPDUSessionRequested == 1,
		SscMode:        selectedSscMode,
		PduSessionType: smReq.PDUSessionType,
		EapState:       context.EapStateInProgress,
		EapIdentifier:  identifier,
		EapRand:        rand,
		EapAutn:        autn,
	}

	ue.PduSessions[int32(pduSessionID)] = pduSessionCtx
	logger.NasLog.Infof("PDU Session %d created with EAP authentication pending", pduSessionID)

	if err := h.amfContext.PersistUEContext(ue); err != nil {
		logger.NasLog.Warnf("Failed to persist UE context: %v", err)
	}

	return h.SendPDUSessionAuthenticationCommand(ue, pduSessionID, rand, autn, identifier)
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

	cfg := factory.GetConfig()
	if cfg != nil && cfg.Configuration != nil && len(cfg.Configuration.SupportTaiList) > 0 {
		taiList := make([]context.Tai, 0, len(cfg.Configuration.SupportTaiList))
		for _, supportTai := range cfg.Configuration.SupportTaiList {
			if supportTai.PlmnId != nil {
				tai := context.Tai{
					PlmnId: context.PlmnId{
						Mcc: supportTai.PlmnId.Mcc,
						Mnc: supportTai.PlmnId.Mnc,
					},
					Tac: supportTai.Tac,
				}
				taiList = append(taiList, tai)
			}
		}
		if len(taiList) > 0 {
			msg.ServiceAreaList = EncodeServiceAreaList(taiList)
			if len(msg.ServiceAreaList) > 0 {
				logger.NasLog.Infof("Including Service Area List in Configuration Update (%d TAIs)", len(taiList))
			}
		}
	}

	if h.amfContext.TimeZoneOffsetMinutes != 0 || h.amfContext.DaylightSavingTime != 0 {
		msg.LocalTimeZone = EncodeLocalTimeZone(h.amfContext.TimeZoneOffsetMinutes)
		logger.NasLog.Infof("Including NITZ Local Time Zone in Configuration Update")
	}

	if h.amfContext.DaylightSavingTime > 0 {
		msg.NetworkDaylightSavingTime = EncodeNetworkDaylightSavingTime(h.amfContext.DaylightSavingTime)
		logger.NasLog.Infof("Including NITZ Daylight Saving Time in Configuration Update")
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

	err = h.ngapHandler.SendDownlinkNASTransport(ue.RanUeNgapId, ue.AmfUeNgapId, nasData)
	if err == nil {
		h.startT3555(ue)
	}
	return err
}

func (h *Handler) HandleConfigurationUpdateComplete(ue *context.UEContext, payload []byte) error {
	logger.NasLog.Infof("Handle Configuration Update Complete for UE SUPI: %s", ue.Supi)

	ue.StopT3555()

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

func (h *Handler) SendGenericUEConfigurationUpdate(ue *context.UEContext, indication uint8) error {
	logger.NasLog.Infof("Sending Generic UE Configuration Update to UE")

	msg := &GenericUEConfigurationUpdateCommandMsg{
		GenericUEConfigurationUpdateIndication: indication,
	}

	payload := EncodeGenericUEConfigurationUpdateCommand(msg)

	nasData, err := EncodeSecuredNASPDU(ue, MsgTypeGenericUEConfigurationUpdateCommand, payload,
		SecurityHeaderTypeIntegrityProtectedAndCiphered)
	if err != nil {
		return fmt.Errorf("failed to encode secured NAS PDU: %v", err)
	}

	return h.ngapHandler.SendDownlinkNASTransport(ue.RanUeNgapId, ue.AmfUeNgapId, nasData)
}

func (h *Handler) HandleGenericUEConfigurationUpdateComplete(ue *context.UEContext, payload []byte) error {
	logger.NasLog.Infof("Handle Generic UE Configuration Update Complete for UE SUPI: %s", ue.Supi)

	_, err := DecodeGenericUEConfigurationUpdateComplete(payload)
	if err != nil {
		logger.NasLog.Errorf("Failed to decode Generic UE Configuration Update Complete: %v", err)
		return err
	}

	logger.NasLog.Infof("Generic UE Configuration Update Complete processed successfully for UE: %s", ue.Supi)

	if err := h.amfContext.PersistUEContext(ue); err != nil {
		logger.NasLog.Warnf("Failed to persist UE context: %v", err)
	}

	return nil
}

func (h *Handler) SendUEParameterUpdate(ue *context.UEContext, updateData []byte) error {
	logger.NasLog.Infof("Sending UE Parameter Update to UE SUPI: %s", ue.Supi)

	msg := &DLNASTransportMsg{
		PayloadContainerType: PayloadContainerTypeUEParameterUpdate,
		PayloadContainer:     updateData,
	}

	payload := EncodeDLNASTransport(msg)

	nasData, err := EncodeSecuredNASPDU(ue, MsgTypeDLNASTransport, payload,
		SecurityHeaderTypeIntegrityProtectedAndCiphered)
	if err != nil {
		return fmt.Errorf("failed to encode secured NAS PDU: %v", err)
	}

	ue.LastParameterUpdateTime = time.Now()

	if err := h.amfContext.PersistUEContext(ue); err != nil {
		logger.NasLog.Warnf("Failed to persist UE context: %v", err)
	}

	return h.ngapHandler.SendDownlinkNASTransport(ue.RanUeNgapId, ue.AmfUeNgapId, nasData)
}

func (h *Handler) HandleUEParameterUpdateResponse(ue *context.UEContext, payloadContainer []byte) error {
	logger.NasLog.Infof("Handle UE Parameter Update Response for UE SUPI: %s", ue.Supi)

	logger.NasLog.Infof("UE Parameter Update acknowledged by UE, payload length: %d bytes", len(payloadContainer))

	if err := h.amfContext.PersistUEContext(ue); err != nil {
		logger.NasLog.Warnf("Failed to persist UE context: %v", err)
	}

	return nil
}

func (h *Handler) SendDeregistrationRequest(ue *context.UEContext, deregType uint8, cause uint8, reregistrationRequired bool) error {
	logger.NasLog.Infof("Sending Network-Initiated Deregistration Request to UE SUPI: %s", ue.Supi)

	ue.DeregType = deregType
	ue.DeregCause = cause
	ue.DeregReregRequired = reregistrationRequired

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

	err = h.ngapHandler.SendDownlinkNASTransport(ue.RanUeNgapId, ue.AmfUeNgapId, nasData)
	if err != nil {
		return err
	}

	h.startT3540(ue)

	return nil
}

func (h *Handler) HandleDeregistrationAccept(ue *context.UEContext, payload []byte) error {
	logger.NasLog.Infof("Handle Deregistration Accept (UE Terminating) for UE SUPI: %s", ue.Supi)

	ue.StopT3540()

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

func (h *Handler) startT3560(ue *context.UEContext, rand, autn []byte) {
	ue.StopT3560()

	cfg := h.amfContext
	if cfg == nil {
		return
	}

	timerConfig := getTimerConfig("T3560")
	if !timerConfig.Enable {
		return
	}

	ue.T3560Counter++
	logger.NasLog.Infof("Starting T3560 timer (attempt %d/%d) for UE: %s",
		ue.T3560Counter, timerConfig.MaxRetryTimes, ue.Supi)

	ue.T3560 = time.AfterFunc(time.Duration(timerConfig.ExpireTime)*time.Second, func() {
		logger.NasLog.Warnf("T3560 timer expired for UE: %s (attempt %d/%d)",
			ue.Supi, ue.T3560Counter, timerConfig.MaxRetryTimes)

		if ue.T3560Counter < timerConfig.MaxRetryTimes {
			logger.NasLog.Infof("Retransmitting Authentication Request for UE: %s", ue.Supi)
			h.SendAuthenticationRequest(ue, rand, autn)
			h.startT3560(ue, rand, autn)
		} else {
			logger.NasLog.Errorf("T3560 max retries reached for UE: %s, authentication failed", ue.Supi)
			ue.StopT3560()
			h.SendAuthenticationReject(ue)
			if h.ngapHandler != nil && ue.Supi != "" {
				h.ngapHandler.NotifyCommunicationFailure(ue, "AUTHENTICATION_FAILURE")
			}
		}
	})
}

func (h *Handler) startT3565(ue *context.UEContext) {
	ue.StopT3565()

	timerConfig := getTimerConfig("T3565")
	if !timerConfig.Enable {
		return
	}

	ue.T3565Counter++
	logger.NasLog.Infof("Starting T3565 timer (attempt %d/%d) for UE: %s",
		ue.T3565Counter, timerConfig.MaxRetryTimes, ue.Supi)

	ue.T3565 = time.AfterFunc(time.Duration(timerConfig.ExpireTime)*time.Second, func() {
		logger.NasLog.Warnf("T3565 timer expired for UE: %s (attempt %d/%d)",
			ue.Supi, ue.T3565Counter, timerConfig.MaxRetryTimes)

		if ue.T3565Counter < timerConfig.MaxRetryTimes {
			logger.NasLog.Infof("Retransmitting Security Mode Command for UE: %s", ue.Supi)
			h.SendSecurityModeCommand(ue)
			h.startT3565(ue)
		} else {
			logger.NasLog.Errorf("T3565 max retries reached for UE: %s, security mode failed", ue.Supi)
			ue.StopT3565()
			h.SendRegistrationReject(ue, CauseProtocolError)
			if h.ngapHandler != nil && ue.Supi != "" {
				h.ngapHandler.NotifyCommunicationFailure(ue, "SECURITY_MODE_FAILURE")
			}
		}
	})
}

func (h *Handler) startT3570(ue *context.UEContext, identityType uint8) {
	ue.StopT3570()

	timerConfig := getTimerConfig("T3570")
	if !timerConfig.Enable {
		return
	}

	ue.T3570Counter++
	logger.NasLog.Infof("Starting T3570 timer (attempt %d/%d) for UE: %s",
		ue.T3570Counter, timerConfig.MaxRetryTimes, ue.Supi)

	ue.T3570 = time.AfterFunc(time.Duration(timerConfig.ExpireTime)*time.Second, func() {
		logger.NasLog.Warnf("T3570 timer expired for UE: %s (attempt %d/%d)",
			ue.Supi, ue.T3570Counter, timerConfig.MaxRetryTimes)

		if ue.T3570Counter < timerConfig.MaxRetryTimes {
			logger.NasLog.Infof("Retransmitting Identity Request for UE: %s", ue.Supi)
			h.SendIdentityRequest(ue, identityType)
		} else {
			logger.NasLog.Errorf("T3570 max retries reached for UE: %s, identity request failed", ue.Supi)
			ue.StopT3570()
			h.SendRegistrationReject(ue, CauseProtocolError)
			if h.ngapHandler != nil && ue.Supi != "" {
				h.ngapHandler.NotifyCommunicationFailure(ue, "IDENTITY_REQUEST_FAILURE")
			}
		}
	})
}

func (h *Handler) startT3550(ue *context.UEContext) {
	ue.StopT3550()

	timerConfig := getTimerConfig("T3550")
	if !timerConfig.Enable {
		return
	}

	ue.T3550Counter++
	logger.NasLog.Infof("Starting T3550 timer (attempt %d/%d) for UE: %s",
		ue.T3550Counter, timerConfig.MaxRetryTimes, ue.Supi)

	ue.T3550 = time.AfterFunc(time.Duration(timerConfig.ExpireTime)*time.Second, func() {
		logger.NasLog.Warnf("T3550 timer expired for UE: %s (attempt %d/%d)",
			ue.Supi, ue.T3550Counter, timerConfig.MaxRetryTimes)

		if ue.T3550Counter < timerConfig.MaxRetryTimes {
			logger.NasLog.Infof("Retransmitting Registration Accept for UE: %s", ue.Supi)
			h.SendRegistrationAccept(ue)
			h.startT3550(ue)
		} else {
			logger.NasLog.Errorf("T3550 max retries reached for UE: %s, registration failed", ue.Supi)
			ue.StopT3550()
			if h.ngapHandler != nil && ue.Supi != "" {
				h.ngapHandler.NotifyCommunicationFailure(ue, "REGISTRATION_ACCEPT_FAILURE")
			}
		}
	})
}

func (h *Handler) startT3555(ue *context.UEContext) {
	ue.StopT3555()

	timerConfig := getTimerConfig("T3555")
	if !timerConfig.Enable {
		return
	}

	ue.T3555Counter++
	logger.NasLog.Infof("Starting T3555 timer (attempt %d/%d) for UE: %s",
		ue.T3555Counter, timerConfig.MaxRetryTimes, ue.Supi)

	ue.T3555 = time.AfterFunc(time.Duration(timerConfig.ExpireTime)*time.Second, func() {
		logger.NasLog.Warnf("T3555 timer expired for UE: %s (attempt %d/%d)",
			ue.Supi, ue.T3555Counter, timerConfig.MaxRetryTimes)

		if ue.T3555Counter < timerConfig.MaxRetryTimes {
			logger.NasLog.Infof("Retransmitting Configuration Update Command for UE: %s", ue.Supi)
			h.SendConfigurationUpdateCommand(ue, ue.Guti)
			h.startT3555(ue)
		} else {
			logger.NasLog.Errorf("T3555 max retries reached for UE: %s, configuration update failed", ue.Supi)
			ue.StopT3555()
			if h.ngapHandler != nil && ue.Supi != "" {
				h.ngapHandler.NotifyCommunicationFailure(ue, "CONFIGURATION_UPDATE_FAILURE")
			}
		}
	})
}

func (h *Handler) startT3512(ue *context.UEContext) {
	ue.StopT3512()

	cfg := h.amfContext
	if cfg == nil {
		return
	}

	t3512ValueSeconds := getT3512Value()
	if t3512ValueSeconds <= 0 {
		return
	}

	logger.NasLog.Infof("Starting T3512 timer (%d seconds) for UE: %s", t3512ValueSeconds, ue.Supi)

	ue.T3512 = time.AfterFunc(time.Duration(t3512ValueSeconds)*time.Second, func() {
		logger.NasLog.Infof("T3512 timer expired for UE: %s, expecting periodic registration update", ue.Supi)
	})
}

func (h *Handler) startT3540(ue *context.UEContext) {
	ue.StopT3540()

	timerConfig := getTimerConfig("T3540")
	if !timerConfig.Enable {
		return
	}

	ue.T3540Counter++
	logger.NasLog.Infof("Starting T3540 timer (attempt %d/%d) for UE: %s",
		ue.T3540Counter, timerConfig.MaxRetryTimes, ue.Supi)

	ue.T3540 = time.AfterFunc(time.Duration(timerConfig.ExpireTime)*time.Second, func() {
		logger.NasLog.Warnf("T3540 timer expired for UE: %s (attempt %d/%d)",
			ue.Supi, ue.T3540Counter, timerConfig.MaxRetryTimes)

		if ue.T3540Counter < timerConfig.MaxRetryTimes {
			logger.NasLog.Infof("Retransmitting Deregistration Request for UE: %s", ue.Supi)
			h.SendDeregistrationRequest(ue, ue.DeregType, ue.DeregCause, ue.DeregReregRequired)
		} else {
			logger.NasLog.Errorf("T3540 max retries reached for UE: %s, deregistration failed", ue.Supi)
			ue.StopT3540()
			if h.ngapHandler != nil && ue.Supi != "" {
				h.ngapHandler.NotifyCommunicationFailure(ue, "DEREGISTRATION_FAILURE")
			}
			ue.RegistrationState = context.RegStateDeregistered
			ue.RmState = context.RmDeregistered
			if err := h.amfContext.PersistUEContext(ue); err != nil {
				logger.NasLog.Warnf("Failed to persist UE context: %v", err)
			}
			h.amfContext.DeleteUEContext(ue.AmfUeNgapId)
			logger.NasLog.Infof("UE context deleted for AMF UE NGAP ID: %d after T3540 timeout", ue.AmfUeNgapId)
		}
	})
}

func (h *Handler) startT3522(ue *context.UEContext) {
	ue.StopT3522()

	timerConfig := getTimerConfig("T3522")
	if !timerConfig.Enable {
		return
	}

	ue.T3522Counter++
	logger.NasLog.Infof("Starting T3522 timer (attempt %d/%d) for UE: %s",
		ue.T3522Counter, timerConfig.MaxRetryTimes, ue.Supi)

	ue.T3522 = time.AfterFunc(time.Duration(timerConfig.ExpireTime)*time.Second, func() {
		logger.NasLog.Warnf("T3522 timer expired for UE: %s (attempt %d/%d)",
			ue.Supi, ue.T3522Counter, timerConfig.MaxRetryTimes)

		if ue.T3522Counter < timerConfig.MaxRetryTimes {
			logger.NasLog.Infof("Retransmitting Deregistration Request for UE: %s", ue.Supi)
			h.SendDeregistrationRequest(ue, ue.DeregType, ue.DeregCause, ue.DeregReregRequired)
		} else {
			logger.NasLog.Errorf("T3522 max retries reached for UE: %s, deregistration failed", ue.Supi)
			ue.StopT3522()
			if h.ngapHandler != nil && ue.Supi != "" {
				h.ngapHandler.NotifyCommunicationFailure(ue, "DEREGISTRATION_ACCEPT_FAILURE")
			}
			ue.RegistrationState = context.RegStateDeregistered
			ue.RmState = context.RmDeregistered
			if err := h.amfContext.PersistUEContext(ue); err != nil {
				logger.NasLog.Warnf("Failed to persist UE context: %v", err)
			}
			h.amfContext.DeleteUEContext(ue.AmfUeNgapId)
			logger.NasLog.Infof("UE context deleted for AMF UE NGAP ID: %d after T3522 timeout", ue.AmfUeNgapId)
		}
	})
}

func (h *Handler) startT3513(ue *context.UEContext) {
	ue.StopT3513()

	timerConfig := getTimerConfig("T3513")
	if !timerConfig.Enable {
		return
	}

	ue.T3513Counter++
	logger.NasLog.Infof("Starting T3513 timer (attempt %d/%d) for UE: %s",
		ue.T3513Counter, timerConfig.MaxRetryTimes, ue.Supi)

	ue.T3513 = time.AfterFunc(time.Duration(timerConfig.ExpireTime)*time.Second, func() {
		logger.NasLog.Warnf("T3513 timer expired for UE: %s (attempt %d/%d)",
			ue.Supi, ue.T3513Counter, timerConfig.MaxRetryTimes)

		if ue.T3513Counter < timerConfig.MaxRetryTimes {
			logger.NasLog.Infof("Retransmitting Paging for UE: %s", ue.Supi)
			if err := h.ngapHandler.(interface{ SendPaging(*context.UEContext) error }).SendPaging(ue); err != nil {
				logger.NasLog.Errorf("Failed to send paging: %v", err)
			}
			h.startT3513(ue)
		} else {
			logger.NasLog.Errorf("T3513 max retries reached for UE: %s, paging failed", ue.Supi)
			ue.StopT3513()
			if h.ngapHandler != nil && ue.Supi != "" {
				h.ngapHandler.NotifyCommunicationFailure(ue, "PAGING_FAILURE")
			}
		}
	})
}

func (h *Handler) startT3517(ue *context.UEContext) {
	ue.StopT3517()

	timerConfig := getTimerConfig("T3517")
	if !timerConfig.Enable {
		return
	}

	ue.T3517Counter++
	logger.NasLog.Infof("Starting T3517 timer (attempt %d/%d) for UE: %s",
		ue.T3517Counter, timerConfig.MaxRetryTimes, ue.Supi)

	ue.T3517 = time.AfterFunc(time.Duration(timerConfig.ExpireTime)*time.Second, func() {
		logger.NasLog.Warnf("T3517 timer expired for UE: %s (attempt %d/%d)",
			ue.Supi, ue.T3517Counter, timerConfig.MaxRetryTimes)

		if ue.T3517Counter < timerConfig.MaxRetryTimes {
			logger.NasLog.Infof("Retransmitting Service Accept for UE: %s", ue.Supi)
			h.SendServiceAccept(ue, ue.ActivePduSessions)
			h.startT3517(ue)
		} else {
			logger.NasLog.Errorf("T3517 max retries reached for UE: %s, service accept failed", ue.Supi)
			ue.StopT3517()
			if h.ngapHandler != nil && ue.Supi != "" {
				h.ngapHandler.NotifyCommunicationFailure(ue, "SERVICE_ACCEPT_FAILURE")
			}
		}
	})
}

func (h *Handler) startT3519(ue *context.UEContext) {
	ue.StopT3519()

	timerConfig := getTimerConfig("T3519")
	if !timerConfig.Enable {
		return
	}

	ue.T3519Counter++
	logger.NasLog.Infof("Starting T3519 timer (attempt %d/%d) for UE: %s",
		ue.T3519Counter, timerConfig.MaxRetryTimes, ue.Supi)

	ue.T3519 = time.AfterFunc(time.Duration(timerConfig.ExpireTime)*time.Second, func() {
		logger.NasLog.Warnf("T3519 timer expired for UE: %s (attempt %d/%d)",
			ue.Supi, ue.T3519Counter, timerConfig.MaxRetryTimes)

		if ue.T3519Counter < timerConfig.MaxRetryTimes {
			logger.NasLog.Infof("Retransmitting Notification for UE: %s", ue.Supi)
			h.startT3519(ue)
		} else {
			logger.NasLog.Errorf("T3519 max retries reached for UE: %s, notification failed", ue.Supi)
			ue.StopT3519()
			if h.ngapHandler != nil && ue.Supi != "" {
				h.ngapHandler.NotifyCommunicationFailure(ue, "NOTIFICATION_FAILURE")
			}
		}
	})
}

func (h *Handler) startT3510(ue *context.UEContext) {
	ue.StopT3510()

	timerConfig := getTimerConfig("T3510")
	if !timerConfig.Enable {
		return
	}

	ue.T3510Counter++
	logger.NasLog.Infof("Starting T3510 timer (attempt %d/%d) for UE: %s",
		ue.T3510Counter, timerConfig.MaxRetryTimes, ue.Supi)

	ue.T3510 = time.AfterFunc(time.Duration(timerConfig.ExpireTime)*time.Second, func() {
		logger.NasLog.Warnf("T3510 timer expired for UE: %s (attempt %d/%d)",
			ue.Supi, ue.T3510Counter, timerConfig.MaxRetryTimes)

		if ue.T3510Counter < timerConfig.MaxRetryTimes {
			logger.NasLog.Infof("Retransmitting Registration Accept for UE: %s", ue.Supi)
			h.SendRegistrationAccept(ue)
			h.startT3510(ue)
		} else {
			logger.NasLog.Errorf("T3510 max retries reached for UE: %s, registration procedure failed", ue.Supi)
			ue.StopT3510()
			if h.ngapHandler != nil && ue.Supi != "" {
				h.ngapHandler.NotifyCommunicationFailure(ue, "REGISTRATION_PROCEDURE_FAILURE")
			}
		}
	})
}

func (h *Handler) startT3511(ue *context.UEContext) {
	ue.StopT3511()

	timerConfig := getTimerConfig("T3511")
	if !timerConfig.Enable {
		return
	}

	ue.T3511Counter++
	logger.NasLog.Infof("Starting T3511 timer for UE: %s (registration retry wait timer)",
		ue.Supi)

	ue.T3511 = time.AfterFunc(time.Duration(timerConfig.ExpireTime)*time.Second, func() {
		logger.NasLog.Infof("T3511 timer expired for UE: %s, UE may retry registration", ue.Supi)
		ue.StopT3511()
	})
}

func (h *Handler) startT3516(ue *context.UEContext) {
	ue.StopT3516()

	timerConfig := getTimerConfig("T3516")
	if !timerConfig.Enable {
		return
	}

	ue.T3516Counter++
	logger.NasLog.Infof("Starting T3516 timer (attempt %d/%d) for UE: %s",
		ue.T3516Counter, timerConfig.MaxRetryTimes, ue.Supi)

	ue.T3516 = time.AfterFunc(time.Duration(timerConfig.ExpireTime)*time.Second, func() {
		logger.NasLog.Warnf("T3516 timer expired for UE: %s (attempt %d/%d)",
			ue.Supi, ue.T3516Counter, timerConfig.MaxRetryTimes)

		if ue.T3516Counter < timerConfig.MaxRetryTimes {
			logger.NasLog.Infof("Retrying 5GMM Status procedure for UE: %s", ue.Supi)
			h.startT3516(ue)
		} else {
			logger.NasLog.Errorf("T3516 max retries reached for UE: %s", ue.Supi)
			ue.StopT3516()
		}
	})
}

func (h *Handler) startT3520(ue *context.UEContext) {
	ue.StopT3520()

	timerConfig := getTimerConfig("T3520")
	if !timerConfig.Enable {
		return
	}

	ue.T3520Counter++
	logger.NasLog.Infof("Starting T3520 timer (attempt %d/%d) for UE: %s",
		ue.T3520Counter, timerConfig.MaxRetryTimes, ue.Supi)

	ue.T3520 = time.AfterFunc(time.Duration(timerConfig.ExpireTime)*time.Second, func() {
		logger.NasLog.Warnf("T3520 timer expired for UE: %s (attempt %d/%d)",
			ue.Supi, ue.T3520Counter, timerConfig.MaxRetryTimes)

		if ue.T3520Counter < timerConfig.MaxRetryTimes {
			logger.NasLog.Infof("Retransmitting GUTI reallocation for UE: %s", ue.Supi)
			if ue.Guti != nil {
				h.SendConfigurationUpdateCommand(ue, ue.Guti)
			}
			h.startT3520(ue)
		} else {
			logger.NasLog.Errorf("T3520 max retries reached for UE: %s, GUTI reallocation failed", ue.Supi)
			ue.StopT3520()
			if h.ngapHandler != nil && ue.Supi != "" {
				h.ngapHandler.NotifyCommunicationFailure(ue, "GUTI_REALLOCATION_FAILURE")
			}
		}
	})
}

func (h *Handler) startT3521(ue *context.UEContext) {
	ue.StopT3521()

	timerConfig := getTimerConfig("T3521")
	if !timerConfig.Enable {
		return
	}

	ue.T3521Counter++
	logger.NasLog.Infof("Starting T3521 timer for UE: %s (UE deregistration request wait timer)",
		ue.Supi)

	ue.T3521 = time.AfterFunc(time.Duration(timerConfig.ExpireTime)*time.Second, func() {
		logger.NasLog.Warnf("T3521 timer expired for UE: %s, deregistration request not received", ue.Supi)
		ue.StopT3521()
	})
}

func (h *Handler) startT3525(ue *context.UEContext) {
	ue.StopT3525()

	timerConfig := getTimerConfig("T3525")
	if !timerConfig.Enable {
		return
	}

	ue.T3525Counter++
	logger.NasLog.Infof("Starting T3525 timer for UE: %s (identity response wait timer)",
		ue.Supi)

	ue.T3525 = time.AfterFunc(time.Duration(timerConfig.ExpireTime)*time.Second, func() {
		logger.NasLog.Warnf("T3525 timer expired for UE: %s, identity response not received", ue.Supi)
		ue.StopT3525()
		if h.ngapHandler != nil && ue.Supi != "" {
			h.ngapHandler.NotifyCommunicationFailure(ue, "IDENTITY_RESPONSE_TIMEOUT")
		}
	})
}

func getTimerConfig(timerName string) *TimerConfig {
	cfg := factory.GetConfig()
	if cfg == nil || cfg.Configuration == nil {
		return &TimerConfig{Enable: false, ExpireTime: 6, MaxRetryTimes: 4}
	}

	var timerValue *factory.TimerValue
	switch timerName {
	case "T3513":
		timerValue = cfg.Configuration.T3513
	case "T3517":
		timerValue = cfg.Configuration.T3517
	case "T3522":
		timerValue = cfg.Configuration.T3522
	case "T3540":
		timerValue = cfg.Configuration.T3540
	case "T3550":
		timerValue = cfg.Configuration.T3550
	case "T3555":
		timerValue = cfg.Configuration.T3555
	case "T3560":
		timerValue = cfg.Configuration.T3560
	case "T3565":
		timerValue = cfg.Configuration.T3565
	case "T3570":
		timerValue = cfg.Configuration.T3570
	case "T3519":
		timerValue = cfg.Configuration.T3519
	case "T3510":
		timerValue = cfg.Configuration.T3510
	case "T3511":
		timerValue = cfg.Configuration.T3511
	case "T3516":
		timerValue = cfg.Configuration.T3516
	case "T3520":
		timerValue = cfg.Configuration.T3520
	case "T3521":
		timerValue = cfg.Configuration.T3521
	case "T3525":
		timerValue = cfg.Configuration.T3525
	default:
		return &TimerConfig{Enable: false, ExpireTime: 6, MaxRetryTimes: 4}
	}

	if timerValue == nil {
		return &TimerConfig{Enable: false, ExpireTime: 6, MaxRetryTimes: 4}
	}

	return &TimerConfig{
		Enable:        timerValue.Enable,
		ExpireTime:    timerValue.ExpireTime,
		MaxRetryTimes: timerValue.MaxRetryTimes,
	}
}

func getT3512Value() int {
	cfg := factory.GetConfig()
	if cfg == nil || cfg.Configuration == nil {
		return 3600
	}
	if cfg.Configuration.T3512Value > 0 {
		return cfg.Configuration.T3512Value
	}
	return 3600
}

func getT3502Value() int {
	cfg := factory.GetConfig()
	if cfg == nil || cfg.Configuration == nil {
		return 720
	}
	if cfg.Configuration.T3502Value > 0 {
		return cfg.Configuration.T3502Value
	}
	return 720
}

type TimerConfig struct {
	Enable        bool
	ExpireTime    int
	MaxRetryTimes int
}

func (h *Handler) StartPagingTimer(ue *context.UEContext) error {
	h.startT3513(ue)
	return nil
}

func (h *Handler) notifyRegistrationStateChange(ue *context.UEContext) {
	if ue.Supi == "" {
		return
	}

	eventType := "REGISTRATION_STATE_REPORT"
	rmState := "REGISTERED"
	if ue.RmState == context.RmDeregistered {
		rmState = "DEREGISTERED"
	}

	additionalData := map[string]interface{}{
		"rmState": rmState,
	}

	h.notifyEvent(eventType, ue.Supi, additionalData)
}

func (h *Handler) notifyConnectivityStateChange(ue *context.UEContext) {
	if ue.Supi == "" {
		return
	}

	eventType := "CONNECTIVITY_STATE_REPORT"
	cmState := "CONNECTED"
	if ue.CmState == context.CmIdle {
		cmState = "IDLE"
	}

	additionalData := map[string]interface{}{
		"cmState": cmState,
	}

	h.notifyEvent(eventType, ue.Supi, additionalData)
}

func (h *Handler) notifyReachabilityChange(ue *context.UEContext, reachable bool) {
	if ue.Supi == "" {
		return
	}

	eventType := "REACHABILITY_REPORT"
	reachability := "REACHABLE"
	if !reachable {
		reachability = "UNREACHABLE"
	}

	additionalData := map[string]interface{}{
		"reachability": reachability,
	}

	h.notifyEvent(eventType, ue.Supi, additionalData)
}

func (h *Handler) notifyEvent(eventType string, supi string, additionalData map[string]interface{}) {
	subscriptions := h.amfContext.GetAllEventSubscriptions()
	if len(subscriptions) == 0 {
		return
	}

	logger.NasLog.Debugf("Triggering %s event for SUPI %s", eventType, supi)
}
