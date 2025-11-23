package ngap

import (
	"fmt"

	"github.com/gavin/amf/internal/context"
	"github.com/gavin/amf/internal/logger"
)

type Handler struct {
	amfContext *context.AMFContext
	server     *Server
	nasHandler NASHandler
}

type NASHandler interface {
	HandleNASMessage(ue *context.UEContext, nasPDU []byte) error
}

func NewHandler(ctx *context.AMFContext) *Handler {
	return &Handler{
		amfContext: ctx,
	}
}

func (h *Handler) SetServer(server *Server) {
	h.server = server
}

func (h *Handler) SetNASHandler(handler NASHandler) {
	h.nasHandler = handler
}

func (h *Handler) HandleNGSetupRequest(ranContext *context.RANContext, pdu *NGAPPDU) error {
	logger.NgapLog.Info("Handling NG Setup Request")

	var ranNodeName string
	var globalRANNodeID *GlobalRANNodeID
	var supportedTAList []SupportedTAItem
	var pagingDRX int

	for _, ie := range pdu.IEs {
		switch ie.Id {
		case ProtocolIEIDRANNodeName:
			if data, ok := ie.Value.([]byte); ok {
				ranNodeName = string(data)
			}
		case ProtocolIEIDGlobalRANNodeID:
			if data, ok := ie.Value.([]byte); ok {
				globalRANNodeID = &GlobalRANNodeID{
					PLMNIdentity: data[:3],
					GNBID:        data[3:],
				}
			}
		case ProtocolIEIDSupportedTAList:
			if data, ok := ie.Value.([]byte); ok {
				if len(data) >= 3 {
					supportedTAList = []SupportedTAItem{
						{
							TAC: data[:3],
							BroadcastPLMNList: []BroadcastPLMNItem{
								{
									PLMNIdentity: data[3:6],
								},
							},
						},
					}
				}
			}
		case ProtocolIEIDDefaultPagingDRX:
			pagingDRX = 32
		}
	}

	ranContext.RanNodeName = ranNodeName
	if globalRANNodeID != nil {
		ranContext.GlobalRanNodeId = &context.GlobalRanNodeId{
			PlmnId: context.PlmnId{
				Mcc: string(globalRANNodeID.PLMNIdentity[:3]),
				Mnc: string(globalRANNodeID.PLMNIdentity[3:]),
			},
			GnbId: string(globalRANNodeID.GNBID),
		}
	}

	if len(supportedTAList) > 0 {
		for _, tai := range supportedTAList {
			ranContext.SupportedTAList = append(ranContext.SupportedTAList, context.SupportedTAI{
				Tai: context.Tai{
					PlmnId: context.PlmnId{
						Mcc: string(tai.TAC[:3]),
						Mnc: "01",
					},
					Tac: string(tai.TAC),
				},
			})
		}
	}

	ranContext.DefaultPagingDrx = fmt.Sprintf("%d", pagingDRX)

	logger.NgapLog.Infof("NG Setup completed for RAN: %s", ranNodeName)

	responsePDU := &NGAPPDU{
		Type:          PDUTypeSuccessfulOutcome,
		ProcedureCode: ProcedureCodeNGSetup,
		Criticality:   CriticalityReject,
		IEs: []ProtocolIE{
			{
				Id:          ProtocolIEIDRANNodeName,
				Criticality: CriticalityReject,
				Value:       []byte(h.amfContext.Name),
			},
		},
	}

	return h.server.SendMessage(ranContext.Conn, responsePDU)
}

func (h *Handler) HandleInitialUEMessage(ranContext *context.RANContext, pdu *NGAPPDU) error {
	logger.NgapLog.Info("Handling Initial UE Message")

	var ranUeNgapId int64
	var nasPDU []byte
	var userLocationInfo *UserLocationInformation

	for _, ie := range pdu.IEs {
		switch ie.Id {
		case ProtocolIEIDRANUENGAPID:
			if val, ok := ie.Value.(int64); ok {
				ranUeNgapId = val
			}
		case ProtocolIEIDNASPDU:
			if data, ok := ie.Value.([]byte); ok {
				nasPDU = data
			}
		case ProtocolIEIDUserLocationInformation:
			if data, ok := ie.Value.([]byte); ok {
				userLocationInfo = &UserLocationInformation{
					NRCGIPresent: true,
					NRCGI: &NRCGI{
						PLMNIdentity: data[:3],
						NRCellID:     data[3:8],
					},
					TAI: &TAI{
						PLMNIdentity: data[8:11],
						TAC:          data[11:14],
					},
				}
			}
		}
	}

	ue := h.amfContext.NewUEContext(ranUeNgapId)
	ue.RanContext = ranContext
	ue.CmState = context.CmConnected
	ue.AccessType = context.AccessType3GPP

	if userLocationInfo != nil && userLocationInfo.TAI != nil {
		ue.Tai = context.Tai{
			PlmnId: context.PlmnId{
				Mcc: string(userLocationInfo.TAI.PLMNIdentity[:3]),
				Mnc: "01",
			},
			Tac: string(userLocationInfo.TAI.TAC),
		}
	}

	ranContext.AddUE(ranUeNgapId, ue)

	logger.NgapLog.Infof("Initial UE Message: RAN UE NGAP ID=%d, AMF UE NGAP ID=%d, NAS PDU length=%d",
		ranUeNgapId, ue.AmfUeNgapId, len(nasPDU))

	if len(nasPDU) > 0 && h.nasHandler != nil {
		if err := h.nasHandler.HandleNASMessage(ue, nasPDU); err != nil {
			logger.NgapLog.Errorf("Failed to handle NAS message: %v", err)
			return err
		}
	}

	return nil
}

