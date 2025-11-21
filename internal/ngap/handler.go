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
