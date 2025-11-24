package nas

import (
	"encoding/binary"
	"fmt"

	"github.com/gavin/amf/internal/context"
)

const (
	SecurityHeaderTypePlainNAS                        = 0x00
	SecurityHeaderTypeIntegrityProtected              = 0x01
	SecurityHeaderTypeIntegrityProtectedAndCiphered   = 0x02
	SecurityHeaderTypeIntegrityProtectedWithNewContext = 0x03
	SecurityHeaderTypeIntegrityProtectedAndCipheredWithNewContext = 0x04
)

const (
	MsgTypeRegistrationRequest        = 0x41
	MsgTypeRegistrationAccept         = 0x42
	MsgTypeRegistrationComplete       = 0x43
	MsgTypeRegistrationReject         = 0x44
	MsgTypeDeregistrationRequestUEOriginating = 0x45
	MsgTypeDeregistrationAcceptUEOriginating  = 0x46
	MsgTypeDeregistrationRequestUETerminating = 0x47
	MsgTypeDeregistrationAcceptUETerminating  = 0x48
	MsgTypeServiceRequest             = 0x4c
	MsgTypeServiceReject              = 0x4d
	MsgTypeServiceAccept              = 0x4e
	MsgTypeExtendedServiceRequest     = 0x4f
	MsgTypeConfigurationUpdateCommand = 0x54
	MsgTypeConfigurationUpdateComplete = 0x55
	MsgTypeAuthenticationRequest      = 0x56
	MsgTypeAuthenticationResponse     = 0x57
	MsgTypeAuthenticationReject       = 0x58
	MsgTypeAuthenticationFailure      = 0x59
	MsgTypeAuthenticationResult       = 0x5a
	MsgTypeIdentityRequest            = 0x5b
	MsgTypeIdentityResponse           = 0x5c
	MsgTypeSecurityModeCommand        = 0x5d
	MsgTypeSecurityModeComplete       = 0x5e
	MsgTypeSecurityModeReject         = 0x5f
	MsgTypeULNASTransport             = 0x67
	MsgTypeDLNASTransport             = 0x68
	MsgTypeGenericUEConfigurationUpdateCommand  = 0xd0
	MsgTypeGenericUEConfigurationUpdateComplete = 0xd1
)

const (
	IEIRegistrationType       = 0x50
	IEINGKSIAndRegistrationType = 0x50
	IEIMCCMNC                 = 0x13
	IEISUPIORSUCI             = 0x77
	IEIGUTI                   = 0x77
	IEIUESecurityCapability   = 0x2e
	IEI5GMMCapability         = 0x10
	IEIPDUSessionStatus       = 0x50
	IEIRequestedNSSAI         = 0x2f
	IEIAllowedNSSAI           = 0x15
	IEIPLMN                   = 0x13
	IEIT3512Value             = 0x5e
	IEIT3502Value             = 0x16
	IEIEAPMessage             = 0x78
	IEIABBA                   = 0x38
	IEINASKeySetIdentifier    = 0x0b
	IEINASSecurityAlgorithms  = 0x57
	IEIReplayed5GSTMSIValue   = 0x77
	IEIIMEISVRequest          = 0xe0
	IEIPayloadContainerType   = 0x08
	IEIPayloadContainer       = 0x7b
	IEIPDUSessionID           = 0x12
	IEIDNN                    = 0x25
	IEISSNSSAl                = 0x22
	IEIRequestedEDrxParameters = 0x6e
	IEINegotiatedEDrxParameters = 0x6e
)

const (
	RegistrationTypeInitial          = 0x01
	RegistrationTypeMobilityUpdate   = 0x02
	RegistrationTypePeriodicUpdate   = 0x03
	RegistrationTypeEmergency        = 0x04
)

const (
	DeregistrationReRegistrationNotRequired = 0x00
	DeregistrationReRegistrationRequired    = 0x08
)

const (
	ProtocolDiscriminator5GMM = 0x7e
	ProtocolDiscriminator5GSM = 0x2e
)

const (
	PayloadContainerTypeN1SMInfo         = 0x01
	PayloadContainerTypeSMS              = 0x02
	PayloadContainerTypeLPP              = 0x03
	PayloadContainerTypeSOR              = 0x04
	PayloadContainerTypeUEPolicy         = 0x05
	PayloadContainerTypeUEParameterUpdate = 0x06
)

const (
	MsgTypePDUSessionEstablishmentRequest = 0xc1
	MsgTypePDUSessionEstablishmentAccept  = 0xc2
	MsgTypePDUSessionEstablishmentReject  = 0xc3
	MsgTypePDUSessionModificationRequest  = 0xc9
	MsgTypePDUSessionModificationCommand  = 0xca
	MsgTypePDUSessionReleaseRequest       = 0xd1
	MsgTypePDUSessionReleaseCommand       = 0xd2
	MsgTypePDUSessionReleaseComplete      = 0xd3
	MsgTypeFiveGSMStatus                  = 0x64
)

const (
	DeregistrationTypeNormal      = 0x01
	DeregistrationTypeReregistration = 0x02
	DeregistrationTypeDisableN1 = 0x03
)

const (
	IdentityTypeSUPI   = 0x01
	IdentityTypeIMEI   = 0x02
	IdentityTypeIMEISV = 0x03
	IdentityTypeSUCI   = 0x04
)