func (h *Handler) HandleUplinkNASTransport(ranContext *context.RANContext, pdu *NGAPPDU) error {
	logger.NgapLog.Info("Handling Uplink NAS Transport")

	var ranUeNgapId int64
	var amfUeNgapId int64
	var nasPDU []byte

	for _, ie := range pdu.IEs {
		switch ie.Id {
		case ProtocolIEIDRANUENGAPID:
			if val, ok := ie.Value.(int64); ok {
				ranUeNgapId = val
			}
		case ProtocolIEIDAMFUENGAPID:
			if val, ok := ie.Value.(int64); ok {
				amfUeNgapId = val
			}
		case ProtocolIEIDNASPDU:
			if data, ok := ie.Value.([]byte); ok {
				nasPDU = data
			}
		}
	}

	ue, ok := h.amfContext.GetUEContextByAmfUeNgapId(amfUeNgapId)
	if !ok {
		return fmt.Errorf("UE context not found for AMF UE NGAP ID: %d", amfUeNgapId)
	}

	logger.NgapLog.Infof("Uplink NAS Transport: RAN UE NGAP ID=%d, AMF UE NGAP ID=%d, NAS PDU length=%d",
		ranUeNgapId, amfUeNgapId, len(nasPDU))

	if len(nasPDU) > 0 && h.nasHandler != nil {
		if err := h.nasHandler.HandleNASMessage(ue, nasPDU); err != nil {
			logger.NgapLog.Errorf("Failed to handle NAS message: %v", err)
			return err
		}
	}

	return nil
}

func (h *Handler) HandleInitialContextSetupResponse(ranContext *context.RANContext, pdu *NGAPPDU) error {
	logger.NgapLog.Info("Handling Initial Context Setup Response")

	var amfUeNgapId int64

	for _, ie := range pdu.IEs {
		if ie.Id == ProtocolIEIDAMFUENGAPID {
			if val, ok := ie.Value.(int64); ok {
				amfUeNgapId = val
			}
		}
	}

	ue, ok := h.amfContext.GetUEContextByAmfUeNgapId(amfUeNgapId)
	if !ok {
		return fmt.Errorf("UE context not found for AMF UE NGAP ID: %d", amfUeNgapId)
	}

	ue.CmState = context.CmConnected

	logger.NgapLog.Infof("Initial Context Setup completed for AMF UE NGAP ID=%d", amfUeNgapId)

	return nil
}

func (h *Handler) HandlePDUSessionResourceSetupResponse(ranContext *context.RANContext, pdu *NGAPPDU) error {
	logger.NgapLog.Info("Handling PDU Session Resource Setup Response")

	var amfUeNgapId int64

	for _, ie := range pdu.IEs {
		if ie.Id == ProtocolIEIDAMFUENGAPID {
			if val, ok := ie.Value.(int64); ok {
				amfUeNgapId = val
			}
		}
	}

	ue, ok := h.amfContext.GetUEContextByAmfUeNgapId(amfUeNgapId)
	if !ok {
		return fmt.Errorf("UE context not found for AMF UE NGAP ID: %d", amfUeNgapId)
	}

	logger.NgapLog.Infof("PDU Session Resource Setup completed for AMF UE NGAP ID=%d", amfUeNgapId)

	_ = ue

	return nil
}

func (h *Handler) HandleUEContextReleaseRequest(ranContext *context.RANContext, pdu *NGAPPDU) error {
	logger.NgapLog.Info("Handling UE Context Release Request")

	var amfUeNgapId int64

	for _, ie := range pdu.IEs {
		if ie.Id == ProtocolIEIDAMFUENGAPID {
			if val, ok := ie.Value.(int64); ok {
				amfUeNgapId = val
			}
		}
	}

	ue, ok := h.amfContext.GetUEContextByAmfUeNgapId(amfUeNgapId)
	if !ok {
		return fmt.Errorf("UE context not found for AMF UE NGAP ID: %d", amfUeNgapId)
	}

	ue.CmState = context.CmIdle
	h.amfContext.DeleteUEContext(amfUeNgapId)

	logger.NgapLog.Infof("UE Context released for AMF UE NGAP ID=%d", amfUeNgapId)

	return nil
}

func (h *Handler) SendDownlinkNASTransport(ranUeNgapId, amfUeNgapId int64, nasPDU []byte) error {
	logger.NgapLog.Info("Sending Downlink NAS Transport")

	ue, ok := h.amfContext.GetUEContextByAmfUeNgapId(amfUeNgapId)
	if !ok {
		return fmt.Errorf("UE context not found for AMF UE NGAP ID: %d", amfUeNgapId)
	}

	if ue.RanContext == nil {
		return fmt.Errorf("RAN context not available for UE")
	}

	pdu := &NGAPPDU{
		Type:          PDUTypeInitiatingMessage,
		ProcedureCode: ProcedureCodeDownlinkNASTransport,
		Criticality:   CriticalityIgnore,
		IEs: []ProtocolIE{
			{
				Id:          ProtocolIEIDAMFUENGAPID,
				Criticality: CriticalityReject,
				Value:       amfUeNgapId,
			},
			{
				Id:          ProtocolIEIDRANUENGAPID,
				Criticality: CriticalityReject,
				Value:       ranUeNgapId,
			},
			{
				Id:          ProtocolIEIDNASPDU,
				Criticality: CriticalityReject,
				Value:       nasPDU,
			},
		},
	}

	logger.NgapLog.Infof("Downlink NAS Transport: RAN UE NGAP ID=%d, AMF UE NGAP ID=%d, NAS PDU length=%d",
		ranUeNgapId, amfUeNgapId, len(nasPDU))

	return h.server.SendMessage(ue.RanContext.Conn, pdu)
}

