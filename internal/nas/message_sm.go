package nas

import (
	"encoding/binary"
	"fmt"
)

type ULNASTransportMsg struct {
	PayloadContainerType uint8
	PayloadContainer     []byte
	PDUSessionID         uint8
	OldPDUSessionID      uint8
	RequestType          uint8
	SNSSAI               []byte
	DNN                  []byte
	AdditionalInfo       []byte
}

type DLNASTransportMsg struct {
	PayloadContainerType uint8
	PayloadContainer     []byte
	PDUSessionID         uint8
	AdditionalInfo       []byte
	Cause5GMM            uint8
	BackoffTimer         []byte
}

type PDUSessionEstablishmentRequestMsg struct {
	IntegrityProtectionMaximumDataRate uint8
	PDUSessionType                      uint8
	SSCMode                             uint8
	Capability5GSM                      []byte
	MaximumNumberOfSupportedPacketFilters uint8
	AlwaysOnPDUSessionRequested         uint8
	SMPDUDNRequestContainer             []byte
	ExtendedProtocolConfigurationOptions []byte
	MPTCP                               uint8
}

type PDUSessionEstablishmentAcceptMsg struct {
	PDUSessionType                      uint8
	SSCMode                             uint8
	QoSRules                            []byte
	SessionAMBR                         []byte
	Cause5GSM                           uint8
	PDUAddress                          []byte
	SNSSAl                              []byte
	AlwaysOnPDUSessionIndication        uint8
	MappedEPSBearerContexts             []byte
	EAPMessage                          []byte
	QoSFlowDescriptions                 []byte
	ExtendedProtocolConfigurationOptions []byte
	DNN                                 []byte
	MPTCP                               uint8
}

type PDUSessionEstablishmentRejectMsg struct {
	Cause5GSM                            uint8
	BackoffTimer                         []byte
	AllowedSSCMode                       uint8
	EAPMessage                           []byte
	ExtendedProtocolConfigurationOptions []byte
}

type PDUSessionModificationRequestMsg struct {
	Capability5GSM                      []byte
	Cause5GSM                           uint8
	MaximumNumberOfSupportedPacketFilters uint8
	AlwaysOnPDUSessionRequested         uint8
	IntegrityProtectionMaximumDataRate  []byte
	RequestedQoSRules                   []byte
	RequestedQoSFlowDescriptions        []byte
	MappedEPSBearerContexts             []byte
	ExtendedProtocolConfigurationOptions []byte
}

type PDUSessionModificationCommandMsg struct {
	Cause5GSM                            uint8
	SessionAMBR                          []byte
	QoSRules                             []byte
	QoSFlowDescriptions                  []byte
	MappedEPSBearerContexts              []byte
	ExtendedProtocolConfigurationOptions []byte
}

type PDUSessionReleaseRequestMsg struct {
	Cause5GSM                            uint8
	ExtendedProtocolConfigurationOptions []byte
}

type PDUSessionReleaseCommandMsg struct {
	Cause5GSM                            uint8
	BackoffTimer                         []byte
	ExtendedProtocolConfigurationOptions []byte
}

type PDUSessionReleaseCompleteMsg struct {
	Cause5GSM                            uint8
	ExtendedProtocolConfigurationOptions []byte
}

type FiveGSMStatusMsg struct {
	Cause5GSM uint8
}

type PDUSessionAuthenticationCommandMsg struct {
	EAPMessage []byte
}

type PDUSessionAuthenticationCompleteMsg struct {
	EAPMessage []byte
}