const (
	CauseIllegalUE                    = 0x03
	CauseIMEINotAccepted              = 0x05
	CauseIllegalME                    = 0x06
	Cause5GSServicesNotAllowed        = 0x07
	CausePLMNNotAllowed               = 0x0b
	CauseTrackingAreaNotAllowed       = 0x0c
	CauseRoamingNotAllowed            = 0x0d
	CauseNoSuitableCellsInTrackingArea = 0x0f
	CauseMACFailure                   = 0x14
	CauseSynchFailure                 = 0x15
	CauseCongestion                   = 0x16
	CauseUESecurityCapabilitiesMismatch = 0x17
	CauseSecurityModeRejectedUnspecified = 0x18
	CauseNonEPSAuthenticationUnacceptable = 0x1a
	CauseN1ModeNotAllowed             = 0x1b
	CausePayloadWasNotForwarded       = 0x3a
	CauseDNNNotSupportedInSlice       = 0x3b
	CauseInsufficientResourcesForSlice = 0x3c
	CauseSemanticallyIncorrectMessage = 0x5f
	CauseInvalidMandatoryInformation  = 0x60
	CauseMessageTypeNonExistent       = 0x61
	CauseMessageTypeNotCompatible     = 0x62
	CauseIENotImplemented             = 0x63
	CauseProtocolError                = 0x6f
)

type NASPDU struct {
	ProtocolDiscriminator uint8
	SecurityHeaderType    uint8
	MessageType           uint8
	SequenceNumber        uint8
	Payload               []byte
	MAC                   []byte
}

type RegistrationRequestMsg struct {
	RegistrationType      uint8
	NgKSI                 uint8
	MobileIdentity        []byte
	UESecurityCapability  []byte
	RequestedNSSAI        []byte
	LastVisitedTAI        []byte
	UENetworkCapability   []byte
	AdditionalGUTI        []byte
	MicoIndication        bool
	RequestedEDrxParameters []byte
}

type RegistrationAcceptMsg struct {
	RegistrationResult    uint8
	MobileIdentity        []byte
	TAIList               []byte
	AllowedNSSAI          []byte
	T3512Value            []byte
	T3502Value            []byte
	EmergencyNumberList   []byte
	MicoIndication        bool
	NetworkSlicingIndication bool
	NegotiatedEDrxParameters []byte
}

type RegistrationRejectMsg struct {
	Cause5GMM             uint8
	T3502Value            []byte
	T3346Value            []byte
	EAPMessage            []byte
}

type AuthenticationRequestMsg struct {
	NgKSI                 uint8
	ABBA                  []byte
	RAND                  []byte
	AUTN                  []byte
	EAPMessage            []byte
}

type AuthenticationResponseMsg struct {
	RES                   []byte
	EAPMessage            []byte
}

type AuthenticationFailureMsg struct {
	Cause5GMM                     uint8
	AuthenticationFailureParameter []byte
}

type SecurityModeCommandMsg struct {
	SelectedNASSecurityAlgorithms uint8
	NgKSI                         uint8
	ReplayedUESecurityCapabilities []byte
	IMEISVRequest                 uint8
	SelectedEPSNASSecurityAlgorithms uint8
	AdditionalSecurityInformation []byte
}

type SecurityModeCompleteMsg struct {
	IMEISV                []byte
	NASMessageContainer   []byte
}

type ConfigurationUpdateCommandMsg struct {
	ConfigurationUpdateIndication uint8
	Guti                          []byte
	TAIList                       []byte
	AllowedNSSAI                  []byte
	ServiceAreaList               []byte
	FullNameForNetwork            []byte
	ShortNameForNetwork           []byte
	LocalTimeZone                 []byte
	NetworkDaylightSavingTime     []byte
	LADNInformation               []byte
}

type ConfigurationUpdateCompleteMsg struct {
}

type GenericUEConfigurationUpdateCommandMsg struct {
	GenericUEConfigurationUpdateIndication uint8
	UERadioCapabilityID                    []byte
	UERadioCapabilityIDDeletionIndication  uint8
	NetworkSlicingIndication               uint8
	OperatorDefinedAccessCategoryDefinitions []byte
	SMSIndication                          uint8
}

type GenericUEConfigurationUpdateCompleteMsg struct {
}

type ServiceRequestMsg struct {
	NgKSI                uint8
	ServiceType          uint8
	TMSI                 uint32
	UplinkDataStatus     []byte
	PDUSessionStatus     []byte
	AllowedPDUSessionStatus []byte
	NASMessageContainer  []byte
}

type ServiceAcceptMsg struct {
	PDUSessionStatus                []byte
	PDUSessionReactivationResult    []byte
	PDUSessionReactivationResultErrorCause []byte
	EAPMessage                      []byte
}

type ServiceRejectMsg struct {
	Cause5GMM            uint8
	PDUSessionStatus     []byte
	T3346Value           []byte
	EAPMessage           []byte
	T3448Value           []byte
}

type DeregistrationRequestMsg struct {
	DeregistrationType   uint8
	Cause5GMM            uint8
}

type IdentityRequestMsg struct {
	IdentityType uint8
}

type IdentityResponseMsg struct {
	MobileIdentity []byte
}