func (h *Handler) SendPDUSessionResourceSetupRequest(ranUeNgapId, amfUeNgapId int64, pduSessionId uint8, nasPDU []byte, n2SmInfo []byte) error {
	logger.NgapLog.Infof("Sending PDU Session Resource Setup Request for PDU Session ID: %d", pduSessionId)

	ue, ok := h.amfContext.GetUEContextByAmfUeNgapId(amfUeNgapId)
	if !ok {
		return fmt.Errorf("UE context not found for AMF UE NGAP ID: %d", amfUeNgapId)
	}

	if ue.RanContext == nil {
		return fmt.Errorf("RAN context not available for UE")
	}

	pduSessionItem := &PDUSessionResourceSetupItem{
		PDUSessionID: int64(pduSessionId),
		NASPDU:       nasPDU,
		SNSSAI: &SNSSAI{
			SST: 1,
		},
	}

	if len(n2SmInfo) > 0 {
		pduSessionItem.TransferData = n2SmInfo
	}

	pdu := &NGAPPDU{
		Type:          PDUTypeInitiatingMessage,
		ProcedureCode: ProcedureCodePDUSessionResourceSetup,
		Criticality:   CriticalityReject,
		IEs: []ProtocolIE{
			{
				Id:          ProtocolIEIDAMFUENGAPID,
				Criticality: CriticalityReject,
				Value:       amfUeNgapId,
			},
			{
				Id:          ProtocolIEIDRANUENGAPID,
				Criticality: CriticalityReject,
				Value:       ranUeNgapId,
			},
			{
				Id:          ProtocolIEIDPDUSessionResourceSetupListSUReq,
				Criticality: CriticalityReject,
				Value:       pduSessionItem,
			},
		},
	}

	logger.NgapLog.Infof("PDU Session Resource Setup Request: RAN UE NGAP ID=%d, AMF UE NGAP ID=%d, PDU Session ID=%d",
		ranUeNgapId, amfUeNgapId, pduSessionId)

	return h.server.SendMessage(ue.RanContext.Conn, pdu)
}

func (h *Handler) SendPaging(ue *context.UEContext) error {
	logger.NgapLog.Info("Sending Paging message")

	if ue.Guti == nil {
		return fmt.Errorf("UE has no GUTI allocated, cannot page")
	}

	if ue.CmState == context.CmConnected {
		logger.NgapLog.Warn("UE is in CM-CONNECTED state, paging not needed")
		return nil
	}

	fiveGSTMSI := EncodeFiveGSTMSI(ue.Guti)

	taiList := &TAIListForPaging{
		TAIItems: []TAI{
			{
				PLMNIdentity: EncodePlmnIdentity(ue.Tai.PlmnId),
				TAC:          EncodeTAC(ue.Tai.Tac),
			},
		},
	}

	ranContexts := h.amfContext.GetRANContextsByTAI(ue.Tai)
	if len(ranContexts) == 0 {
		return fmt.Errorf("no RAN contexts found for TAI: %+v", ue.Tai)
	}

	logger.NgapLog.Infof("Paging UE in %d RAN(s)", len(ranContexts))

	for _, ran := range ranContexts {
		pdu := &NGAPPDU{
			Type:          PDUTypeInitiatingMessage,
			ProcedureCode: ProcedureCodePaging,
			Criticality:   CriticalityIgnore,
			IEs: []ProtocolIE{
				{
					Id:          ProtocolIEIDUEPagingIdentity,
					Criticality: CriticalityIgnore,
					Value:       fiveGSTMSI,
				},
				{
					Id:          ProtocolIEIDTAIListForPaging,
					Criticality: CriticalityIgnore,
					Value:       EncodeTAIListForPaging(taiList),
				},
			},
		}

		if ran.DefaultPagingDrx != "" {
			pdu.IEs = append(pdu.IEs, ProtocolIE{
				Id:          ProtocolIEIDPagingDRX,
				Criticality: CriticalityIgnore,
				Value:       int64(32),
			})
		}

		if err := h.server.SendMessage(ran.Conn, pdu); err != nil {
			logger.NgapLog.Errorf("Failed to send paging to RAN %s: %v", ran.RanNodeName, err)
			continue
		}

		logger.NgapLog.Infof("Paging sent to RAN: %s for UE with 5G-S-TMSI", ran.RanNodeName)
	}

	return nil
}

func EncodeFiveGSTMSI(guti *context.Guti) []byte {
	fiveGSTMSI := make([]byte, 6)

	amfSetId := uint16(0)
	amfPointer := uint8(0)

	fiveGSTMSI[0] = byte(amfSetId >> 8)
	fiveGSTMSI[1] = byte(amfSetId)
	fiveGSTMSI[2] = amfPointer

	fiveGSTMSI[3] = byte(guti.Tmsi >> 24)
	fiveGSTMSI[4] = byte(guti.Tmsi >> 16)
	fiveGSTMSI[5] = byte(guti.Tmsi >> 8)

	return fiveGSTMSI
}

func EncodePlmnIdentity(plmnId context.PlmnId) []byte {
	plmn := make([]byte, 3)
	if len(plmnId.Mcc) >= 3 {
		plmn[0] = plmnId.Mcc[0]
		plmn[1] = plmnId.Mcc[1]
		plmn[2] = plmnId.Mcc[2]
	}
	return plmn
}

func EncodeTAC(tac string) []byte {
	tacBytes := make([]byte, 3)
	if len(tac) >= 3 {
		tacBytes[0] = tac[0]
		tacBytes[1] = tac[1]
		tacBytes[2] = tac[2]
	}
	return tacBytes
}

func EncodeTAIListForPaging(taiList *TAIListForPaging) []byte {
	result := make([]byte, 0)
	for _, tai := range taiList.TAIItems {
		result = append(result, tai.PLMNIdentity...)
		result = append(result, tai.TAC...)
	}
	return result
}

