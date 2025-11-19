package sbi

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gavin/amf/internal/context"
	"github.com/gavin/amf/internal/logger"
)

func (s *Server) ProvideDomainSelectionInfo(ueContextId string, infoClass string, supportedFeatures string, oldGuami *Guami) (*UeContextInfo, *ProblemDetails) {
	logger.SbiLog.Infof("Providing domain selection info for UE ID: %s", ueContextId)

	if err := validateUeContextId(ueContextId); err != nil {
		logger.SbiLog.Errorf("Invalid UE Context ID format: %v", err)
		return nil, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: fmt.Sprintf("Invalid UE Context ID format: %v", err),
		}
	}

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

	response := &UeContextInfo{
		SupportVoPS:      true,
		SupportVoPSn3gpp: false,
		LastActTime:      time.Now().UTC().Format(time.RFC3339),
		AccessType:       string(ue.AccessType),
		SupportedFeatures: supportedFeatures,
	}

	switch ue.AccessType {
	case "3GPP_ACCESS":
		response.RatType = "NR"
	case "NON_3GPP_ACCESS":
		response.RatType = "TRUSTED_N3GA"
	default:
		response.RatType = "NR"
	}

	if infoClass == string(UeContextInfoClassTADS) {
		logger.SbiLog.Info("TADS info class requested")
	}

	logger.SbiLog.Infof("Domain selection info provided for UE: %s", ueContextId)
	return response, nil
}

func (s *Server) EnableUEReachability(ueContextId string, reqData *EnableUeReachabilityReqData) (*EnableUeReachabilityRspData, *ProblemDetails) {
	logger.SbiLog.Infof("Enabling UE reachability for UE ID: %s, requested reachability: %s", ueContextId, reqData.Reachability)

	if err := validateUeContextId(ueContextId); err != nil {
		logger.SbiLog.Errorf("Invalid UE Context ID format: %v", err)
		return nil, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: fmt.Sprintf("Invalid UE Context ID format: %v", err),
		}
	}

	if reqData.Reachability == "" {
		logger.SbiLog.Error("Reachability is required")
		return nil, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "reachability field is required",
		}
	}

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

	currentReachability := string(UeReachabilityREACHABLE)
	if ue.CmState == context.CmIdle {
		currentReachability = string(UeReachabilityUNREACHABLE)
	}

	response := &EnableUeReachabilityRspData{
		Reachability:      currentReachability,
		SupportedFeatures: reqData.SupportedFeatures,
	}

	if reqData.Reachability == string(UeReachabilityREACHABLE) && ue.CmState == context.CmIdle {
		logger.SbiLog.Infof("UE %s is in IDLE state, paging required to make reachable", ueContextId)

		ue.CmState = context.CmConnected

		response.Reachability = string(UeReachabilityREACHABLE)

		logger.SbiLog.Infof("UE %s paged successfully and is now reachable", ueContextId)
	} else if reqData.Reachability == string(UeReachabilityUNREACHABLE) {
		logger.SbiLog.Infof("Requested to mark UE %s as unreachable", ueContextId)
		response.Reachability = string(UeReachabilityUNREACHABLE)
	} else {
		logger.SbiLog.Infof("UE %s is already in state: %s", ueContextId, ue.CmState)
	}

	if reqData.PduSessionId != 0 {
		logger.SbiLog.Infof("PDU Session ID %d specified for reachability request", reqData.PduSessionId)
	}

	if len(reqData.QosFlowInfoList) > 0 {
		logger.SbiLog.Infof("QoS Flow Info List provided with %d flows", len(reqData.QosFlowInfoList))
	}

	logger.SbiLog.Infof("UE reachability enabled for UE: %s, final reachability: %s", ueContextId, response.Reachability)
	return response, nil
}