func DecodeNASPDU(data []byte) (*NASPDU, error) {
	if len(data) < 2 {
		return nil, fmt.Errorf("NAS message too short")
	}

	pdu := &NASPDU{}
	offset := 0

	extendedProtocolDiscriminator := data[offset] >> 4
	securityHeaderType := data[offset] & 0x0f
	offset++

	if securityHeaderType != SecurityHeaderTypePlainNAS {
		if len(data) < 7 {
			return nil, fmt.Errorf("secured NAS message too short")
		}
		pdu.SecurityHeaderType = securityHeaderType
		pdu.ProtocolDiscriminator = extendedProtocolDiscriminator
		pdu.SequenceNumber = data[offset]
		offset++
		pdu.MAC = data[offset : offset+4]
		offset += 4
	} else {
		pdu.SecurityHeaderType = securityHeaderType
		pdu.ProtocolDiscriminator = extendedProtocolDiscriminator
	}

	if offset >= len(data) {
		return nil, fmt.Errorf("no message type in NAS PDU")
	}

	pdu.MessageType = data[offset]
	offset++

	pdu.Payload = data[offset:]
	return pdu, nil
}

func EncodeNASPDU(pdu *NASPDU) []byte {
	data := make([]byte, 0)

	firstByte := (pdu.ProtocolDiscriminator << 4) | (pdu.SecurityHeaderType & 0x0f)
	data = append(data, firstByte)

	if pdu.SecurityHeaderType != SecurityHeaderTypePlainNAS {
		data = append(data, pdu.SequenceNumber)
		if len(pdu.MAC) == 4 {
			data = append(data, pdu.MAC...)
		} else {
			data = append(data, 0, 0, 0, 0)
		}
	}

	data = append(data, pdu.MessageType)
	data = append(data, pdu.Payload...)

	return data
}

func DecodeRegistrationRequest(payload []byte) (*RegistrationRequestMsg, error) {
	if len(payload) < 1 {
		return nil, fmt.Errorf("registration request too short")
	}

	msg := &RegistrationRequestMsg{}
	offset := 0

	msg.NgKSI = (payload[offset] >> 4) & 0x0f
	msg.RegistrationType = payload[offset] & 0x0f
	offset++

	for offset < len(payload) {
		if offset+1 >= len(payload) {
			break
		}

		iei := payload[offset]
		offset++

		switch iei {
		case IEISUPIORSUCI:
			if offset+1 >= len(payload) {
				return msg, nil
			}
			length := int(binary.BigEndian.Uint16(payload[offset : offset+2]))
			offset += 2
			if offset+length > len(payload) {
				return nil, fmt.Errorf("invalid mobile identity length")
			}
			msg.MobileIdentity = payload[offset : offset+length]
			offset += length

		case IEIUESecurityCapability:
			if offset >= len(payload) {
				return msg, nil
			}
			length := int(payload[offset])
			offset++
			if offset+length > len(payload) {
				return nil, fmt.Errorf("invalid UE security capability length")
			}
			msg.UESecurityCapability = payload[offset : offset+length]
			offset += length

		case IEIRequestedNSSAI:
			if offset >= len(payload) {
				return msg, nil
			}
			length := int(payload[offset])
			offset++
			if offset+length > len(payload) {
				return nil, fmt.Errorf("invalid requested NSSAI length")
			}
			msg.RequestedNSSAI = payload[offset : offset+length]
			offset += length

		case IEI5GMMCapability:
			if offset >= len(payload) {
				return msg, nil
			}
			length := int(payload[offset])
			offset++
			if offset+length > len(payload) {
				return nil, fmt.Errorf("invalid 5GMM capability length")
			}
			if length > 0 && (payload[offset]&0x01) == 0x01 {
				msg.MicoIndication = true
			}
			offset += length

		case IEIRequestedEDrxParameters:
			if offset >= len(payload) {
				return msg, nil
			}
			length := int(payload[offset])
			offset++
			if offset+length > len(payload) {
				return nil, fmt.Errorf("invalid requested eDRX parameters length")
			}
			msg.RequestedEDrxParameters = payload[offset : offset+length]
			offset += length

		default:
			if offset >= len(payload) {
				return msg, nil
			}
			if iei&0x80 == 0 {
				length := int(payload[offset])
				offset++
				if offset+length > len(payload) {
					return msg, nil
				}
				offset += length
			}
		}
	}

	return msg, nil
}