func (h *Handler) HandleNGReset(ranContext *context.RANContext, pdu *NGAPPDU) error {
	logger.NgapLog.Info("Handling NG Reset")

	var resetType ResetType = ResetTypeNGInterface
	var ueList []UEAssociatedLogicalNGConnectionItem

	for _, ie := range pdu.IEs {
		switch ie.Id {
		case ProtocolIEIDResetType:
			if data, ok := ie.Value.([]byte); ok && len(data) > 0 {
				if data[0] == 0x01 {
					resetType = ResetTypePartOfNGInterface
				}
			}
		case ProtocolIEIDUEAssociatedLogicalNGConnectionList:
			if data, ok := ie.Value.([]byte); ok {
				ueList = parseUEAssociatedList(data)
			}
		}
	}

	if resetType == ResetTypeNGInterface {
		logger.NgapLog.Info("Resetting all UE-associated logical NG connections")

		ranContext.RangeUEs(func(ranUeNgapId int64, ue *context.UEContext) bool {
			logger.NgapLog.Infof("Releasing UE context for AMF UE NGAP ID=%d", ue.AmfUeNgapId)
			ue.CmState = context.CmIdle
			h.amfContext.DeleteUEContext(ue.AmfUeNgapId)
			return true
		})

		ranContext.ClearAllUEs()
	} else {
		logger.NgapLog.Infof("Resetting %d UE-associated logical NG connections", len(ueList))

		for _, item := range ueList {
			if item.AMFUENGAPID != nil {
				amfUeNgapId := *item.AMFUENGAPID
				ue, ok := h.amfContext.GetUEContextByAmfUeNgapId(amfUeNgapId)
				if ok {
					logger.NgapLog.Infof("Releasing UE context for AMF UE NGAP ID=%d", amfUeNgapId)
					ue.CmState = context.CmIdle
					h.amfContext.DeleteUEContext(amfUeNgapId)
				}
			}
		}
	}

	logger.NgapLog.Info("NG Reset completed, sending NG Reset Acknowledge")

	return h.SendNGResetAcknowledge(ranContext, resetType, ueList)
}

func (h *Handler) SendNGResetAcknowledge(ranContext *context.RANContext, resetType ResetType, ueList []UEAssociatedLogicalNGConnectionItem) error {
	logger.NgapLog.Info("Sending NG Reset Acknowledge")

	pdu := &NGAPPDU{
		Type:          PDUTypeSuccessfulOutcome,
		ProcedureCode: ProcedureCodeNGReset,
		Criticality:   CriticalityReject,
		IEs:           []ProtocolIE{},
	}

	if resetType == ResetTypePartOfNGInterface && len(ueList) > 0 {
		ueListData := encodeUEAssociatedList(ueList)
		pdu.IEs = append(pdu.IEs, ProtocolIE{
			Id:          ProtocolIEIDUEAssociatedLogicalNGConnectionList,
			Criticality: CriticalityIgnore,
			Value:       ueListData,
		})
	}

	logger.NgapLog.Info("NG Reset Acknowledge sent successfully")

	return h.server.SendMessage(ranContext.Conn, pdu)
}

func parseUEAssociatedList(data []byte) []UEAssociatedLogicalNGConnectionItem {
	items := []UEAssociatedLogicalNGConnectionItem{}

	offset := 0
	for offset+8 <= len(data) {
		item := UEAssociatedLogicalNGConnectionItem{}

		if offset+4 <= len(data) {
			amfUeNgapId := int64(data[offset])<<24 | int64(data[offset+1])<<16 | int64(data[offset+2])<<8 | int64(data[offset+3])
			item.AMFUENGAPID = &amfUeNgapId
			offset += 4
		}

		if offset+4 <= len(data) {
			ranUeNgapId := int64(data[offset])<<24 | int64(data[offset+1])<<16 | int64(data[offset+2])<<8 | int64(data[offset+3])
			item.RANUENGAPID = &ranUeNgapId
			offset += 4
		}

		items = append(items, item)
	}

	return items
}

func encodeUEAssociatedList(items []UEAssociatedLogicalNGConnectionItem) []byte {
	data := make([]byte, 0)

	for _, item := range items {
		if item.AMFUENGAPID != nil {
			amfId := *item.AMFUENGAPID
			data = append(data, byte(amfId>>24), byte(amfId>>16), byte(amfId>>8), byte(amfId))
		}
		if item.RANUENGAPID != nil {
			ranId := *item.RANUENGAPID
			data = append(data, byte(ranId>>24), byte(ranId>>16), byte(ranId>>8), byte(ranId))
		}
	}

	return data
}

func (h *Handler) HandleErrorIndication(ranContext *context.RANContext, pdu *NGAPPDU) error {
	logger.NgapLog.Info("Handling Error Indication")

	var amfUeNgapId *int64
	var ranUeNgapId *int64
	var cause *Cause

	for _, ie := range pdu.IEs {
		switch ie.Id {
		case ProtocolIEIDAMFUENGAPID:
			if val, ok := ie.Value.(int64); ok {
				amfUeNgapId = &val
			}
		case ProtocolIEIDRANUENGAPID:
			if val, ok := ie.Value.(int64); ok {
				ranUeNgapId = &val
			}
		case ProtocolIEIDCause:
			if data, ok := ie.Value.([]byte); ok && len(data) >= 2 {
				cause = &Cause{
					CauseGroup: int(data[0]),
					CauseValue: int(data[1]),
				}
			}
		case ProtocolIEIDCriticalityDiagnostics:
		}
	}

	if amfUeNgapId != nil && ranUeNgapId != nil {
		logger.NgapLog.Warnf("Error Indication received for AMF UE NGAP ID=%d, RAN UE NGAP ID=%d", *amfUeNgapId, *ranUeNgapId)
	} else if amfUeNgapId != nil {
		logger.NgapLog.Warnf("Error Indication received for AMF UE NGAP ID=%d", *amfUeNgapId)
	} else if ranUeNgapId != nil {
		logger.NgapLog.Warnf("Error Indication received for RAN UE NGAP ID=%d", *ranUeNgapId)
	} else {
		logger.NgapLog.Warn("Error Indication received (no UE context)")
	}

	if cause != nil {
		logger.NgapLog.Warnf("Cause: Group=%d, Value=%d", cause.CauseGroup, cause.CauseValue)
	}

	return nil
}

func (h *Handler) HandleOverloadStart(ranContext *context.RANContext, pdu *NGAPPDU) error {
	logger.NgapLog.Info("Handling Overload Start")

	var overloadResponse OverloadResponse = OverloadResponseAccept
	var trafficLoadReduction int = 0

	for _, ie := range pdu.IEs {
		switch ie.Id {
		case ProtocolIEIDAMFOverloadResponse:
			if data, ok := ie.Value.([]byte); ok && len(data) > 0 {
				overloadResponse = OverloadResponse(data[0])
			}
		case ProtocolIEIDTrafficLoadReductionIndication:
			if data, ok := ie.Value.([]byte); ok && len(data) > 0 {
				trafficLoadReduction = int(data[0])
			}
		case ProtocolIEIDOverloadStartNSSAIList:
		}
	}

	ranContext.IsOverloaded = true
	ranContext.TrafficLoadReductionIndication = trafficLoadReduction

	logger.NgapLog.Infof("RAN %s entered overload state - Response: %d, Traffic Load Reduction: %d%%",
		ranContext.RanNodeName, overloadResponse, trafficLoadReduction)

	return nil
}

