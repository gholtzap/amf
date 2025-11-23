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

		if s.ngapHandler != nil {
			if err := s.ngapHandler.SendPaging(ue); err != nil {
				logger.SbiLog.Errorf("Failed to send paging for UE %s: %v", ueContextId, err)
				return nil, &ProblemDetails{
					Type:   "about:blank",
					Title:  "Internal Server Error",
					Status: http.StatusInternalServerError,
					Detail: fmt.Sprintf("Failed to send paging: %v", err),
				}
			}
			logger.SbiLog.Infof("Paging initiated for UE %s", ueContextId)
		}

		response.Reachability = string(UeReachabilityREACHABLE)

		logger.SbiLog.Infof("UE %s paging process started", ueContextId)
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

func (s *Server) EnableGroupReachability(reqData *EnableGroupReachabilityReqData) (*EnableGroupReachabilityRspData, *ProblemDetails) {
	logger.SbiLog.Infof("Enabling group reachability for TMGI: %s-%s-%s",
		reqData.Tmgi.PlmnId.Mcc, reqData.Tmgi.PlmnId.Mnc, reqData.Tmgi.MbsServiceId)

	if len(reqData.UeInfoList) == 0 {
		logger.SbiLog.Error("UeInfoList is required and cannot be empty")
		return nil, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "ueInfoList is required and cannot be empty",
		}
	}

	if reqData.Tmgi == nil {
		logger.SbiLog.Error("TMGI is required")
		return nil, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "tmgi is required",
		}
	}

	var connectedUeList []string

	for _, ueInfo := range reqData.UeInfoList {
		if len(ueInfo.UeList) == 0 {
			continue
		}

		for _, supi := range ueInfo.UeList {
			ue := s.findUEContextById(supi)
			if ue == nil {
				logger.SbiLog.Warnf("UE not found for SUPI: %s", supi)
				continue
			}

			if ue.CmState == context.CmIdle {
				logger.SbiLog.Infof("Paging UE %s to make it reachable for MBS session", supi)
				if s.ngapHandler != nil {
					if err := s.ngapHandler.SendPaging(ue); err != nil {
						logger.SbiLog.Errorf("Failed to send paging for UE %s: %v", supi, err)
						continue
					}
				}
			}

			if ue.CmState == context.CmConnected {
				connectedUeList = append(connectedUeList, supi)
				logger.SbiLog.Infof("UE %s is now connected for MBS session", supi)
			}
		}
	}

	response := &EnableGroupReachabilityRspData{
		UeConnectedList:   connectedUeList,
		SupportedFeatures: reqData.SupportedFeatures,
	}

	if reqData.ReachabilityNotifyUri != "" {
		logger.SbiLog.Infof("Reachability notification URI provided: %s", reqData.ReachabilityNotifyUri)
	}

	if reqData.FiveQi != 0 {
		logger.SbiLog.Infof("5QI specified for group reachability: %d", reqData.FiveQi)
	}

	logger.SbiLog.Infof("Group reachability enabled, %d UEs connected out of total requested", len(connectedUeList))
	return response, nil
}