func DecodeGutiMobileIdentity(mobileIdentity []byte) (*context.Guti, error) {
	if len(mobileIdentity) < 11 {
		return nil, fmt.Errorf("invalid GUTI length: %d", len(mobileIdentity))
	}

	identityType := mobileIdentity[0] & 0x07
	if identityType != 0x02 {
		return nil, fmt.Errorf("not a GUTI (type: 0x%02x)", identityType)
	}

	mcc := fmt.Sprintf("%d%d%d",
		mobileIdentity[1]&0x0f,
		(mobileIdentity[1]>>4)&0x0f,
		mobileIdentity[2]&0x0f)

	var mnc string
	if (mobileIdentity[2]>>4)&0x0f == 0x0f {
		mnc = fmt.Sprintf("%d%d",
			mobileIdentity[3]&0x0f,
			(mobileIdentity[3]>>4)&0x0f)
	} else {
		mnc = fmt.Sprintf("%d%d%d",
			mobileIdentity[3]&0x0f,
			(mobileIdentity[3]>>4)&0x0f,
			(mobileIdentity[2]>>4)&0x0f)
	}

	amfRegionId := fmt.Sprintf("%02x", mobileIdentity[4])
	amfSetId := fmt.Sprintf("%03x", (uint16(mobileIdentity[5])<<2)|uint16((mobileIdentity[6]>>6)&0x03))
	amfPointer := fmt.Sprintf("%02x", mobileIdentity[6]&0x3f)
	tmsi := binary.BigEndian.Uint32(mobileIdentity[7:11])

	return &context.Guti{
		PlmnId: context.PlmnId{
			Mcc: mcc,
			Mnc: mnc,
		},
		AmfRegionId: amfRegionId,
		AmfSetId:    amfSetId,
		AmfPointer:  amfPointer,
		Tmsi:        tmsi,
	}, nil
}

func EncodeGPRSTimer2(seconds int) []byte {
	if seconds <= 0 {
		return nil
	}

	var unit uint8
	var value uint8

	if seconds <= 62 {
		unit = 0x00
		value = uint8(seconds / 2)
	} else if seconds <= 1860 {
		unit = 0x01
		value = uint8(seconds / 60)
	} else if seconds <= 11160 {
		unit = 0x02
		value = uint8(seconds / 360)
	} else {
		unit = 0x02
		value = 31
	}

	timerByte := (unit << 5) | (value & 0x1f)
	return []byte{timerByte}
}

func EncodeGutiMobileIdentity(guti *context.Guti) []byte {
	if guti == nil {
		return nil
	}

	mobileIdentity := make([]byte, 11)

	mobileIdentity[0] = 0xf2

	mcc := []byte(guti.PlmnId.Mcc)
	mnc := []byte(guti.PlmnId.Mnc)

	mobileIdentity[1] = ((mcc[0] - '0') << 0) | ((mcc[1] - '0') << 4)
	if len(mnc) == 2 {
		mobileIdentity[2] = ((mcc[2] - '0') << 0) | 0xf0
		mobileIdentity[3] = ((mnc[0] - '0') << 0) | ((mnc[1] - '0') << 4)
	} else {
		mobileIdentity[2] = ((mcc[2] - '0') << 0) | ((mnc[2] - '0') << 4)
		mobileIdentity[3] = ((mnc[0] - '0') << 0) | ((mnc[1] - '0') << 4)
	}

	amfRegionId := uint8(0)
	if len(guti.AmfRegionId) > 0 {
		fmt.Sscanf(guti.AmfRegionId, "%x", &amfRegionId)
	}
	amfSetId := uint16(0)
	if len(guti.AmfSetId) > 0 {
		fmt.Sscanf(guti.AmfSetId, "%x", &amfSetId)
	}
	amfPointer := uint8(0)
	if len(guti.AmfPointer) > 0 {
		fmt.Sscanf(guti.AmfPointer, "%x", &amfPointer)
	}

	mobileIdentity[4] = amfRegionId
	mobileIdentity[5] = uint8((amfSetId >> 2) & 0xff)
	mobileIdentity[6] = uint8((uint8(amfSetId&0x03) << 6) | (amfPointer & 0x3f))

	binary.BigEndian.PutUint32(mobileIdentity[7:11], guti.Tmsi)

	return mobileIdentity
}

func EncodeRegistrationAccept(msg *RegistrationAcceptMsg) []byte {
	payload := make([]byte, 0)

	payload = append(payload, msg.RegistrationResult)

	if len(msg.MobileIdentity) > 0 {
		payload = append(payload, IEIGUTI)
		lengthBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(lengthBytes, uint16(len(msg.MobileIdentity)))
		payload = append(payload, lengthBytes...)
		payload = append(payload, msg.MobileIdentity...)
	}

	if len(msg.AllowedNSSAI) > 0 {
		payload = append(payload, IEIAllowedNSSAI)
		payload = append(payload, uint8(len(msg.AllowedNSSAI)))
		payload = append(payload, msg.AllowedNSSAI...)
	}

	if len(msg.TAIList) > 0 {
		payload = append(payload, 0x54)
		payload = append(payload, uint8(len(msg.TAIList)))
		payload = append(payload, msg.TAIList...)
	}

	if len(msg.T3512Value) > 0 {
		payload = append(payload, IEIT3512Value)
		payload = append(payload, uint8(len(msg.T3512Value)))
		payload = append(payload, msg.T3512Value...)
	}

	if len(msg.T3502Value) > 0 {
		payload = append(payload, IEIT3502Value)
		payload = append(payload, uint8(len(msg.T3502Value)))
		payload = append(payload, msg.T3502Value...)
	}

	if len(msg.NegotiatedEDrxParameters) > 0 {
		payload = append(payload, IEINegotiatedEDrxParameters)
		payload = append(payload, uint8(len(msg.NegotiatedEDrxParameters)))
		payload = append(payload, msg.NegotiatedEDrxParameters...)
	}

	if msg.MicoIndication {
		payload = append(payload, 0xB0)
	}

	if msg.NetworkSlicingIndication {
		payload = append(payload, 0x90)
	}

	return payload
}