func (h *Handler) HandleOverloadStop(ranContext *context.RANContext, pdu *NGAPPDU) error {
	logger.NgapLog.Info("Handling Overload Stop")

	ranContext.IsOverloaded = false
	ranContext.TrafficLoadReductionIndication = 0

	logger.NgapLog.Infof("RAN %s exited overload state", ranContext.RanNodeName)

	return nil
}

func (h *Handler) SendOverloadStart(ranContext *context.RANContext, overloadAction OverloadAction, trafficLoadReduction int) error {
	logger.NgapLog.Info("Sending Overload Start")

	if ranContext == nil || ranContext.Conn == nil {
		return fmt.Errorf("invalid RAN context")
	}

	ies := []ProtocolIE{
		{
			Id:          ProtocolIEIDOverloadAction,
			Criticality: CriticalityReject,
			Value:       []byte{byte(overloadAction)},
		},
	}

	if trafficLoadReduction > 0 && trafficLoadReduction <= 99 {
		ies = append(ies, ProtocolIE{
			Id:          ProtocolIEIDTrafficLoadReductionIndication,
			Criticality: CriticalityIgnore,
			Value:       []byte{byte(trafficLoadReduction)},
		})
	}

	pdu := &NGAPPDU{
		Type:          PDUTypeInitiatingMessage,
		ProcedureCode: ProcedureCodeOverloadStart,
		Criticality:   CriticalityIgnore,
		IEs:           ies,
	}

	h.amfContext.IsOverloaded = true
	h.amfContext.OverloadAction = int(overloadAction)

	logger.NgapLog.Infof("Sending Overload Start to RAN %s - Action: %d, Traffic Load Reduction: %d%%",
		ranContext.RanNodeName, overloadAction, trafficLoadReduction)

	return h.server.SendMessage(ranContext.Conn, pdu)
}

func (h *Handler) SendOverloadStop(ranContext *context.RANContext) error {
	logger.NgapLog.Info("Sending Overload Stop")

	if ranContext == nil || ranContext.Conn == nil {
		return fmt.Errorf("invalid RAN context")
	}

	pdu := &NGAPPDU{
		Type:          PDUTypeInitiatingMessage,
		ProcedureCode: ProcedureCodeOverloadStop,
		Criticality:   CriticalityIgnore,
		IEs:           []ProtocolIE{},
	}

	h.amfContext.IsOverloaded = false
	h.amfContext.OverloadAction = 0

	logger.NgapLog.Infof("Sending Overload Stop to RAN %s", ranContext.RanNodeName)

	return h.server.SendMessage(ranContext.Conn, pdu)
}

func (h *Handler) SendAMFConfigurationUpdate(ranContext *context.RANContext, amfName string) error {
	logger.NgapLog.Info("Sending AMF Configuration Update")

	if ranContext == nil || ranContext.Conn == nil {
		return fmt.Errorf("invalid RAN context")
	}

	ies := []ProtocolIE{}

	if amfName != "" {
		ies = append(ies, ProtocolIE{
			Id:          ProtocolIEIDAMFName,
			Criticality: CriticalityReject,
			Value:       []byte(amfName),
		})
	}

	if len(h.amfContext.ServedGuami) > 0 {
		var guamiData []byte
		for _, guami := range h.amfContext.ServedGuami {
			guamiData = append(guamiData, []byte(guami.PlmnId.Mcc)...)
			guamiData = append(guamiData, []byte(guami.PlmnId.Mnc)...)
			guamiData = append(guamiData, []byte(guami.AmfId)...)
		}
		if len(guamiData) > 0 {
			ies = append(ies, ProtocolIE{
				Id:          ProtocolIEIDServedGUAMIList,
				Criticality: CriticalityReject,
				Value:       guamiData,
			})
		}
	}

	relativeCapacity := byte(255)
	ies = append(ies, ProtocolIE{
		Id:          ProtocolIEIDRelativeAMFCapacity,
		Criticality: CriticalityIgnore,
		Value:       []byte{relativeCapacity},
	})

	if len(h.amfContext.PlmnSupportList) > 0 {
		var plmnData []byte
		for _, plmnSupport := range h.amfContext.PlmnSupportList {
			plmnData = append(plmnData, []byte(plmnSupport.PlmnId.Mcc)...)
			plmnData = append(plmnData, []byte(plmnSupport.PlmnId.Mnc)...)
			for _, snssai := range plmnSupport.SNssaiList {
				plmnData = append(plmnData, byte(snssai.Sst))
				plmnData = append(plmnData, []byte(snssai.Sd)...)
			}
		}
		if len(plmnData) > 0 {
			ies = append(ies, ProtocolIE{
				Id:          ProtocolIEIDPLMNSupportList,
				Criticality: CriticalityReject,
				Value:       plmnData,
			})
		}
	}

	pdu := &NGAPPDU{
		Type:          PDUTypeInitiatingMessage,
		ProcedureCode: ProcedureCodeAMFConfigurationUpdate,
		Criticality:   CriticalityReject,
		IEs:           ies,
	}

	logger.NgapLog.Infof("Sending AMF Configuration Update to RAN %s", ranContext.RanNodeName)

	return h.server.SendMessage(ranContext.Conn, pdu)
}

func (h *Handler) HandleAMFConfigurationUpdateAcknowledge(ranContext *context.RANContext, pdu *NGAPPDU) error {
	logger.NgapLog.Info("Handling AMF Configuration Update Acknowledge")

	for _, ie := range pdu.IEs {
		switch ie.Id {
		case ProtocolIEIDAMFTNLAssociationToAddList:
		case ProtocolIEIDAMFTNLAssociationToRemoveList:
		case ProtocolIEIDAMFTNLAssociationToUpdateList:
		case ProtocolIEIDCriticalityDiagnostics:
		}
	}

	logger.NgapLog.Infof("AMF Configuration Update acknowledged by RAN %s", ranContext.RanNodeName)

	return nil
}

