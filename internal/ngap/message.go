package ngap

import (
	"encoding/binary"
	"fmt"
)

const (
	ProcedureCodeNGSetup                   = 21
	ProcedureCodeInitialUEMessage          = 15
	ProcedureCodeDownlinkNASTransport      = 4
	ProcedureCodeUplinkNASTransport        = 46
	ProcedureCodeInitialContextSetup       = 14
	ProcedureCodePDUSessionResourceSetup   = 29
	ProcedureCodeUEContextRelease          = 41
	ProcedureCodeNGReset                   = 20
	ProcedureCodePaging                    = 8
	ProcedureCodeHandoverPreparation       = 0
	ProcedureCodeHandoverResourceAllocation = 1
	ProcedureCodeHandoverNotification      = 3
	ProcedureCodePathSwitchRequest         = 2
	ProcedureCodeErrorIndication           = 11
	ProcedureCodeOverloadStart             = 24
	ProcedureCodeOverloadStop              = 25
	ProcedureCodeAMFConfigurationUpdate    = 39
	ProcedureCodeRANConfigurationUpdate    = 40
	ProcedureCodeUERadioCapabilityInfoIndication = 43
	ProcedureCodeUETNLABindingReleaseRequest = 49
	ProcedureCodeTraceStart                = 13
	ProcedureCodeDeactivateTrace           = 23
	ProcedureCodeWriteReplaceWarning       = 34
	ProcedureCodeLocationReportingControl  = 9
	ProcedureCodeLocationReport            = 10
	ProcedureCodeCellTrafficTrace          = 5
	ProcedureCodeRANCPRelocationIndication = 33
)

const (
	CriticalityReject = 0
	CriticalityIgnore = 1
	CriticalityNotify = 2
)

type PDUType string

const (
	PDUTypeInitiatingMessage   PDUType = "initiatingMessage"
	PDUTypeSuccessfulOutcome   PDUType = "successfulOutcome"
	PDUTypeUnsuccessfulOutcome PDUType = "unsuccessfulOutcome"
)

type NGAPPDU struct {
	Type          PDUType
	ProcedureCode int
	Criticality   int
	IEs           []ProtocolIE
}

type ProtocolIE struct {
	Id          int
	Criticality int
	Value       interface{}
}

const (
	ProtocolIEIDRANUENGAPID                   = 85
	ProtocolIEIDAMFUENGAPID                   = 10
	ProtocolIEIDNASPDU                        = 38
	ProtocolIEIDUserLocationInformation       = 121
	ProtocolIEIDRRCEstablishmentCause         = 90
	ProtocolIEIDFiveGSTMSI                    = 23
	ProtocolIEIDGlobalRANNodeID               = 27
	ProtocolIEIDSupportedTAList               = 102
	ProtocolIEIDDefaultPagingDRX              = 21
	ProtocolIEIDRANNodeName                   = 82
	ProtocolIEIDPDUSessionResourceSetupListCxtReq = 74
	ProtocolIEIDPDUSessionResourceSetupListSUReq  = 74
	ProtocolIEIDPDUSessionResourceToBeSwitchedList = 75
	ProtocolIEIDAllowedNSSAI                  = 0
	ProtocolIEIDUESecurityCapabilities        = 119
	ProtocolIEIDSecurityKey                   = 94
	ProtocolIEIDUEPagingIdentity              = 114
	ProtocolIEIDTAIListForPaging              = 105
	ProtocolIEIDPagingDRX                     = 70
	ProtocolIEIDPagingPriority                = 71
	ProtocolIEIDResetType                     = 40
	ProtocolIEIDUEAssociatedLogicalNGConnectionList = 114
	ProtocolIEIDCause                         = 15
	ProtocolIEIDCriticalityDiagnostics        = 19
	ProtocolIEIDOverloadStartNSSAIList        = 69
	ProtocolIEIDAMFOverloadResponse           = 1
	ProtocolIEIDOverloadAction                = 68
	ProtocolIEIDTrafficLoadReductionIndication = 109
	ProtocolIEIDAMFName                       = 82
	ProtocolIEIDServedGUAMIList               = 96
	ProtocolIEIDRelativeAMFCapacity           = 87
	ProtocolIEIDPLMNSupportList               = 80
	ProtocolIEIDAMFTNLAssociationToAddList    = 5
	ProtocolIEIDAMFTNLAssociationToRemoveList = 6
	ProtocolIEIDAMFTNLAssociationToUpdateList = 7
	ProtocolIEIDUERadioCapability             = 118
	ProtocolIEIDUERadioCapabilityForPaging    = 117
	ProtocolIEIDTraceActivation               = 112
	ProtocolIEIDTraceReference                = 113
	ProtocolIEIDMessageIdentifier             = 111
	ProtocolIEIDSerialNumber                  = 92
	ProtocolIEIDWarningAreaList               = 128
	ProtocolIEIDRepetitionPeriod              = 88
	ProtocolIEIDNumberOfBroadcastsRequested   = 63
	ProtocolIEIDWarningType                   = 129
	ProtocolIEIDWarningSecurityInfo           = 130
	ProtocolIEIDDataCodingScheme              = 20
	ProtocolIEIDWarningMessageContents        = 131
	ProtocolIEIDConcurrentWarningMessageIndicator = 18
	ProtocolIEIDLocationReportingRequestType  = 64
	ProtocolIEIDLocationReportingReferenceID  = 65
	ProtocolIEIDNGRANCGI                      = 67
	ProtocolIEIDHandoverType                  = 28
	ProtocolIEIDTargetID                      = 104
	ProtocolIEIDPDUSessionResourceListHORqd   = 72
	ProtocolIEIDSourceToTargetTransparentContainer = 99
	ProtocolIEIDTargetToSourceTransparentContainer = 100
	ProtocolIEIDPDUSessionResourceAdmittedList = 73
	ProtocolIEIDPDUSessionResourceFailedToSetupListHOAck = 12
)