func EncodeRegistrationReject(msg *RegistrationRejectMsg) []byte {
	payload := make([]byte, 0)

	payload = append(payload, msg.Cause5GMM)

	if len(msg.T3502Value) > 0 {
		payload = append(payload, IEIT3502Value)
		payload = append(payload, uint8(len(msg.T3502Value)))
		payload = append(payload, msg.T3502Value...)
	}

	if len(msg.T3346Value) > 0 {
		payload = append(payload, 0x5f)
		payload = append(payload, uint8(len(msg.T3346Value)))
		payload = append(payload, msg.T3346Value...)
	}

	if len(msg.EAPMessage) > 0 {
		payload = append(payload, 0x78)
		lengthBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(lengthBytes, uint16(len(msg.EAPMessage)))
		payload = append(payload, lengthBytes...)
		payload = append(payload, msg.EAPMessage...)
	}

	return payload
}

func EncodeAuthenticationRequest(msg *AuthenticationRequestMsg) []byte {
	payload := make([]byte, 0)

	ngksiAndSpareHalf := (msg.NgKSI << 4) | 0x0f
	payload = append(payload, ngksiAndSpareHalf)

	if len(msg.ABBA) > 0 {
		payload = append(payload, uint8(len(msg.ABBA)))
		payload = append(payload, msg.ABBA...)
	} else {
		payload = append(payload, 0x02, 0x00, 0x00)
	}

	if len(msg.RAND) == 16 {
		payload = append(payload, 0x21)
		payload = append(payload, msg.RAND...)
	}

	if len(msg.AUTN) > 0 {
		payload = append(payload, 0x20)
		payload = append(payload, uint8(len(msg.AUTN)))
		payload = append(payload, msg.AUTN...)
	}

	if len(msg.EAPMessage) > 0 {
		payload = append(payload, IEIEAPMessage)
		lengthBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(lengthBytes, uint16(len(msg.EAPMessage)))
		payload = append(payload, lengthBytes...)
		payload = append(payload, msg.EAPMessage...)
	}

	return payload
}

func DecodeAuthenticationResponse(payload []byte) (*AuthenticationResponseMsg, error) {
	if len(payload) < 1 {
		return nil, fmt.Errorf("authentication response too short")
	}

	msg := &AuthenticationResponseMsg{}
	offset := 0

	for offset < len(payload) {
		if offset >= len(payload) {
			break
		}

		iei := payload[offset]
		offset++

		switch iei {
		case 0x2d:
			if offset >= len(payload) {
				return msg, nil
			}
			length := int(payload[offset])
			offset++
			if offset+length > len(payload) {
				return nil, fmt.Errorf("invalid RES length")
			}
			msg.RES = payload[offset : offset+length]
			offset += length

		case IEIEAPMessage:
			if offset+1 >= len(payload) {
				return msg, nil
			}
			length := int(binary.BigEndian.Uint16(payload[offset : offset+2]))
			offset += 2
			if offset+length > len(payload) {
				return nil, fmt.Errorf("invalid EAP message length")
			}
			msg.EAPMessage = payload[offset : offset+length]
			offset += length

		default:
			if offset >= len(payload) {
				return msg, nil
			}
			if iei&0x80 == 0 {
				length := int(payload[offset])
				offset++
				if offset+length > len(payload) {
					return msg, nil
				}
				offset += length
			}
		}
	}

	return msg, nil
}

func DecodeAuthenticationFailure(payload []byte) (*AuthenticationFailureMsg, error) {
	if len(payload) < 1 {
		return nil, fmt.Errorf("authentication failure too short")
	}

	msg := &AuthenticationFailureMsg{}
	offset := 0

	msg.Cause5GMM = payload[offset]
	offset++

	for offset < len(payload) {
		if offset >= len(payload) {
			break
		}

		iei := payload[offset]
		offset++

		switch iei {
		case 0x30:
			if offset >= len(payload) {
				return msg, nil
			}
			length := int(payload[offset])
			offset++
			if offset+length > len(payload) {
				return nil, fmt.Errorf("invalid authentication failure parameter length")
			}
			msg.AuthenticationFailureParameter = payload[offset : offset+length]
			offset += length

		default:
			if offset >= len(payload) {
				return msg, nil
			}
			if iei&0x80 == 0 {
				length := int(payload[offset])
				offset++
				if offset+length > len(payload) {
					return msg, nil
				}
				offset += length
			}
		}
	}

	return msg, nil
}

func EncodeSecurityModeCommand(msg *SecurityModeCommandMsg) []byte {
	payload := make([]byte, 0)

	payload = append(payload, msg.SelectedNASSecurityAlgorithms)

	ngksiAndSpareHalf := (msg.NgKSI << 4) | 0x0f
	payload = append(payload, ngksiAndSpareHalf)

	if len(msg.ReplayedUESecurityCapabilities) > 0 {
		payload = append(payload, IEIUESecurityCapability)
		payload = append(payload, uint8(len(msg.ReplayedUESecurityCapabilities)))
		payload = append(payload, msg.ReplayedUESecurityCapabilities...)
	}

	if msg.IMEISVRequest > 0 {
		imeisv := 0xe0 | (msg.IMEISVRequest & 0x0f)
		payload = append(payload, uint8(imeisv))
	}

	return payload
}