func (h *Handler) HandleAMFConfigurationUpdateFailure(ranContext *context.RANContext, pdu *NGAPPDU) error {
	logger.NgapLog.Info("Handling AMF Configuration Update Failure")

	var cause *Cause
	var timeToWait int
	var criticalityDiagnostics []byte

	for _, ie := range pdu.IEs {
		switch ie.Id {
		case ProtocolIEIDCause:
			if data, ok := ie.Value.([]byte); ok && len(data) >= 2 {
				cause = &Cause{
					CauseGroup: int(data[0]),
					CauseValue: int(data[1]),
				}
			}
		case ProtocolIEIDCriticalityDiagnostics:
			if data, ok := ie.Value.([]byte); ok {
				criticalityDiagnostics = data
			}
		}
	}

	if cause != nil {
		logger.NgapLog.Errorf("AMF Configuration Update failed for RAN %s - Cause Group: %d, Cause Value: %d",
			ranContext.RanNodeName, cause.CauseGroup, cause.CauseValue)
	} else {
		logger.NgapLog.Errorf("AMF Configuration Update failed for RAN %s - Unknown cause", ranContext.RanNodeName)
	}

	if timeToWait > 0 {
		logger.NgapLog.Infof("Time to wait before retry: %d seconds", timeToWait)
	}

	if len(criticalityDiagnostics) > 0 {
		logger.NgapLog.Debugf("Criticality Diagnostics: %v", criticalityDiagnostics)
	}

	return nil
}

func (h *Handler) HandleRANConfigurationUpdate(ranContext *context.RANContext, pdu *NGAPPDU) error {
	logger.NgapLog.Info("Handling RAN Configuration Update")

	var ranNodeName *string
	var globalRANNodeID *GlobalRANNodeID
	var supportedTAList []SupportedTAItem
	var pagingDRX *int

	for _, ie := range pdu.IEs {
		switch ie.Id {
		case ProtocolIEIDRANNodeName:
			if data, ok := ie.Value.([]byte); ok {
				name := string(data)
				ranNodeName = &name
			}
		case ProtocolIEIDGlobalRANNodeID:
			if data, ok := ie.Value.([]byte); ok && len(data) >= 3 {
				globalRANNodeID = &GlobalRANNodeID{
					PLMNIdentity: data[:3],
					GNBID:        data[3:],
				}
			}
		case ProtocolIEIDSupportedTAList:
			if data, ok := ie.Value.([]byte); ok {
				if len(data) >= 6 {
					supportedTAList = []SupportedTAItem{
						{
							TAC: data[:3],
							BroadcastPLMNList: []BroadcastPLMNItem{
								{
									PLMNIdentity: data[3:6],
								},
							},
						},
					}
				}
			}
		case ProtocolIEIDDefaultPagingDRX:
			if data, ok := ie.Value.([]byte); ok && len(data) > 0 {
				drx := int(data[0])
				pagingDRX = &drx
			}
		}
	}

	if ranNodeName != nil {
		ranContext.RanNodeName = *ranNodeName
		logger.NgapLog.Infof("Updated RAN Node Name: %s", *ranNodeName)
	}

	if globalRANNodeID != nil {
		ranContext.GlobalRanNodeId = &context.GlobalRanNodeId{
			PlmnId: context.PlmnId{
				Mcc: string(globalRANNodeID.PLMNIdentity[:3]),
				Mnc: string(globalRANNodeID.PLMNIdentity[3:]),
			},
			GnbId: string(globalRANNodeID.GNBID),
		}
		logger.NgapLog.Info("Updated Global RAN Node ID")
	}

	if len(supportedTAList) > 0 {
		ranContext.SupportedTAList = []context.SupportedTAI{}
		for _, tai := range supportedTAList {
			ranContext.SupportedTAList = append(ranContext.SupportedTAList, context.SupportedTAI{
				Tai: context.Tai{
					PlmnId: context.PlmnId{
						Mcc: string(tai.TAC[:3]),
						Mnc: "01",
					},
					Tac: string(tai.TAC),
				},
			})
		}
		logger.NgapLog.Infof("Updated Supported TA List with %d entries", len(supportedTAList))
	}

	if pagingDRX != nil {
		ranContext.DefaultPagingDrx = fmt.Sprintf("%d", *pagingDRX)
		logger.NgapLog.Infof("Updated Default Paging DRX: %d", *pagingDRX)
	}

	logger.NgapLog.Infof("RAN Configuration Update completed for RAN: %s", ranContext.RanNodeName)

	return h.SendRANConfigurationUpdateAcknowledge(ranContext)
}

func (h *Handler) SendRANConfigurationUpdateAcknowledge(ranContext *context.RANContext) error {
	logger.NgapLog.Info("Sending RAN Configuration Update Acknowledge")

	if ranContext == nil || ranContext.Conn == nil {
		return fmt.Errorf("invalid RAN context")
	}

	pdu := &NGAPPDU{
		Type:          PDUTypeSuccessfulOutcome,
		ProcedureCode: ProcedureCodeRANConfigurationUpdate,
		Criticality:   CriticalityReject,
		IEs:           []ProtocolIE{},
	}

	logger.NgapLog.Infof("RAN Configuration Update Acknowledge sent to RAN %s", ranContext.RanNodeName)

	return h.server.SendMessage(ranContext.Conn, pdu)
}

