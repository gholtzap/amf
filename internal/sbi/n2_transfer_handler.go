package sbi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gavin/amf/internal/logger"
)

func (s *Server) NonUeN2MessageTransfer(reqData *N2InformationTransferReqData, binaryParts map[string][]byte) (*N2InformationTransferRspData, *ProblemDetails) {
	logger.SbiLog.Info("Processing Non-UE N2 Message Transfer")

	if reqData.N2Information == nil {
		logger.SbiLog.Warn("N2 Information is required")
		return nil, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "N2 Information is required",
		}
	}

	if len(reqData.TaiList) == 0 && len(reqData.GlobalRanNodeList) == 0 {
		logger.SbiLog.Warn("Either TAI list or RAN node list must be provided")
		return nil, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "Either TAI list or RAN node list must be provided",
		}
	}

	if len(reqData.TaiList) > 0 {
		logger.SbiLog.Infof("Processing N2 transfer for %d TAIs", len(reqData.TaiList))
		for i, tai := range reqData.TaiList {
			logger.SbiLog.Infof("TAI[%d]: MCC=%s, MNC=%s, TAC=%s", i, tai.PlmnId.Mcc, tai.PlmnId.Mnc, tai.Tac)
		}
	}

	if len(reqData.GlobalRanNodeList) > 0 {
		logger.SbiLog.Infof("Processing N2 transfer for %d RAN nodes", len(reqData.GlobalRanNodeList))
	}

	if reqData.RatSelector != "" {
		logger.SbiLog.Infof("RAT selector: %s", reqData.RatSelector)
	}

	if reqData.N2Information.SmInfo != nil && reqData.N2Information.SmInfo.N2InfoContent != nil {
		contentId := reqData.N2Information.SmInfo.N2InfoContent.ContentId
		if binaryData, ok := binaryParts[contentId]; ok {
			logger.SbiLog.Infof("Found N2 SM info binary data, size: %d bytes", len(binaryData))
		}
	}

	if reqData.N2Information.RanInfo != nil && reqData.N2Information.RanInfo.N2InfoContent != nil {
		contentId := reqData.N2Information.RanInfo.N2InfoContent.ContentId
		if binaryData, ok := binaryParts[contentId]; ok {
			logger.SbiLog.Infof("Found N2 RAN info binary data, size: %d bytes", len(binaryData))
		}
	}

	if reqData.N2Information.PwsInfo != nil && reqData.N2Information.PwsInfo.PwsContainer != nil {
		contentId := reqData.N2Information.PwsInfo.PwsContainer.ContentId
		if binaryData, ok := binaryParts[contentId]; ok {
			logger.SbiLog.Infof("Found PWS container binary data, size: %d bytes", len(binaryData))
		}
	}

	logger.SbiLog.Info("Non-UE N2 Message Transfer initiated successfully")

	response := &N2InformationTransferRspData{
		Result:            string(N2InformationTransferResultINITIATED),
		SupportedFeatures: reqData.SupportedFeatures,
	}

	if reqData.N2NotifyCallbackUri != "" {
		logger.SbiLog.Infof("N2 transfer result callback will be sent to: %s", reqData.N2NotifyCallbackUri)
		go s.sendN2TransferResultNotification(reqData.N2NotifyCallbackUri, response)
	}

	return response, nil
}

func (s *Server) sendN2TransferResultNotification(callbackUri string, result *N2InformationTransferRspData) {
	notification := &N2InfoNotification{
		Result:             result.Result,
		PwsRspData:         result.PwsRspData,
		TssRspPerNgranList: result.TssRspPerNgranList,
	}

	jsonData, err := json.Marshal(notification)
	if err != nil {
		logger.SbiLog.Errorf("Failed to marshal N2 transfer notification: %v", err)
		return
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("POST", callbackUri, bytes.NewBuffer(jsonData))
	if err != nil {
		logger.SbiLog.Errorf("Failed to create N2 transfer notification request: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		logger.SbiLog.Errorf("Failed to send N2 transfer notification to %s: %v", callbackUri, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		logger.SbiLog.Infof("N2 transfer notification sent successfully to %s (status: %d)", callbackUri, resp.StatusCode)
	} else {
		logger.SbiLog.Warnf("N2 transfer notification received non-success response from %s: %d", callbackUri, resp.StatusCode)
	}
}
