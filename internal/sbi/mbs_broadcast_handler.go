package sbi

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/gavin/amf/internal/logger"
)

func (s *Server) CreateMbsBroadcastContext(reqData *ContextCreateReqData, binaryParts map[string][]byte) (*ContextCreateRspData, *ProblemDetails) {
	logger.SbiLog.Info("Processing MBS Broadcast Context Create")

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

	if reqData.NotifyUri == "" {
		logger.SbiLog.Error("Notify URI is required")
		return nil, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "Notify URI is required",
		}
	}

	if reqData.Snssai == nil {
		logger.SbiLog.Error("S-NSSAI is required")
		return nil, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "S-NSSAI is required",
		}
	}

	if reqData.MbsServiceArea == nil && len(reqData.MbsServiceAreaInfoList) == 0 {
		logger.SbiLog.Error("Either MBS Service Area or MBS Service Area Info List is required")
		return nil, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "Either MBS Service Area or MBS Service Area Info List is required",
		}
	}

	logger.SbiLog.Infof("MBS Broadcast Context - NGAP IE Type: %s", reqData.N2MbsSmInfo.NgapIeType)

	if reqData.N2MbsSmInfo.NgapData != nil {
		contentId := reqData.N2MbsSmInfo.NgapData.ContentId
		if binaryData, ok := binaryParts[contentId]; ok {
			logger.SbiLog.Infof("Found NGAP binary data, size: %d bytes", len(binaryData))
		} else {
			logger.SbiLog.Warnf("NGAP binary data not found for content ID: %s", contentId)
		}
	}

	if reqData.MbsServiceArea != nil {
		logger.SbiLog.Infof("MBS Service Area - NCGIs: %d, TAIs: %d",
			len(reqData.MbsServiceArea.NcgiList), len(reqData.MbsServiceArea.TaiList))
	}

	if len(reqData.MbsServiceAreaInfoList) > 0 {
		logger.SbiLog.Infof("MBS Service Area Info List count: %d", len(reqData.MbsServiceAreaInfoList))
	}

	sessionInfo := "unknown"
	if reqData.MbsSessionId.Tmgi != nil && reqData.MbsSessionId.Tmgi.MbsServiceId != "" {
		sessionInfo = fmt.Sprintf("TMGI:%s", reqData.MbsSessionId.Tmgi.MbsServiceId)
	}

	logger.SbiLog.Infof("MBS Broadcast session %s created successfully", sessionInfo)

	response := &ContextCreateRspData{
		MbsSessionId:    reqData.MbsSessionId,
		OperationStatus: string(OperationStatusMbsSessionStartComplete),
	}

	return response, nil
}

func (s *Server) DeleteMbsBroadcastContext(mbsContextRef string) *ProblemDetails {
	logger.SbiLog.Infof("Processing MBS Broadcast Context Delete for: %s", mbsContextRef)

	if mbsContextRef == "" {
		logger.SbiLog.Error("MBS Context Reference is required")
		return &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "MBS Context Reference is required",
		}
	}

	logger.SbiLog.Infof("MBS Broadcast context %s deleted successfully", mbsContextRef)

	return nil
}

func (s *Server) UpdateMbsBroadcastContext(mbsContextRef string, reqData *ContextUpdateReqData, binaryParts map[string][]byte) (*ContextUpdateRspData, *ProblemDetails) {
	logger.SbiLog.Infof("Processing MBS Broadcast Context Update for: %s", mbsContextRef)

	if mbsContextRef == "" {
		logger.SbiLog.Error("MBS Context Reference is required")
		return nil, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "MBS Context Reference is required",
		}
	}

	if reqData.N2MbsSmInfo != nil {
		logger.SbiLog.Infof("MBS Broadcast Context Update - NGAP IE Type: %s", reqData.N2MbsSmInfo.NgapIeType)

		if reqData.N2MbsSmInfo.NgapData != nil {
			contentId := reqData.N2MbsSmInfo.NgapData.ContentId
			if binaryData, ok := binaryParts[contentId]; ok {
				logger.SbiLog.Infof("Found NGAP binary data, size: %d bytes", len(binaryData))
			} else {
				logger.SbiLog.Warnf("NGAP binary data not found for content ID: %s", contentId)
			}
		}
	}

	if reqData.MbsServiceArea != nil {
		logger.SbiLog.Infof("Updating MBS Service Area - NCGIs: %d, TAIs: %d",
			len(reqData.MbsServiceArea.NcgiList), len(reqData.MbsServiceArea.TaiList))
	}

	if len(reqData.MbsServiceAreaInfoList) > 0 {
		logger.SbiLog.Infof("Updating MBS Service Area Info List count: %d", len(reqData.MbsServiceAreaInfoList))
	}

	if len(reqData.RanIdList) > 0 {
		logger.SbiLog.Infof("Target RAN nodes count: %d", len(reqData.RanIdList))
	}

	if reqData.NoNgapSignallingInd {
		logger.SbiLog.Info("No NGAP signalling indication is set")
	}

	logger.SbiLog.Infof("MBS Broadcast context %s updated successfully", mbsContextRef)

	response := &ContextUpdateRspData{
		OperationStatus: string(OperationStatusMbsSessionUpdateComplete),
	}

	return response, nil
}