func DecodeSecurityModeComplete(payload []byte) (*SecurityModeCompleteMsg, error) {
	msg := &SecurityModeCompleteMsg{}
	offset := 0

	for offset < len(payload) {
		if offset >= len(payload) {
			break
		}

		iei := payload[offset]
		offset++

		switch iei {
		case 0x77:
			if offset+1 >= len(payload) {
				return msg, nil
			}
			length := int(binary.BigEndian.Uint16(payload[offset : offset+2]))
			offset += 2
			if offset+length > len(payload) {
				return nil, fmt.Errorf("invalid IMEISV length")
			}
			msg.IMEISV = payload[offset : offset+length]
			offset += length

		case 0x71:
			if offset >= len(payload) {
				return msg, nil
			}
			length := int(payload[offset])
			offset++
			if offset+length > len(payload) {
				return nil, fmt.Errorf("invalid NAS message container length")
			}
			msg.NASMessageContainer = payload[offset : offset+length]
			offset += length

		default:
			if offset >= len(payload) {
				return msg, nil
			}
			if iei&0x80 == 0 {
				length := int(payload[offset])
				offset++
				if offset+length > len(payload) {
					return msg, nil
				}
				offset += length
			}
		}
	}

	return msg, nil
}

func EncodeConfigurationUpdateCommand(msg *ConfigurationUpdateCommandMsg) []byte {
	payload := make([]byte, 0)

	if msg.ConfigurationUpdateIndication > 0 {
		payload = append(payload, 0xd0|(msg.ConfigurationUpdateIndication&0x0f))
	}

	if len(msg.Guti) > 0 {
		payload = append(payload, IEIGUTI)
		lengthBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(lengthBytes, uint16(len(msg.Guti)))
		payload = append(payload, lengthBytes...)
		payload = append(payload, msg.Guti...)
	}

	if len(msg.TAIList) > 0 {
		payload = append(payload, 0x54)
		payload = append(payload, uint8(len(msg.TAIList)))
		payload = append(payload, msg.TAIList...)
	}

	if len(msg.AllowedNSSAI) > 0 {
		payload = append(payload, IEIAllowedNSSAI)
		payload = append(payload, uint8(len(msg.AllowedNSSAI)))
		payload = append(payload, msg.AllowedNSSAI...)
	}

	if len(msg.ServiceAreaList) > 0 {
		payload = append(payload, 0x27)
		lengthBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(lengthBytes, uint16(len(msg.ServiceAreaList)))
		payload = append(payload, lengthBytes...)
		payload = append(payload, msg.ServiceAreaList...)
	}

	if len(msg.FullNameForNetwork) > 0 {
		payload = append(payload, 0x43)
		payload = append(payload, uint8(len(msg.FullNameForNetwork)))
		payload = append(payload, msg.FullNameForNetwork...)
	}

	if len(msg.ShortNameForNetwork) > 0 {
		payload = append(payload, 0x45)
		payload = append(payload, uint8(len(msg.ShortNameForNetwork)))
		payload = append(payload, msg.ShortNameForNetwork...)
	}

	if len(msg.LocalTimeZone) > 0 {
		payload = append(payload, 0x46)
		payload = append(payload, msg.LocalTimeZone...)
	}

	if len(msg.NetworkDaylightSavingTime) > 0 {
		payload = append(payload, 0x49)
		payload = append(payload, uint8(len(msg.NetworkDaylightSavingTime)))
		payload = append(payload, msg.NetworkDaylightSavingTime...)
	}

	if len(msg.LADNInformation) > 0 {
		payload = append(payload, 0x79)
		lengthBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(lengthBytes, uint16(len(msg.LADNInformation)))
		payload = append(payload, lengthBytes...)
		payload = append(payload, msg.LADNInformation...)
	}

	return payload
}

func EncodeGenericUEConfigurationUpdateCommand(msg *GenericUEConfigurationUpdateCommandMsg) []byte {
	payload := make([]byte, 0)

	payload = append(payload, 0xd0|(msg.GenericUEConfigurationUpdateIndication&0x0f))

	if len(msg.UERadioCapabilityID) > 0 {
		payload = append(payload, 0x67)
		lengthBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(lengthBytes, uint16(len(msg.UERadioCapabilityID)))
		payload = append(payload, lengthBytes...)
		payload = append(payload, msg.UERadioCapabilityID...)
	}

	if msg.UERadioCapabilityIDDeletionIndication > 0 {
		payload = append(payload, 0xa0|(msg.UERadioCapabilityIDDeletionIndication&0x0f))
	}

	if msg.NetworkSlicingIndication > 0 {
		payload = append(payload, 0x90|(msg.NetworkSlicingIndication&0x0f))
	}

	if len(msg.OperatorDefinedAccessCategoryDefinitions) > 0 {
		payload = append(payload, 0x76)
		lengthBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(lengthBytes, uint16(len(msg.OperatorDefinedAccessCategoryDefinitions)))
		payload = append(payload, lengthBytes...)
		payload = append(payload, msg.OperatorDefinedAccessCategoryDefinitions...)
	}

	if msg.SMSIndication > 0 {
		payload = append(payload, 0xf0|(msg.SMSIndication&0x0f))
	}

	return payload
}