func DecodeULNASTransport(payload []byte) (*ULNASTransportMsg, error) {
	if len(payload) < 2 {
		return nil, fmt.Errorf("UL NAS transport too short")
	}

	msg := &ULNASTransportMsg{}
	offset := 0

	msg.PayloadContainerType = payload[offset] & 0x0f
	offset++

	for offset < len(payload) {
		if offset >= len(payload) {
			break
		}

		iei := payload[offset]
		offset++

		switch iei {
		case IEIPayloadContainer:
			if offset+1 >= len(payload) {
				return msg, nil
			}
			length := int(binary.BigEndian.Uint16(payload[offset : offset+2]))
			offset += 2
			if offset+length > len(payload) {
				return nil, fmt.Errorf("invalid payload container length")
			}
			msg.PayloadContainer = payload[offset : offset+length]
			offset += length

		case IEIPDUSessionID:
			if offset >= len(payload) {
				return msg, nil
			}
			msg.PDUSessionID = payload[offset]
			offset++

		case 0x59:
			if offset >= len(payload) {
				return msg, nil
			}
			msg.OldPDUSessionID = payload[offset]
			offset++

		case 0x8:
			if offset >= len(payload) {
				return msg, nil
			}
			msg.RequestType = payload[offset] & 0x07
			offset++

		case IEISSNSSAl:
			if offset >= len(payload) {
				return msg, nil
			}
			length := int(payload[offset])
			offset++
			if offset+length > len(payload) {
				return nil, fmt.Errorf("invalid S-NSSAI length")
			}
			msg.SNSSAI = payload[offset : offset+length]
			offset += length

		case IEIDNN:
			if offset >= len(payload) {
				return msg, nil
			}
			length := int(payload[offset])
			offset++
			if offset+length > len(payload) {
				return nil, fmt.Errorf("invalid DNN length")
			}
			msg.DNN = payload[offset : offset+length]
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

func EncodeDLNASTransport(msg *DLNASTransportMsg) []byte {
	payload := make([]byte, 0)

	payload = append(payload, msg.PayloadContainerType&0x0f)

	if len(msg.PayloadContainer) > 0 {
		payload = append(payload, IEIPayloadContainer)
		lengthBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(lengthBytes, uint16(len(msg.PayloadContainer)))
		payload = append(payload, lengthBytes...)
		payload = append(payload, msg.PayloadContainer...)
	}

	if msg.PDUSessionID > 0 {
		payload = append(payload, IEIPDUSessionID)
		payload = append(payload, msg.PDUSessionID)
	}

	if msg.Cause5GMM > 0 {
		payload = append(payload, 0x58)
		payload = append(payload, msg.Cause5GMM)
	}

	if len(msg.BackoffTimer) > 0 {
		payload = append(payload, 0x37)
		payload = append(payload, uint8(len(msg.BackoffTimer)))
		payload = append(payload, msg.BackoffTimer...)
	}

	return payload
}

func DecodePDUSessionEstablishmentRequest(payload []byte) (*PDUSessionEstablishmentRequestMsg, error) {
	msg := &PDUSessionEstablishmentRequestMsg{}
	offset := 0

	if offset >= len(payload) {
		return msg, nil
	}

	msg.IntegrityProtectionMaximumDataRate = payload[offset]
	offset++

	for offset < len(payload) {
		if offset >= len(payload) {
			break
		}

		iei := payload[offset]
		offset++

		switch iei {
		case 0x09:
			if offset >= len(payload) {
				return msg, nil
			}
			msg.PDUSessionType = payload[offset] & 0x07
			offset++

		case 0x0a:
			if offset >= len(payload) {
				return msg, nil
			}
			msg.SSCMode = payload[offset] & 0x07
			offset++

		case 0x0b:
			if offset >= len(payload) {
				return msg, nil
			}
			msg.AlwaysOnPDUSessionRequested = payload[offset] & 0x01
			offset++

		case 0x06:
			if offset >= len(payload) {
				return msg, nil
			}
			msg.MPTCP = payload[offset] & 0x0f
			offset++

		case 0x28:
			if offset >= len(payload) {
				return msg, nil
			}
			length := int(payload[offset])
			offset++
			if offset+length > len(payload) {
				return nil, fmt.Errorf("invalid 5GSM capability length")
			}
			msg.Capability5GSM = payload[offset : offset+length]
			offset += length

		case 0x55:
			if offset >= len(payload) {
				return msg, nil
			}
			length := int(payload[offset])
			offset++
			offset += length

		case 0x7b:
			if offset+1 >= len(payload) {
				return msg, nil
			}
			length := int(binary.BigEndian.Uint16(payload[offset : offset+2]))
			offset += 2
			if offset+length > len(payload) {
				return nil, fmt.Errorf("invalid extended protocol configuration options length")
			}
			msg.ExtendedProtocolConfigurationOptions = payload[offset : offset+length]
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

func EncodePDUSessionEstablishmentAccept(msg *PDUSessionEstablishmentAcceptMsg) []byte {
	payload := make([]byte, 0)

	payload = append(payload, (msg.PDUSessionType&0x07)|(msg.SSCMode<<4))

	if len(msg.QoSRules) > 0 {
		payload = append(payload, 0x79)
		lengthBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(lengthBytes, uint16(len(msg.QoSRules)))
		payload = append(payload, lengthBytes...)
		payload = append(payload, msg.QoSRules...)
	}

	if len(msg.SessionAMBR) > 0 {
		payload = append(payload, 0x2a)
		payload = append(payload, uint8(len(msg.SessionAMBR)))
		payload = append(payload, msg.SessionAMBR...)
	}

	if len(msg.PDUAddress) > 0 {
		payload = append(payload, 0x29)
		payload = append(payload, uint8(len(msg.PDUAddress)))
		payload = append(payload, msg.PDUAddress...)
	}

	if len(msg.SNSSAl) > 0 {
		payload = append(payload, IEISSNSSAl)
		payload = append(payload, uint8(len(msg.SNSSAl)))
		payload = append(payload, msg.SNSSAl...)
	}

	if msg.AlwaysOnPDUSessionIndication > 0 {
		payload = append(payload, 0x08)
		payload = append(payload, msg.AlwaysOnPDUSessionIndication&0x01)
	}

	if len(msg.QoSFlowDescriptions) > 0 {
		payload = append(payload, 0x79)
		lengthBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(lengthBytes, uint16(len(msg.QoSFlowDescriptions)))
		payload = append(payload, lengthBytes...)
		payload = append(payload, msg.QoSFlowDescriptions...)
	}

	if len(msg.ExtendedProtocolConfigurationOptions) > 0 {
		payload = append(payload, 0x7b)
		lengthBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(lengthBytes, uint16(len(msg.ExtendedProtocolConfigurationOptions)))
		payload = append(payload, lengthBytes...)
		payload = append(payload, msg.ExtendedProtocolConfigurationOptions...)
	}

	if len(msg.DNN) > 0 {
		payload = append(payload, IEIDNN)
		payload = append(payload, uint8(len(msg.DNN)))
		payload = append(payload, msg.DNN...)
	}

	if msg.MPTCP > 0 {
		payload = append(payload, 0x69)
		payload = append(payload, 0x03)
		payload = append(payload, msg.MPTCP)
	}

	return payload
}

func EncodePDUSessionEstablishmentReject(msg *PDUSessionEstablishmentRejectMsg) []byte {
	payload := make([]byte, 0)

	payload = append(payload, msg.Cause5GSM)

	if len(msg.BackoffTimer) > 0 {
		payload = append(payload, 0x37)
		payload = append(payload, uint8(len(msg.BackoffTimer)))
		payload = append(payload, msg.BackoffTimer...)
	}

	if msg.AllowedSSCMode > 0 {
		payload = append(payload, 0x0f)
		payload = append(payload, msg.AllowedSSCMode&0x07)
	}

	if len(msg.EAPMessage) > 0 {
		payload = append(payload, 0x78)
		lengthBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(lengthBytes, uint16(len(msg.EAPMessage)))
		payload = append(payload, lengthBytes...)
		payload = append(payload, msg.EAPMessage...)
	}

	if len(msg.ExtendedProtocolConfigurationOptions) > 0 {
		payload = append(payload, 0x7b)
		lengthBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(lengthBytes, uint16(len(msg.ExtendedProtocolConfigurationOptions)))
		payload = append(payload, lengthBytes...)
		payload = append(payload, msg.ExtendedProtocolConfigurationOptions...)
	}

	return payload
}

func DecodePDUSessionModificationRequest(payload []byte) (*PDUSessionModificationRequestMsg, error) {
	msg := &PDUSessionModificationRequestMsg{}
	offset := 0

	for offset < len(payload) {
		if offset >= len(payload) {
			break
		}

		iei := payload[offset]
		offset++

		switch iei {
		case 0x59:
			if offset >= len(payload) {
				return msg, nil
			}
			msg.Cause5GSM = payload[offset]
			offset++

		case 0x28:
			if offset >= len(payload) {
				return msg, nil
			}
			length := int(payload[offset])
			offset++
			if offset+length > len(payload) {
				return nil, fmt.Errorf("invalid 5GSM capability length")
			}
			msg.Capability5GSM = payload[offset : offset+length]
			offset += length

		case 0x55:
			if offset >= len(payload) {
				return msg, nil
			}
			length := int(payload[offset])
			offset++
			if offset+length > len(payload) {
				return nil, fmt.Errorf("invalid maximum number of packet filters length")
			}
			offset += length

		case 0x13:
			if offset >= len(payload) {
				return msg, nil
			}
			length := int(payload[offset])
			offset++
			if offset+length > len(payload) {
				return nil, fmt.Errorf("invalid integrity protection max data rate length")
			}
			msg.IntegrityProtectionMaximumDataRate = payload[offset : offset+length]
			offset += length

		case 0x7a:
			if offset+1 >= len(payload) {
				return msg, nil
			}
			length := int(binary.BigEndian.Uint16(payload[offset : offset+2]))
			offset += 2
			if offset+length > len(payload) {
				return nil, fmt.Errorf("invalid requested QoS rules length")
			}
			msg.RequestedQoSRules = payload[offset : offset+length]
			offset += length

		case 0x79:
			if offset+1 >= len(payload) {
				return msg, nil
			}
			length := int(binary.BigEndian.Uint16(payload[offset : offset+2]))
			offset += 2
			if offset+length > len(payload) {
				return nil, fmt.Errorf("invalid requested QoS flow descriptions length")
			}
			msg.RequestedQoSFlowDescriptions = payload[offset : offset+length]
			offset += length

		case 0x7f:
			if offset+1 >= len(payload) {
				return msg, nil
			}
			length := int(binary.BigEndian.Uint16(payload[offset : offset+2]))
			offset += 2
			if offset+length > len(payload) {
				return nil, fmt.Errorf("invalid mapped EPS bearer contexts length")
			}
			msg.MappedEPSBearerContexts = payload[offset : offset+length]
			offset += length

		case 0x7b:
			if offset+1 >= len(payload) {
				return msg, nil
			}
			length := int(binary.BigEndian.Uint16(payload[offset : offset+2]))
			offset += 2
			if offset+length > len(payload) {
				return nil, fmt.Errorf("invalid extended protocol configuration options length")
			}
			msg.ExtendedProtocolConfigurationOptions = payload[offset : offset+length]
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

func EncodePDUSessionModificationCommand(msg *PDUSessionModificationCommandMsg) []byte {
	payload := make([]byte, 0)

	if msg.Cause5GSM > 0 {
		payload = append(payload, 0x59)
		payload = append(payload, msg.Cause5GSM)
	}

	if len(msg.SessionAMBR) > 0 {
		payload = append(payload, 0x2a)
		payload = append(payload, uint8(len(msg.SessionAMBR)))
		payload = append(payload, msg.SessionAMBR...)
	}

	if len(msg.QoSRules) > 0 {
		payload = append(payload, 0x7a)
		lengthBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(lengthBytes, uint16(len(msg.QoSRules)))
		payload = append(payload, lengthBytes...)
		payload = append(payload, msg.QoSRules...)
	}

	if len(msg.QoSFlowDescriptions) > 0 {
		payload = append(payload, 0x79)
		lengthBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(lengthBytes, uint16(len(msg.QoSFlowDescriptions)))
		payload = append(payload, lengthBytes...)
		payload = append(payload, msg.QoSFlowDescriptions...)
	}

	if len(msg.MappedEPSBearerContexts) > 0 {
		payload = append(payload, 0x7f)
		lengthBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(lengthBytes, uint16(len(msg.MappedEPSBearerContexts)))
		payload = append(payload, lengthBytes...)
		payload = append(payload, msg.MappedEPSBearerContexts...)
	}

	if len(msg.ExtendedProtocolConfigurationOptions) > 0 {
		payload = append(payload, 0x7b)
		lengthBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(lengthBytes, uint16(len(msg.ExtendedProtocolConfigurationOptions)))
		payload = append(payload, lengthBytes...)
		payload = append(payload, msg.ExtendedProtocolConfigurationOptions...)
	}

	return payload
}

func DecodePDUSessionReleaseRequest(payload []byte) (*PDUSessionReleaseRequestMsg, error) {
	msg := &PDUSessionReleaseRequestMsg{}
	offset := 0

	for offset < len(payload) {
		if offset >= len(payload) {
			break
		}

		iei := payload[offset]
		offset++

		switch iei {
		case 0x59:
			if offset >= len(payload) {
				return msg, nil
			}
			msg.Cause5GSM = payload[offset]
			offset++

		case 0x7b:
			if offset+1 >= len(payload) {
				return msg, nil
			}
			length := int(binary.BigEndian.Uint16(payload[offset : offset+2]))
			offset += 2
			if offset+length > len(payload) {
				return nil, fmt.Errorf("invalid extended protocol configuration options length")
			}
			msg.ExtendedProtocolConfigurationOptions = payload[offset : offset+length]
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

func EncodePDUSessionReleaseCommand(msg *PDUSessionReleaseCommandMsg) []byte {
	payload := make([]byte, 0)

	payload = append(payload, msg.Cause5GSM)

	if len(msg.BackoffTimer) > 0 {
		payload = append(payload, 0x37)
		payload = append(payload, uint8(len(msg.BackoffTimer)))
		payload = append(payload, msg.BackoffTimer...)
	}

	if len(msg.ExtendedProtocolConfigurationOptions) > 0 {
		payload = append(payload, 0x7b)
		lengthBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(lengthBytes, uint16(len(msg.ExtendedProtocolConfigurationOptions)))
		payload = append(payload, lengthBytes...)
		payload = append(payload, msg.ExtendedProtocolConfigurationOptions...)
	}

	return payload
}

func EncodePDUSessionReleaseComplete(msg *PDUSessionReleaseCompleteMsg) []byte {
	payload := make([]byte, 0)

	if msg.Cause5GSM > 0 {
		payload = append(payload, 0x59)
		payload = append(payload, msg.Cause5GSM)
	}

	if len(msg.ExtendedProtocolConfigurationOptions) > 0 {
		payload = append(payload, 0x7b)
		lengthBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(lengthBytes, uint16(len(msg.ExtendedProtocolConfigurationOptions)))
		payload = append(payload, lengthBytes...)
		payload = append(payload, msg.ExtendedProtocolConfigurationOptions...)
	}

	return payload
}

func DecodeFiveGSMStatus(payload []byte) (*FiveGSMStatusMsg, error) {
	if len(payload) < 1 {
		return nil, fmt.Errorf("5GSM status message too short")
	}

	msg := &FiveGSMStatusMsg{
		Cause5GSM: payload[0],
	}

	return msg, nil
}

func EncodeFiveGSMStatus(msg *FiveGSMStatusMsg) []byte {
	payload := make([]byte, 0)
	payload = append(payload, msg.Cause5GSM)
	return payload
}

func EncodePDUSessionAuthenticationCommand(msg *PDUSessionAuthenticationCommandMsg) []byte {
	payload := make([]byte, 0)

	if len(msg.EAPMessage) > 0 {
		payload = append(payload, 0x78)
		lengthBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(lengthBytes, uint16(len(msg.EAPMessage)))
		payload = append(payload, lengthBytes...)
		payload = append(payload, msg.EAPMessage...)
	}

	return payload
}

func DecodePDUSessionAuthenticationComplete(payload []byte) (*PDUSessionAuthenticationCompleteMsg, error) {
	msg := &PDUSessionAuthenticationCompleteMsg{}
	offset := 0

	for offset < len(payload) {
		if offset >= len(payload) {
			break
		}

		iei := payload[offset]
		offset++

		switch iei {
		case 0x78:
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

type QoSRule struct {
	Identifier          uint8
	Precedence          uint8
	QFI                 uint8
	PacketFilterList    []PacketFilter
	ReflectiveQosActive bool
	ReflectiveQosTimer  uint16
}

type PacketFilter struct {
	Direction       uint8
	Identifier      uint8
	ComponentType   uint8
	ComponentValue  []byte
}

const (
	RuleOperationCodeCreateNewQoSRule    = 0x01
	RuleOperationCodeDeleteExistingQoSRule = 0x02
	RuleOperationCodeModifyExistingQoSRule = 0x03
	PacketFilterDirectionDownlink        = 0x01
	PacketFilterDirectionUplink          = 0x02
	PacketFilterDirectionBidirectional   = 0x03
)

func BuildQoSRules(rules []QoSRule) []byte {
	if len(rules) == 0 {
		return []byte{0x01, 0x01, 0x01, 0x01, 0x01, 0x01}
	}

	result := make([]byte, 0)

	for _, rule := range rules {
		ruleBytes := encodeQoSRule(rule)
		result = append(result, ruleBytes...)
	}

	return result
}

func encodeQoSRule(rule QoSRule) []byte {
	ruleData := make([]byte, 0)

	numFilters := len(rule.PacketFilterList)
	if numFilters > 15 {
		numFilters = 15
	}

	ruleOpCode := uint8(RuleOperationCodeCreateNewQoSRule)
	dqrBit := uint8(0)
	if rule.ReflectiveQosActive {
		dqrBit = 1
	}
	opCodeByte := (ruleOpCode << 5) | (dqrBit << 4) | uint8(numFilters)
	ruleData = append(ruleData, opCodeByte)

	for _, filter := range rule.PacketFilterList {
		ruleData = append(ruleData, (filter.Direction<<4)|filter.Identifier)

		filterContents := make([]byte, 0)
		filterContents = append(filterContents, filter.ComponentType)
		filterContents = append(filterContents, filter.ComponentValue...)
		ruleData = append(ruleData, uint8(len(filterContents)))
		ruleData = append(ruleData, filterContents...)
	}

	ruleData = append(ruleData, rule.Precedence)

	ruleData = append(ruleData, rule.QFI)

	segregationBit := uint8(0)
	ruleData = append(ruleData, segregationBit)

	if rule.ReflectiveQosActive && rule.ReflectiveQosTimer > 0 {
		ruleData = append(ruleData, 0x01)
		timerBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(timerBytes, rule.ReflectiveQosTimer)
		ruleData = append(ruleData, timerBytes...)
	}

	result := make([]byte, 0)
	result = append(result, rule.Identifier)
	lengthBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(lengthBytes, uint16(len(ruleData)))
	result = append(result, lengthBytes...)
	result = append(result, ruleData...)

	return result
}

func BuildDefaultQoSRule(qfi uint8, reflectiveQos bool, rqTimer uint16) []byte {
	rule := QoSRule{
		Identifier:          0x01,
		Precedence:          0xFF,
		QFI:                 qfi,
		PacketFilterList:    []PacketFilter{},
		ReflectiveQosActive: reflectiveQos,
		ReflectiveQosTimer:  rqTimer,
	}

	return BuildQoSRules([]QoSRule{rule})
}
