package sbi

import (
	"fmt"
	"net/http"

	"github.com/gavin/amf/internal/context"
	"github.com/gavin/amf/internal/logger"
)

func (s *Server) N1N2MessageTransfer(ueContextId string, reqData *N1N2MessageTransferReqData, binaryParts map[string][]byte) (*N1N2MessageTransferRspData, *ProblemDetails) {
	logger.SbiLog.Infof("Processing N1N2 Message Transfer for UE: %s", ueContextId)

	ue := s.findUEContextById(ueContextId)
	if ue == nil {
		logger.SbiLog.Warnf("UE Context not found for ID: %s", ueContextId)
		return nil, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Not Found",
			Status: http.StatusNotFound,
			Detail: fmt.Sprintf("UE Context not found for ID: %s", ueContextId),
		}
	}

	if ue.CmState != context.CmConnected {
		logger.SbiLog.Warnf("UE %s not in CM-CONNECTED state", ueContextId)
		return &N1N2MessageTransferRspData{
			Cause: "ATTEMPTING_TO_REACH_UE",
		}, nil
	}

	if reqData.N1MessageContainer != nil && reqData.N1MessageContainer.N1MessageContent != nil {
		logger.SbiLog.Infof("Processing N1 message for UE %s", ueContextId)
		contentId := reqData.N1MessageContainer.N1MessageContent.ContentId
		if binaryData, ok := binaryParts[contentId]; ok {
			logger.SbiLog.Infof("Found N1 message binary data, size: %d bytes", len(binaryData))
		}
	}

	if reqData.N2InfoContainer != nil {
		logger.SbiLog.Infof("Processing N2 information for UE %s", ueContextId)

		if reqData.N2InfoContainer.SmInfo != nil {
			logger.SbiLog.Infof("N2 SM Information for PDU Session ID: %d", reqData.N2InfoContainer.SmInfo.PduSessionId)

			if reqData.N2InfoContainer.SmInfo.N2InfoContent != nil {
				contentId := reqData.N2InfoContainer.SmInfo.N2InfoContent.ContentId
				if binaryData, ok := binaryParts[contentId]; ok {
					logger.SbiLog.Infof("Found N2 SM info binary data, size: %d bytes", len(binaryData))
				}
			}
		}

		if reqData.N2InfoContainer.RanInfo != nil && reqData.N2InfoContainer.RanInfo.N2InfoContent != nil {
			logger.SbiLog.Infof("Processing N2 RAN information")
			contentId := reqData.N2InfoContainer.RanInfo.N2InfoContent.ContentId
			if binaryData, ok := binaryParts[contentId]; ok {
				logger.SbiLog.Infof("Found N2 RAN info binary data, size: %d bytes", len(binaryData))
			}
		}
	}

	if reqData.PduSessionId != 0 {
		logger.SbiLog.Infof("PDU Session ID specified: %d", reqData.PduSessionId)
		if _, exists := ue.PduSessions[reqData.PduSessionId]; !exists {
			logger.SbiLog.Warnf("PDU Session %d not found for UE %s", reqData.PduSessionId, ueContextId)
			return nil, &ProblemDetails{
				Type:   "about:blank",
				Title:  "Not Found",
				Status: http.StatusNotFound,
				Detail: fmt.Sprintf("PDU Session %d not found", reqData.PduSessionId),
			}
		}
	}

	logger.SbiLog.Infof("N1N2 Message Transfer initiated successfully for UE: %s", ueContextId)

	return &N1N2MessageTransferRspData{
		Cause: "N1_N2_TRANSFER_INITIATED",
	}, nil
}