func EncodeLocalTimeZone(offsetMinutes int) []byte {
	quarters := offsetMinutes / 15

	absQuarters := quarters
	sign := byte(0)
	if quarters < 0 {
		sign = 0x08
		absQuarters = -quarters
	}

	value := byte(absQuarters & 0x7F)
	return []byte{value | sign}
}

func EncodeNetworkDaylightSavingTime(dstValue int) []byte {
	if dstValue < 0 || dstValue > 2 {
		dstValue = 0
	}
	return []byte{byte(len([]byte{byte(dstValue)})), byte(dstValue)}
}

func EncodePLMN(mcc, mnc string) []byte {
	plmnBytes := make([]byte, 3)

	mccBytes := []byte(mcc)
	mncBytes := []byte(mnc)

	plmnBytes[0] = ((mccBytes[0] - '0') << 0) | ((mccBytes[1] - '0') << 4)
	if len(mnc) == 2 {
		plmnBytes[1] = ((mccBytes[2] - '0') << 0) | 0xf0
		plmnBytes[2] = ((mncBytes[0] - '0') << 0) | ((mncBytes[1] - '0') << 4)
	} else {
		plmnBytes[1] = ((mccBytes[2] - '0') << 0) | ((mncBytes[2] - '0') << 4)
		plmnBytes[2] = ((mncBytes[0] - '0') << 0) | ((mncBytes[1] - '0') << 4)
	}

	return plmnBytes
}

func EncodeServiceAreaList(taiList []context.Tai) []byte {
	if len(taiList) == 0 {
		return nil
	}

	serviceAreaMap := make(map[string][]string)

	for _, tai := range taiList {
		plmnKey := tai.PlmnId.Mcc + tai.PlmnId.Mnc
		serviceAreaMap[plmnKey] = append(serviceAreaMap[plmnKey], tai.Tac)
	}

	payload := make([]byte, 0)

	for plmnKey, tacs := range serviceAreaMap {
		if len(tacs) > 15 {
			tacs = tacs[:15]
		}

		listType := byte(0x00)
		numElements := byte(len(tacs))
		payload = append(payload, (listType<<6)|numElements)

		mcc := plmnKey[0:3]
		mnc := plmnKey[3:]
		plmnBytes := EncodePLMN(mcc, mnc)
		payload = append(payload, plmnBytes...)

		for _, tac := range tacs {
			tacBytes := make([]byte, 3)
			for i := 0; i < 3 && i < len(tac); i++ {
				if tac[i] >= '0' && tac[i] <= '9' {
					tacBytes[i] = tac[i] - '0'
				} else if tac[i] >= 'A' && tac[i] <= 'F' {
					tacBytes[i] = tac[i] - 'A' + 10
				} else if tac[i] >= 'a' && tac[i] <= 'f' {
					tacBytes[i] = tac[i] - 'a' + 10
				}
			}
			tacValue := (uint32(tacBytes[0]) << 16) | (uint32(tacBytes[1]) << 8) | uint32(tacBytes[2])
			payload = append(payload, byte(tacValue>>16), byte(tacValue>>8), byte(tacValue))
		}
	}

	return payload
}

func DecodeConfigurationUpdateComplete(payload []byte) (*ConfigurationUpdateCompleteMsg, error) {
	msg := &ConfigurationUpdateCompleteMsg{}
	return msg, nil
}

func DecodeGenericUEConfigurationUpdateComplete(payload []byte) (*GenericUEConfigurationUpdateCompleteMsg, error) {
	msg := &GenericUEConfigurationUpdateCompleteMsg{}
	return msg, nil
}

func DecodeServiceRequest(payload []byte) (*ServiceRequestMsg, error) {
	if len(payload) < 1 {
		return nil, fmt.Errorf("service request too short")
	}

	msg := &ServiceRequestMsg{}
	offset := 0

	msg.NgKSI = (payload[offset] >> 4) & 0x0f
	msg.ServiceType = payload[offset] & 0x0f
	offset++

	if offset+4 <= len(payload) {
		msg.TMSI = binary.BigEndian.Uint32(payload[offset : offset+4])
		offset += 4
	}

	for offset < len(payload) {
		if offset >= len(payload) {
			break
		}

		iei := payload[offset]
		offset++

		switch iei {
		case 0x40:
			if offset >= len(payload) {
				return msg, nil
			}
			length := int(binary.BigEndian.Uint16(payload[offset : offset+2]))
			offset += 2
			if offset+length > len(payload) {
				return nil, fmt.Errorf("invalid uplink data status length")
			}
			msg.UplinkDataStatus = payload[offset : offset+length]
			offset += length

		case IEIPDUSessionStatus:
			if offset >= len(payload) {
				return msg, nil
			}
			length := int(binary.BigEndian.Uint16(payload[offset : offset+2]))
			offset += 2
			if offset+length > len(payload) {
				return nil, fmt.Errorf("invalid PDU session status length")
			}
			msg.PDUSessionStatus = payload[offset : offset+length]
			offset += length

		case 0x25:
			if offset >= len(payload) {
				return msg, nil
			}
			length := int(binary.BigEndian.Uint16(payload[offset : offset+2]))
			offset += 2
			if offset+length > len(payload) {
				return nil, fmt.Errorf("invalid allowed PDU session status length")
			}
			msg.AllowedPDUSessionStatus = payload[offset : offset+length]
			offset += length

		case 0x71:
			if offset >= len(payload) {
				return msg, nil
			}
			length := int(binary.BigEndian.Uint16(payload[offset : offset+2]))
			offset += 2
			if offset+length > len(payload) {
				return nil, fmt.Errorf("invalid NAS message container length")
			}
			msg.NASMessageContainer = payload[offset : offset+length]
			offset += length

		default:
			if offset >= len(payload) {
				return msg, nil
			}
			if iei&0x80 == 0 {
				if offset >= len(payload) {
					return msg, nil
				}
				length := int(payload[offset])
				offset++
				if offset+length > len(payload) {
					return msg, nil
				}
				offset += length
			}
		}
	}

	return msg, nil
}