type GlobalRANNodeID struct {
	PLMNIdentity []byte
	GNBIDType    int
	GNBID        []byte
}

type SupportedTAItem struct {
	TAC                   []byte
	BroadcastPLMNList     []BroadcastPLMNItem
}

type BroadcastPLMNItem struct {
	PLMNIdentity []byte
	TAISliceList []SliceSupport
}

type SliceSupport struct {
	SST int
	SD  []byte
}

type UserLocationInformation struct {
	NRCGIPresent bool
	NRCGI        *NRCGI
	TAI          *TAI
}

type NRCGI struct {
	PLMNIdentity []byte
	NRCellID     []byte
}

type TAI struct {
	PLMNIdentity []byte
	TAC          []byte
}

type PDUSessionResourceSetupItemCxtReq struct {
	PDUSessionID            int
	SNSSAI                  SNSSAI
	PDUSessionResourceSetupRequestTransfer []byte
}

type PDUSessionResourceSetupItem struct {
	PDUSessionID int64
	NASPDU       []byte
	SNSSAI       *SNSSAI
	TransferData []byte
}

type SNSSAI struct {
	SST int
	SD  []byte
}

type UEPagingIdentity struct {
	FiveGSTMSI    *FiveGSTMSI
	IMSIPagingTAI *TAI
}

type FiveGSTMSI struct {
	AMFSetID    uint16
	AMFPointer  uint8
	FiveGTMSI   uint32
}

type TAIListForPaging struct {
	TAIItems []TAI
}

type PagingDRX int

const (
	PagingDRXv32  PagingDRX = 32
	PagingDRXv64  PagingDRX = 64
	PagingDRXv128 PagingDRX = 128
	PagingDRXv256 PagingDRX = 256
)

type ResetType int

const (
	ResetTypeNGInterface ResetType = 0
	ResetTypePartOfNGInterface ResetType = 1
)

type UEAssociatedLogicalNGConnectionItem struct {
	AMFUENGAPID *int64
	RANUENGAPID *int64
}

type Cause struct {
	CauseGroup int
	CauseValue int
}

type OverloadAction int

const (
	OverloadActionRejectNonEmergencyMoData OverloadAction = 0
	OverloadActionRejectRRCCrSignalling    OverloadAction = 1
	OverloadActionPermitEmergencySessionsAndMobileTerminatedServicesOnly OverloadAction = 2
	OverloadActionPermitHighPrioritySessionsAndMobileTerminatedServicesOnly OverloadAction = 3
)

type OverloadResponse int

const (
	OverloadResponseReject OverloadResponse = 0
	OverloadResponseAccept OverloadResponse = 1
)

type TraceActivation struct {
	NGRANTraceID       []byte
	InterfacesToTrace  []byte
	TraceDepth         int
	TraceCollectionEntityIPAddress []byte
}