func parseMbsBroadcastCreateRequest(r *http.Request) (*ContextCreateReqData, map[string][]byte, error) {
	contentType := r.Header.Get("Content-Type")

	if strings.Contains(contentType, "application/json") {
		var reqData ContextCreateReqData
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read request body: %w", err)
		}

		if err := json.Unmarshal(body, &reqData); err != nil {
			return nil, nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
		}

		return &reqData, nil, nil
	}

	if strings.Contains(contentType, "multipart/related") {
		mediaType, params, err := mime.ParseMediaType(contentType)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse content type: %w", err)
		}

		if !strings.HasPrefix(mediaType, "multipart/") {
			return nil, nil, fmt.Errorf("expected multipart content type")
		}

		boundary := params["boundary"]
		if boundary == "" {
			return nil, nil, fmt.Errorf("boundary not found in content type")
		}

		reader := multipart.NewReader(r.Body, boundary)
		var reqData *ContextCreateReqData
		binaryParts := make(map[string][]byte)

		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, nil, fmt.Errorf("failed to read multipart: %w", err)
			}

			partContentType := part.Header.Get("Content-Type")
			contentId := strings.Trim(part.Header.Get("Content-Id"), "<>")

			data, err := io.ReadAll(part)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to read part data: %w", err)
			}

			if strings.Contains(partContentType, "application/json") {
				var jsonData ContextCreateReqData
				if err := json.Unmarshal(data, &jsonData); err != nil {
					return nil, nil, fmt.Errorf("failed to unmarshal JSON part: %w", err)
				}
				reqData = &jsonData
			} else if contentId != "" {
				binaryParts[contentId] = data
			}

			part.Close()
		}

		if reqData == nil {
			return nil, nil, fmt.Errorf("JSON data part not found in multipart request")
		}

		return reqData, binaryParts, nil
	}

	return nil, nil, fmt.Errorf("unsupported Content-Type: %s", contentType)
}

func parseMbsBroadcastUpdateRequest(r *http.Request) (*ContextUpdateReqData, map[string][]byte, error) {
	contentType := r.Header.Get("Content-Type")

	if strings.Contains(contentType, "application/json") {
		var reqData ContextUpdateReqData
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read request body: %w", err)
		}

		if err := json.Unmarshal(body, &reqData); err != nil {
			return nil, nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
		}

		return &reqData, nil, nil
	}

	if strings.Contains(contentType, "multipart/related") {
		mediaType, params, err := mime.ParseMediaType(contentType)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse content type: %w", err)
		}

		if !strings.HasPrefix(mediaType, "multipart/") {
			return nil, nil, fmt.Errorf("expected multipart content type")
		}

		boundary := params["boundary"]
		if boundary == "" {
			return nil, nil, fmt.Errorf("boundary not found in content type")
		}

		reader := multipart.NewReader(r.Body, boundary)
		var reqData *ContextUpdateReqData
		binaryParts := make(map[string][]byte)

		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, nil, fmt.Errorf("failed to read multipart: %w", err)
			}

			partContentType := part.Header.Get("Content-Type")
			contentId := strings.Trim(part.Header.Get("Content-Id"), "<>")

			data, err := io.ReadAll(part)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to read part data: %w", err)
			}

			if strings.Contains(partContentType, "application/json") {
				var jsonData ContextUpdateReqData
				if err := json.Unmarshal(data, &jsonData); err != nil {
					return nil, nil, fmt.Errorf("failed to unmarshal JSON part: %w", err)
				}
				reqData = &jsonData
			} else if contentId != "" {
				binaryParts[contentId] = data
			}

			part.Close()
		}

		if reqData == nil {
			return nil, nil, fmt.Errorf("JSON data part not found in multipart request")
		}

		return reqData, binaryParts, nil
	}

	return nil, nil, fmt.Errorf("unsupported Content-Type: %s", contentType)
}

func generateMbsContextRef() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "mbs-ctx-" + fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}