func (h *Handler) HandleUERadioCapabilityInfoIndication(ranContext *context.RANContext, pdu *NGAPPDU) error {
	logger.NgapLog.Info("Handling UE Radio Capability Info Indication")

	var amfUeNgapId int64
	var ranUeNgapId int64
	var ueRadioCapability []byte
	var ueRadioCapabilityForPaging []byte

	for _, ie := range pdu.IEs {
		switch ie.Id {
		case ProtocolIEIDAMFUENGAPID:
			if val, ok := ie.Value.(int64); ok {
				amfUeNgapId = val
			}
		case ProtocolIEIDRANUENGAPID:
			if val, ok := ie.Value.(int64); ok {
				ranUeNgapId = val
			}
		case ProtocolIEIDUERadioCapability:
			if data, ok := ie.Value.([]byte); ok {
				ueRadioCapability = data
			}
		case ProtocolIEIDUERadioCapabilityForPaging:
			if data, ok := ie.Value.([]byte); ok {
				ueRadioCapabilityForPaging = data
			}
		}
	}

	ue, ok := h.amfContext.GetUEContextByAmfUeNgapId(amfUeNgapId)
	if !ok {
		return fmt.Errorf("UE context not found for AMF UE NGAP ID: %d", amfUeNgapId)
	}

	if ue.UeCapability == nil {
		ue.UeCapability = &context.UeCapability{}
	}

	ue.UeCapability.UeRadioCapability = ueRadioCapability
	ue.UeCapability.UeRadioCapabilityForPaging = ueRadioCapabilityForPaging

	logger.NgapLog.Infof("UE Radio Capability Info received for AMF UE NGAP ID=%d, RAN UE NGAP ID=%d, Capability length=%d, Paging Capability length=%d",
		amfUeNgapId, ranUeNgapId, len(ueRadioCapability), len(ueRadioCapabilityForPaging))

	return nil
}

func (h *Handler) SendUETNLABindingReleaseRequest(ue *context.UEContext) error {
	logger.NgapLog.Info("Sending UE TNLA Binding Release Request")

	if ue.RanContext == nil || ue.RanContext.Conn == nil {
		return fmt.Errorf("UE has no RAN connection")
	}

	pdu := &NGAPPDU{
		Type:          PDUTypeInitiatingMessage,
		ProcedureCode: ProcedureCodeUETNLABindingReleaseRequest,
		Criticality:   CriticalityIgnore,
		IEs: []ProtocolIE{
			{
				Id:          ProtocolIEIDAMFUENGAPID,
				Criticality: CriticalityReject,
				Value:       ue.AmfUeNgapId,
			},
			{
				Id:          ProtocolIEIDRANUENGAPID,
				Criticality: CriticalityReject,
				Value:       ue.RanUeNgapId,
			},
		},
	}

	if err := h.server.SendMessage(ue.RanContext.Conn, pdu); err != nil {
		return fmt.Errorf("failed to send UE TNLA Binding Release Request: %w", err)
	}

	logger.NgapLog.Infof("UE TNLA Binding Release Request sent for AMF UE NGAP ID=%d, RAN UE NGAP ID=%d",
		ue.AmfUeNgapId, ue.RanUeNgapId)

	return nil
}

func (h *Handler) SendTraceStart(ue *context.UEContext, traceReference []byte, traceDepth int, traceCollectionEntityIP []byte) error {
	logger.NgapLog.Info("Sending Trace Start")

	if ue.RanContext == nil || ue.RanContext.Conn == nil {
		return fmt.Errorf("UE has no RAN connection")
	}

	ngranTraceID := make([]byte, 8)
	copy(ngranTraceID[:3], ue.Tai.PlmnId.Mcc[:3])
	copy(ngranTraceID[3:], traceReference)

	traceActivation := encodeTraceActivation(ngranTraceID, traceDepth, traceCollectionEntityIP)

	pdu := &NGAPPDU{
		Type:          PDUTypeInitiatingMessage,
		ProcedureCode: ProcedureCodeTraceStart,
		Criticality:   CriticalityIgnore,
		IEs: []ProtocolIE{
			{
				Id:          ProtocolIEIDAMFUENGAPID,
				Criticality: CriticalityReject,
				Value:       ue.AmfUeNgapId,
			},
			{
				Id:          ProtocolIEIDRANUENGAPID,
				Criticality: CriticalityReject,
				Value:       ue.RanUeNgapId,
			},
			{
				Id:          ProtocolIEIDTraceActivation,
				Criticality: CriticalityIgnore,
				Value:       traceActivation,
			},
		},
	}

	if err := h.server.SendMessage(ue.RanContext.Conn, pdu); err != nil {
		return fmt.Errorf("failed to send Trace Start: %w", err)
	}

	logger.NgapLog.Infof("Trace Start sent for AMF UE NGAP ID=%d, RAN UE NGAP ID=%d, Trace Depth=%d",
		ue.AmfUeNgapId, ue.RanUeNgapId, traceDepth)

	return nil
}

func (h *Handler) SendDeactivateTrace(ue *context.UEContext, traceReference []byte) error {
	logger.NgapLog.Info("Sending Deactivate Trace")

	if ue.RanContext == nil || ue.RanContext.Conn == nil {
		return fmt.Errorf("UE has no RAN connection")
	}

	ngranTraceID := make([]byte, 8)
	copy(ngranTraceID[:3], ue.Tai.PlmnId.Mcc[:3])
	copy(ngranTraceID[3:], traceReference)

	pdu := &NGAPPDU{
		Type:          PDUTypeInitiatingMessage,
		ProcedureCode: ProcedureCodeDeactivateTrace,
		Criticality:   CriticalityIgnore,
		IEs: []ProtocolIE{
			{
				Id:          ProtocolIEIDAMFUENGAPID,
				Criticality: CriticalityReject,
				Value:       ue.AmfUeNgapId,
			},
			{
				Id:          ProtocolIEIDRANUENGAPID,
				Criticality: CriticalityReject,
				Value:       ue.RanUeNgapId,
			},
			{
				Id:          ProtocolIEIDTraceReference,
				Criticality: CriticalityIgnore,
				Value:       ngranTraceID,
			},
		},
	}

	if err := h.server.SendMessage(ue.RanContext.Conn, pdu); err != nil {
		return fmt.Errorf("failed to send Deactivate Trace: %w", err)
	}

	logger.NgapLog.Infof("Deactivate Trace sent for AMF UE NGAP ID=%d, RAN UE NGAP ID=%d",
		ue.AmfUeNgapId, ue.RanUeNgapId)

	return nil
}