type TraceReference struct {
	PLMNIdentity []byte
	TraceID      []byte
}

type WarningAreaList struct {
	CellIDList []NRCGI
	TAIList    []TAI
	EmergencyAreaIDList [][]byte
}

type WarningType []byte

type LocationReportingRequestType int

const (
	LocationReportingRequestTypeStartOfUEPresenceInAreaOfInterest LocationReportingRequestType = 0
	LocationReportingRequestTypeStopOfUEPresenceInAreaOfInterest  LocationReportingRequestType = 1
	LocationReportingRequestTypeUEPresenceInAreaOfInterest        LocationReportingRequestType = 2
	LocationReportingRequestTypeDirectLocationReporting           LocationReportingRequestType = 3
	LocationReportingRequestTypeChangeOfUEPresenceInAreaOfInterest LocationReportingRequestType = 4
	LocationReportingRequestTypeStopChangeOfUEPresenceInAreaOfInterest LocationReportingRequestType = 5
)

type HandoverType int

const (
	HandoverTypeIntra5GS HandoverType = 0
	HandoverTypeFiveGSToEPS HandoverType = 1
	HandoverTypeEPSTo5GS HandoverType = 2
)

type TargetID struct {
	TargetRANNodeID *GlobalRANNodeID
	TAI             *TAI
}

type PDUSessionResourceItemHORqd struct {
	PDUSessionID     int
	HandoverTransfer []byte
}

type PDUSessionResourceAdmittedItem struct {
	PDUSessionID                int
	HandoverRequestAckTransfer  []byte
}

type PDUSessionResourceFailedToSetupItem struct {
	PDUSessionID int
	Cause        *Cause
}

func DecodeNGAPPDU(data []byte) (*NGAPPDU, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf("message too short")
	}

	pdu := &NGAPPDU{
		IEs: make([]ProtocolIE, 0),
	}

	index := 0

	if data[index] == 0x00 {
		pdu.Type = PDUTypeInitiatingMessage
	} else if data[index] == 0x20 {
		pdu.Type = PDUTypeSuccessfulOutcome
	} else if data[index] == 0x40 {
		pdu.Type = PDUTypeUnsuccessfulOutcome
	}
	index++

	length, consumed, err := decodeLength(data[index:])
	if err != nil {
		return nil, fmt.Errorf("failed to decode length: %w", err)
	}
	index += consumed
	_ = length

	if index >= len(data) {
		return nil, fmt.Errorf("incomplete message")
	}

	if data[index] == 0x30 {
		index++
		length, consumed, err = decodeLength(data[index:])
		if err != nil {
			return nil, fmt.Errorf("failed to decode sequence length: %w", err)
		}
		index += consumed
	}

	if index+1 >= len(data) {
		return nil, fmt.Errorf("incomplete message")
	}

	if data[index] == 0x02 {
		index++
		if data[index] == 0x01 {
			index++
			pdu.ProcedureCode = int(data[index])
			index++
		}
	}

	if index+1 >= len(data) {
		return nil, fmt.Errorf("incomplete message")
	}

	if data[index] == 0x0a {
		index++
		if data[index] == 0x01 {
			index++
			pdu.Criticality = int(data[index])
			index++
		}
	}

	return pdu, nil
}

func EncodeNGAPPDU(pdu *NGAPPDU) ([]byte, error) {
	result := make([]byte, 0)

	switch pdu.Type {
	case PDUTypeInitiatingMessage:
		result = append(result, 0x00)
	case PDUTypeSuccessfulOutcome:
		result = append(result, 0x20)
	case PDUTypeUnsuccessfulOutcome:
		result = append(result, 0x40)
	}

	content := make([]byte, 0)

	content = append(content, 0x30)
	seqContent := make([]byte, 0)

	seqContent = append(seqContent, 0x02, 0x01, byte(pdu.ProcedureCode))

	seqContent = append(seqContent, 0x0a, 0x01, byte(pdu.Criticality))

	seqContent = append(seqContent, 0xa0)
	valueContent := encodeProtocolIEs(pdu.IEs)
	seqContent = append(seqContent, encodeLength(len(valueContent))...)
	seqContent = append(seqContent, valueContent...)

	content = append(content, encodeLength(len(seqContent))...)
	content = append(content, seqContent...)

	result = append(result, encodeLength(len(content))...)
	result = append(result, content...)

	return result, nil
}

