package sbi

import (
	"fmt"
	"net/http"

	"github.com/gavin/amf/internal/logger"
)

func (s *Server) MbsN2MessageTransfer(reqData *MbsN2MessageTransferReqData, binaryParts map[string][]byte) (*MbsN2MessageTransferRspData, *ProblemDetails) {
	logger.SbiLog.Info("Processing MBS N2 Message Transfer")

	if reqData.MbsSessionId == nil {
		logger.SbiLog.Error("MBS Session ID is required")
		return nil, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "MBS Session ID is required",
		}
	}

	if reqData.N2MbsSmInfo == nil {
		logger.SbiLog.Error("N2 MBS SM Info is required")
		return nil, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "N2 MBS SM Info is required",
		}
	}

	if reqData.N2MbsSmInfo.NgapData == nil {
		logger.SbiLog.Error("NGAP Data is required in N2 MBS SM Info")
		return nil, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "NGAP Data is required",
		}
	}

	logger.SbiLog.Infof("MBS Session - NGAP IE Type: %s", reqData.N2MbsSmInfo.NgapIeType)

	contentId := reqData.N2MbsSmInfo.NgapData.ContentId
	if binaryData, ok := binaryParts[contentId]; ok {
		logger.SbiLog.Infof("Found NGAP binary data, size: %d bytes", len(binaryData))
	} else {
		logger.SbiLog.Warnf("NGAP binary data not found for content ID: %s", contentId)
	}

	if len(reqData.RanNodeIdList) > 0 {
		logger.SbiLog.Infof("Target RAN nodes count: %d", len(reqData.RanNodeIdList))
	} else {
		logger.SbiLog.Info("No specific RAN nodes targeted, broadcasting to all")
	}

	if reqData.NotifyUri != "" {
		logger.SbiLog.Infof("Notification URI provided: %s", reqData.NotifyUri)
	}

	var failureList []RanFailure
	for i, ranNode := range reqData.RanNodeIdList {
		if ranNode.PlmnId == nil {
			logger.SbiLog.Warnf("RAN node %d has no PLMN ID, adding to failure list", i)
			failureList = append(failureList, RanFailure{
				RanId: &ranNode,
				RanFailureCause: &NgApCause{
					Group: 0,
					Value: 1,
				},
			})
		}
	}

	result := "SUCCESS"
	if len(failureList) > 0 {
		result = "FAILURE"
		if len(failureList) < len(reqData.RanNodeIdList) {
			result = "PARTIAL_SUCCESS"
		}
	}

	logger.SbiLog.Infof("MBS N2 Message Transfer completed with result: %s", result)

	response := &MbsN2MessageTransferRspData{
		Result:            result,
		SupportedFeatures: reqData.SupportedFeatures,
	}

	if len(failureList) > 0 {
		response.FailureList = failureList
	}

	sessionInfo := "unknown"
	if reqData.MbsSessionId.Tmgi != nil && reqData.MbsSessionId.Tmgi.MbsServiceId != "" {
		sessionInfo = fmt.Sprintf("TMGI:%s", reqData.MbsSessionId.Tmgi.MbsServiceId)
	}
	logger.SbiLog.Infof("MBS session %s processing complete", sessionInfo)

	return response, nil
}