func encodeTraceActivation(ngranTraceID []byte, traceDepth int, traceCollectionEntityIP []byte) []byte {
	result := make([]byte, 0)

	result = append(result, ngranTraceID...)

	result = append(result, 0xFF, 0xFF)

	result = append(result, byte(traceDepth))

	result = append(result, traceCollectionEntityIP...)

	return result
}

func (h *Handler) SendWriteReplaceWarningRequest(ranContext *context.RANContext, messageID uint16, serialNumber uint32, warningAreaList *WarningAreaList, repetitionPeriod uint32, numBroadcasts uint32, warningType []byte, warningMessage []byte, dataCodingScheme uint8) error {
	logger.NgapLog.Info("Sending Write-Replace Warning Request")

	if ranContext == nil || ranContext.Conn == nil {
		return fmt.Errorf("RAN has no connection")
	}

	msgIDBytes := make([]byte, 2)
	msgIDBytes[0] = byte(messageID >> 8)
	msgIDBytes[1] = byte(messageID)

	serialBytes := make([]byte, 2)
	serialBytes[0] = byte(serialNumber >> 8)
	serialBytes[1] = byte(serialNumber)

	ies := []ProtocolIE{
		{
			Id:          ProtocolIEIDMessageIdentifier,
			Criticality: CriticalityReject,
			Value:       msgIDBytes,
		},
		{
			Id:          ProtocolIEIDSerialNumber,
			Criticality: CriticalityReject,
			Value:       serialBytes,
		},
	}

	if warningAreaList != nil {
		warningAreaBytes := encodeWarningAreaList(warningAreaList)
		ies = append(ies, ProtocolIE{
			Id:          ProtocolIEIDWarningAreaList,
			Criticality: CriticalityIgnore,
			Value:       warningAreaBytes,
		})
	}

	if repetitionPeriod > 0 {
		repPeriodBytes := make([]byte, 4)
		repPeriodBytes[0] = byte(repetitionPeriod >> 24)
		repPeriodBytes[1] = byte(repetitionPeriod >> 16)
		repPeriodBytes[2] = byte(repetitionPeriod >> 8)
		repPeriodBytes[3] = byte(repetitionPeriod)
		ies = append(ies, ProtocolIE{
			Id:          ProtocolIEIDRepetitionPeriod,
			Criticality: CriticalityReject,
			Value:       repPeriodBytes,
		})
	}

	if numBroadcasts > 0 {
		numBroadcastsBytes := make([]byte, 2)
		numBroadcastsBytes[0] = byte(numBroadcasts >> 8)
		numBroadcastsBytes[1] = byte(numBroadcasts)
		ies = append(ies, ProtocolIE{
			Id:          ProtocolIEIDNumberOfBroadcastsRequested,
			Criticality: CriticalityReject,
			Value:       numBroadcastsBytes,
		})
	}

	if len(warningType) > 0 {
		ies = append(ies, ProtocolIE{
			Id:          ProtocolIEIDWarningType,
			Criticality: CriticalityIgnore,
			Value:       warningType,
		})
	}

	if len(warningMessage) > 0 {
		dcsBytes := []byte{dataCodingScheme}
		ies = append(ies, ProtocolIE{
			Id:          ProtocolIEIDDataCodingScheme,
			Criticality: CriticalityIgnore,
			Value:       dcsBytes,
		})

		ies = append(ies, ProtocolIE{
			Id:          ProtocolIEIDWarningMessageContents,
			Criticality: CriticalityIgnore,
			Value:       warningMessage,
		})
	}

	pdu := &NGAPPDU{
		Type:          PDUTypeInitiatingMessage,
		ProcedureCode: ProcedureCodeWriteReplaceWarning,
		Criticality:   CriticalityReject,
		IEs:           ies,
	}

	if err := h.server.SendMessage(ranContext.Conn, pdu); err != nil {
		return fmt.Errorf("failed to send Write-Replace Warning Request: %w", err)
	}

	logger.NgapLog.Infof("Write-Replace Warning Request sent to RAN %s, Message ID=%d, Serial Number=%d",
		ranContext.RanNodeName, messageID, serialNumber)

	return nil
}

func (h *Handler) HandleWriteReplaceWarningResponse(ranContext *context.RANContext, pdu *NGAPPDU) error {
	logger.NgapLog.Info("Handling Write-Replace Warning Response")

	var messageID uint16
	var serialNumber uint32
	var broadcastCompletedAreaList []byte

	for _, ie := range pdu.IEs {
		switch ie.Id {
		case ProtocolIEIDMessageIdentifier:
			if data, ok := ie.Value.([]byte); ok && len(data) >= 2 {
				messageID = uint16(data[0])<<8 | uint16(data[1])
			}
		case ProtocolIEIDSerialNumber:
			if data, ok := ie.Value.([]byte); ok && len(data) >= 2 {
				serialNumber = uint32(data[0])<<8 | uint32(data[1])
			}
		case 22:
			if data, ok := ie.Value.([]byte); ok {
				broadcastCompletedAreaList = data
			}
		}
	}

	logger.NgapLog.Infof("Write-Replace Warning Response received from RAN %s, Message ID=%d, Serial Number=%d, Completed Area List length=%d",
		ranContext.RanNodeName, messageID, serialNumber, len(broadcastCompletedAreaList))

	return nil
}

func encodeWarningAreaList(warningAreaList *WarningAreaList) []byte {
	result := make([]byte, 0)

	if len(warningAreaList.CellIDList) > 0 {
		for _, nrcgi := range warningAreaList.CellIDList {
			result = append(result, nrcgi.PLMNIdentity...)
			result = append(result, nrcgi.NRCellID...)
		}
	}

	if len(warningAreaList.TAIList) > 0 {
		for _, tai := range warningAreaList.TAIList {
			result = append(result, tai.PLMNIdentity...)
			result = append(result, tai.TAC...)
		}
	}

	if len(warningAreaList.EmergencyAreaIDList) > 0 {
		for _, eaid := range warningAreaList.EmergencyAreaIDList {
			result = append(result, eaid...)
		}
	}

	return result
}
