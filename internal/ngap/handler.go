package ngap

import (
	"fmt"

	"github.com/gavin/amf/internal/context"
	"github.com/gavin/amf/internal/logger"
)

type Handler struct {
	amfContext *context.AMFContext
	server     *Server
}

func NewHandler(ctx *context.AMFContext) *Handler {
	return &Handler{
		amfContext: ctx,
	}
}

func (h *Handler) SetServer(server *Server) {
	h.server = server
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

	_ = ue

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