func EncodeServiceAccept(msg *ServiceAcceptMsg) []byte {
	payload := make([]byte, 0)

	if len(msg.PDUSessionStatus) > 0 {
		payload = append(payload, IEIPDUSessionStatus)
		lengthBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(lengthBytes, uint16(len(msg.PDUSessionStatus)))
		payload = append(payload, lengthBytes...)
		payload = append(payload, msg.PDUSessionStatus...)
	}

	if len(msg.PDUSessionReactivationResult) > 0 {
		payload = append(payload, 0x26)
		lengthBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(lengthBytes, uint16(len(msg.PDUSessionReactivationResult)))
		payload = append(payload, lengthBytes...)
		payload = append(payload, msg.PDUSessionReactivationResult...)
	}

	if len(msg.PDUSessionReactivationResultErrorCause) > 0 {
		payload = append(payload, 0x72)
		lengthBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(lengthBytes, uint16(len(msg.PDUSessionReactivationResultErrorCause)))
		payload = append(payload, lengthBytes...)
		payload = append(payload, msg.PDUSessionReactivationResultErrorCause...)
	}

	if len(msg.EAPMessage) > 0 {
		payload = append(payload, IEIEAPMessage)
		lengthBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(lengthBytes, uint16(len(msg.EAPMessage)))
		payload = append(payload, lengthBytes...)
		payload = append(payload, msg.EAPMessage...)
	}

	return payload
}

func EncodeServiceReject(msg *ServiceRejectMsg) []byte {
	payload := make([]byte, 0)

	payload = append(payload, msg.Cause5GMM)

	if len(msg.PDUSessionStatus) > 0 {
		payload = append(payload, IEIPDUSessionStatus)
		lengthBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(lengthBytes, uint16(len(msg.PDUSessionStatus)))
		payload = append(payload, lengthBytes...)
		payload = append(payload, msg.PDUSessionStatus...)
	}

	if len(msg.T3346Value) > 0 {
		payload = append(payload, 0x5f)
		payload = append(payload, uint8(len(msg.T3346Value)))
		payload = append(payload, msg.T3346Value...)
	}

	if len(msg.EAPMessage) > 0 {
		payload = append(payload, IEIEAPMessage)
		lengthBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(lengthBytes, uint16(len(msg.EAPMessage)))
		payload = append(payload, lengthBytes...)
		payload = append(payload, msg.EAPMessage...)
	}

	if len(msg.T3448Value) > 0 {
		payload = append(payload, 0x6b)
		payload = append(payload, uint8(len(msg.T3448Value)))
		payload = append(payload, msg.T3448Value...)
	}

	return payload
}

func EncodeDeregistrationRequest(msg *DeregistrationRequestMsg) []byte {
	payload := make([]byte, 0)

	deregTypeAndSpare := (msg.DeregistrationType & 0x0f)
	payload = append(payload, deregTypeAndSpare)

	if msg.Cause5GMM > 0 {
		payload = append(payload, 0x58)
		payload = append(payload, uint8(1))
		payload = append(payload, msg.Cause5GMM)
	}

	return payload
}

func EncodeIdentityRequest(msg *IdentityRequestMsg) []byte {
	payload := make([]byte, 0)
	identityTypeAndSpare := (msg.IdentityType & 0x0f)
	payload = append(payload, identityTypeAndSpare)
	return payload
}

func DecodeIdentityResponse(payload []byte) (*IdentityResponseMsg, error) {
	if len(payload) < 1 {
		return nil, fmt.Errorf("identity response too short")
	}

	msg := &IdentityResponseMsg{}
	offset := 0

	for offset < len(payload) {
		if offset >= len(payload) {
			break
		}

		iei := payload[offset]
		offset++

		switch iei {
		case IEISUPIORSUCI:
			if offset+1 >= len(payload) {
				return msg, nil
			}
			length := int(binary.BigEndian.Uint16(payload[offset : offset+2]))
			offset += 2
			if offset+length > len(payload) {
				return nil, fmt.Errorf("invalid mobile identity length")
			}
			msg.MobileIdentity = payload[offset : offset+length]
			offset += length

		default:
			if offset >= len(payload) {
				return msg, nil
			}
			if iei&0x80 == 0 {
				length := int(payload[offset])
				offset++
				if offset+length > len(payload) {
					return msg, nil
				}
				offset += length
			}
		}
	}

	return msg, nil
}