func encodeProtocolIEs(ies []ProtocolIE) []byte {
	result := make([]byte, 0)

	result = append(result, 0x30)
	content := make([]byte, 0)

	for _, ie := range ies {
		ieBytes := encodeProtocolIE(ie)
		content = append(content, ieBytes...)
	}

	result = append(result, encodeLength(len(content))...)
	result = append(result, content...)

	return result
}

func encodeProtocolIE(ie ProtocolIE) []byte {
	result := make([]byte, 0)

	result = append(result, 0x30)
	content := make([]byte, 0)

	content = append(content, 0x02, 0x01, byte(ie.Id))

	content = append(content, 0x0a, 0x01, byte(ie.Criticality))

	content = append(content, 0xa0)
	var valueBytes []byte
	switch v := ie.Value.(type) {
	case []byte:
		valueBytes = v
	case int64:
		valueBytes = encodeInteger(v)
	case string:
		valueBytes = encodeOctetString([]byte(v))
	default:
		valueBytes = []byte{}
	}
	content = append(content, encodeLength(len(valueBytes))...)
	content = append(content, valueBytes...)

	result = append(result, encodeLength(len(content))...)
	result = append(result, content...)

	return result
}

func encodeInteger(value int64) []byte {
	result := make([]byte, 0)
	result = append(result, 0x02)

	if value >= -128 && value <= 127 {
		result = append(result, 0x01, byte(value))
	} else if value >= -32768 && value <= 32767 {
		result = append(result, 0x02)
		buf := make([]byte, 2)
		binary.BigEndian.PutUint16(buf, uint16(value))
		result = append(result, buf...)
	} else {
		result = append(result, 0x04)
		buf := make([]byte, 4)
		binary.BigEndian.PutUint32(buf, uint32(value))
		result = append(result, buf...)
	}

	return result
}

func encodeOctetString(data []byte) []byte {
	result := make([]byte, 0)
	result = append(result, 0x04)
	result = append(result, encodeLength(len(data))...)
	result = append(result, data...)
	return result
}

func encodeLength(length int) []byte {
	if length < 128 {
		return []byte{byte(length)}
	} else if length < 256 {
		return []byte{0x81, byte(length)}
	} else if length < 65536 {
		buf := make([]byte, 3)
		buf[0] = 0x82
		binary.BigEndian.PutUint16(buf[1:], uint16(length))
		return buf
	}
	buf := make([]byte, 5)
	buf[0] = 0x84
	binary.BigEndian.PutUint32(buf[1:], uint32(length))
	return buf
}

func decodeLength(data []byte) (int, int, error) {
	if len(data) < 1 {
		return 0, 0, fmt.Errorf("insufficient data for length")
	}

	first := data[0]
	if first < 128 {
		return int(first), 1, nil
	}

	numOctets := int(first & 0x7f)
	if len(data) < 1+numOctets {
		return 0, 0, fmt.Errorf("insufficient data for length")
	}

	length := 0
	for i := 0; i < numOctets; i++ {
		length = (length << 8) | int(data[1+i])
	}

	return length, 1 + numOctets, nil
}

func DecodeInteger(data []byte, offset int) (int64, int, error) {
	if offset >= len(data) {
		return 0, 0, fmt.Errorf("offset out of bounds")
	}

	if data[offset] != 0x02 {
		return 0, 0, fmt.Errorf("not an integer")
	}
	offset++

	length, consumed, err := decodeLength(data[offset:])
	if err != nil {
		return 0, 0, err
	}
	offset += consumed

	if offset+length > len(data) {
		return 0, 0, fmt.Errorf("insufficient data")
	}

	var value int64
	for i := 0; i < length; i++ {
		value = (value << 8) | int64(data[offset+i])
	}

	return value, consumed + length + 1, nil
}

func DecodeOctetString(data []byte, offset int) ([]byte, int, error) {
	if offset >= len(data) {
		return nil, 0, fmt.Errorf("offset out of bounds")
	}

	if data[offset] != 0x04 {
		return nil, 0, fmt.Errorf("not an octet string")
	}
	offset++

	length, consumed, err := decodeLength(data[offset:])
	if err != nil {
		return nil, 0, err
	}
	offset += consumed

	if offset+length > len(data) {
		return nil, 0, fmt.Errorf("insufficient data")
	}

	value := make([]byte, length)
	copy(value, data[offset:offset+length])

	return value, consumed + length + 1, nil
}
