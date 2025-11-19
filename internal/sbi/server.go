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

	"github.com/gavin/amf/internal/context"
	"github.com/gavin/amf/internal/logger"
	"github.com/gavin/amf/pkg/factory"
)

type Server struct {
	httpServer *http.Server
	router     *http.ServeMux
	amfContext *context.AMFContext
}

func NewServer(ctx *context.AMFContext) *Server {
	return &Server{
		router:     http.NewServeMux(),
		amfContext: ctx,
	}
}

func (s *Server) Run() error {
	config := factory.GetConfig()
	if config == nil {
		return fmt.Errorf("configuration not loaded")
	}

	sbiConfig := config.Configuration.Sbi
	addr := fmt.Sprintf("%s:%d", sbiConfig.BindingIPv4, sbiConfig.Port)

	s.registerRoutes()

	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: s.router,
	}

	logger.SbiLog.Infof("Starting AMF SBI server on %s", addr)
	return s.httpServer.ListenAndServe()
}

func (s *Server) registerRoutes() {

	s.router.HandleFunc("/namf-comm/v1/ue-contexts", s.handleUEContexts)
	s.router.HandleFunc("/namf-comm/v1/ue-contexts/", s.handleUEContext)

	s.router.HandleFunc("/namf-evts/v1/subscriptions", s.handleEventSubscriptions)
	s.router.HandleFunc("/namf-evts/v1/subscriptions/", s.handleEventSubscription)

	s.router.HandleFunc("/namf-loc/v1/", s.handleLocationService)

	s.router.HandleFunc("/namf-mt/v1/ue-contexts/", s.handleMTService)

	s.router.HandleFunc("/health", s.handleHealthCheck)

	logger.SbiLog.Info("SBI routes registered")
}

func (s *Server) handleUEContexts(w http.ResponseWriter, r *http.Request) {
	logger.SbiLog.Infof("Handle UE Contexts: %s %s", r.Method, r.URL.Path)

	switch r.Method {
	case http.MethodGet:

		w.WriteHeader(http.StatusNotImplemented)
	default:
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Method Not Allowed",
			Status: http.StatusMethodNotAllowed,
			Detail: fmt.Sprintf("Method %s not allowed on this resource", r.Method),
		})
	}
}

func (s *Server) handleUEContext(w http.ResponseWriter, r *http.Request) {
	logger.SbiLog.Infof("Handle UE Context: %s %s", r.Method, r.URL.Path)

	path := r.URL.Path
	prefix := "/namf-comm/v1/ue-contexts/"
	if !strings.HasPrefix(path, prefix) {
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "Invalid path format",
		})
		return
	}

	pathAfterPrefix := strings.TrimPrefix(path, prefix)
	if pathAfterPrefix == "" {
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "UE Context ID is required",
		})
		return
	}

	parts := strings.Split(pathAfterPrefix, "/")
	ueContextId := parts[0]

	if len(parts) > 1 && parts[1] == "release" {
		if r.Method == http.MethodPost {
			s.handleReleaseUEContext(w, r, ueContextId)
		} else {
			sendProblemDetails(w, &ProblemDetails{
				Type:   "about:blank",
				Title:  "Method Not Allowed",
				Status: http.StatusMethodNotAllowed,
				Detail: fmt.Sprintf("Method %s not allowed on this resource", r.Method),
			})
		}
		return
	}

	if len(parts) > 1 && parts[1] == "n1-n2-messages" {
		if r.Method == http.MethodPost {
			s.handleN1N2MessageTransfer(w, r, ueContextId)
		} else {
			sendProblemDetails(w, &ProblemDetails{
				Type:   "about:blank",
				Title:  "Method Not Allowed",
				Status: http.StatusMethodNotAllowed,
				Detail: fmt.Sprintf("Method %s not allowed on this resource", r.Method),
			})
		}
		return
	}

	switch r.Method {
	case http.MethodPut:
		s.handleCreateUEContext(w, r, ueContextId)
	default:
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Method Not Allowed",
			Status: http.StatusMethodNotAllowed,
			Detail: fmt.Sprintf("Method %s not allowed on this resource", r.Method),
		})
	}
}

func (s *Server) handleN1N2MessageTransfer(w http.ResponseWriter, r *http.Request, ueContextId string) {
	logger.SbiLog.Infof("Handle N1N2 Message Transfer for UE: %s, %s %s", ueContextId, r.Method, r.URL.Path)

	if r.Method != http.MethodPost {
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Method Not Allowed",
			Status: http.StatusMethodNotAllowed,
			Detail: fmt.Sprintf("Method %s not allowed on this resource", r.Method),
		})
		return
	}

	transferData, binaryParts, err := parseN1N2TransferRequest(r)
	if err != nil {
		logger.SbiLog.Errorf("Failed to parse N1N2 transfer request: %v", err)
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: fmt.Sprintf("Failed to parse request: %v", err),
		})
		return
	}

	response, problemDetails := s.N1N2MessageTransfer(ueContextId, transferData, binaryParts)
	if problemDetails != nil {
		sendProblemDetails(w, problemDetails)
		return
	}

	location := fmt.Sprintf("/namf-comm/v1/ue-contexts/%s/n1-n2-messages/%s", ueContextId, generateTransferId())
	w.Header().Set("Location", location)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.SbiLog.Errorf("Failed to encode response: %v", err)
	}
}

func (s *Server) handleEventSubscriptions(w http.ResponseWriter, r *http.Request) {
	logger.SbiLog.Infof("Handle Event Subscriptions: %s %s", r.Method, r.URL.Path)

	switch r.Method {
	case http.MethodPost:
		s.handleCreateEventSubscription(w, r)
	default:
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Method Not Allowed",
			Status: http.StatusMethodNotAllowed,
			Detail: fmt.Sprintf("Method %s not allowed on this resource", r.Method),
		})
	}
}

func (s *Server) handleCreateEventSubscription(w http.ResponseWriter, r *http.Request) {
	logger.SbiLog.Info("Creating event subscription")

	var createData AmfCreateEventSubscription
	if err := json.NewDecoder(r.Body).Decode(&createData); err != nil {
		logger.SbiLog.Errorf("Failed to decode request body: %v", err)
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: fmt.Sprintf("Failed to decode request body: %v", err),
		})
		return
	}

	response, problemDetails := s.CreateEventSubscription(&createData)
	if problemDetails != nil {
		sendProblemDetails(w, problemDetails)
		return
	}

	location := fmt.Sprintf("/namf-evts/v1/subscriptions/%s", response.SubscriptionId)
	w.Header().Set("Location", location)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.SbiLog.Errorf("Failed to encode response: %v", err)
	}
}

func (s *Server) handleEventSubscription(w http.ResponseWriter, r *http.Request) {
	logger.SbiLog.Infof("Handle Event Subscription: %s %s", r.Method, r.URL.Path)

	path := r.URL.Path
	prefix := "/namf-evts/v1/subscriptions/"
	if !strings.HasPrefix(path, prefix) {
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "Invalid path format",
		})
		return
	}

	subscriptionId := strings.TrimPrefix(path, prefix)
	if subscriptionId == "" {
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "Subscription ID is required",
		})
		return
	}

	switch r.Method {
	case http.MethodDelete:
		s.handleDeleteEventSubscription(w, subscriptionId)
	default:
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Method Not Allowed",
			Status: http.StatusMethodNotAllowed,
			Detail: fmt.Sprintf("Method %s not allowed on this resource", r.Method),
		})
	}
}

func (s *Server) handleDeleteEventSubscription(w http.ResponseWriter, subscriptionId string) {
	logger.SbiLog.Infof("Deleting event subscription: %s", subscriptionId)

	problemDetails := s.DeleteEventSubscription(subscriptionId)
	if problemDetails != nil {
		sendProblemDetails(w, problemDetails)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleLocationService(w http.ResponseWriter, r *http.Request) {
	logger.SbiLog.Infof("Handle Location Service: %s %s", r.Method, r.URL.Path)

	path := r.URL.Path
	prefix := "/namf-loc/v1/"
	if !strings.HasPrefix(path, prefix) {
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "Invalid path format",
		})
		return
	}

	pathAfterPrefix := strings.TrimPrefix(path, prefix)
	if pathAfterPrefix == "" {
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "UE Context ID is required",
		})
		return
	}

	parts := strings.Split(pathAfterPrefix, "/")
	if len(parts) < 2 {
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "Invalid location service path",
		})
		return
	}

	ueContextId := parts[0]
	operation := parts[1]

	if operation == "provide-loc-info" && r.Method == http.MethodPost {
		s.handleProvideLocationInfo(w, r, ueContextId)
	} else {
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Method Not Allowed",
			Status: http.StatusMethodNotAllowed,
			Detail: fmt.Sprintf("Method %s not allowed on this resource", r.Method),
		})
	}
}

func (s *Server) handleProvideLocationInfo(w http.ResponseWriter, r *http.Request, ueContextId string) {
	logger.SbiLog.Infof("Provide location info for UE: %s", ueContextId)

	var requestData RequestLocInfo
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		logger.SbiLog.Errorf("Failed to decode request body: %v", err)
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: fmt.Sprintf("Failed to decode request body: %v", err),
		})
		return
	}

	response, problemDetails := s.ProvideLocationInfo(ueContextId, &requestData)
	if problemDetails != nil {
		sendProblemDetails(w, problemDetails)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.SbiLog.Errorf("Failed to encode response: %v", err)
	}
}

func (s *Server) handleMTService(w http.ResponseWriter, r *http.Request) {
	logger.SbiLog.Infof("Handle MT Service: %s %s", r.Method, r.URL.Path)

	path := r.URL.Path
	prefix := "/namf-mt/v1/"
	if !strings.HasPrefix(path, prefix) {
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "Invalid path format",
		})
		return
	}

	pathAfterPrefix := strings.TrimPrefix(path, prefix)
	if pathAfterPrefix == "" {
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "Resource path is required",
		})
		return
	}

	parts := strings.Split(pathAfterPrefix, "/")

	if parts[0] == "ue-contexts" && len(parts) > 1 {
		if len(parts) == 2 && r.Method == http.MethodGet {
			ueContextId := parts[1]
			s.handleProvideDomainSelectionInfo(w, r, ueContextId)
			return
		}

		if len(parts) == 3 && parts[2] == "ue-reachind" && r.Method == http.MethodPut {
			ueContextId := parts[1]
			s.handleEnableUEReachability(w, r, ueContextId)
			return
		}

		if parts[1] == "enable-group-reachability" && r.Method == http.MethodPost {
			s.handleEnableGroupReachability(w, r)
			return
		}
	}

	sendProblemDetails(w, &ProblemDetails{
		Type:   "about:blank",
		Title:  "Method Not Allowed",
		Status: http.StatusMethodNotAllowed,
		Detail: fmt.Sprintf("Method %s not allowed on this resource", r.Method),
	})
}

func (s *Server) handleProvideDomainSelectionInfo(w http.ResponseWriter, r *http.Request, ueContextId string) {
	logger.SbiLog.Infof("Provide Domain Selection Info for UE: %s", ueContextId)

	infoClass := r.URL.Query().Get("info-class")
	supportedFeatures := r.URL.Query().Get("supported-features")
	oldGuamiStr := r.URL.Query().Get("old-guami")

	var oldGuami *Guami
	if oldGuamiStr != "" {
		if err := json.Unmarshal([]byte(oldGuamiStr), &oldGuami); err != nil {
			logger.SbiLog.Errorf("Failed to parse old-guami: %v", err)
		}
	}

	response, problemDetails := s.ProvideDomainSelectionInfo(ueContextId, infoClass, supportedFeatures, oldGuami)
	if problemDetails != nil {
		sendProblemDetails(w, problemDetails)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.SbiLog.Errorf("Failed to encode response: %v", err)
	}
}

func (s *Server) handleEnableUEReachability(w http.ResponseWriter, r *http.Request, ueContextId string) {
	logger.SbiLog.Infof("Enable UE Reachability for UE: %s", ueContextId)

	var reqData EnableUeReachabilityReqData
	if err := json.NewDecoder(r.Body).Decode(&reqData); err != nil {
		logger.SbiLog.Errorf("Failed to decode request body: %v", err)
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: fmt.Sprintf("Failed to decode request body: %v", err),
		})
		return
	}

	response, problemDetails := s.EnableUEReachability(ueContextId, &reqData)
	if problemDetails != nil {
		sendProblemDetails(w, problemDetails)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.SbiLog.Errorf("Failed to encode response: %v", err)
	}
}

func (s *Server) handleEnableGroupReachability(w http.ResponseWriter, r *http.Request) {
	logger.SbiLog.Info("Enable Group Reachability")

	var reqData EnableGroupReachabilityReqData
	if err := json.NewDecoder(r.Body).Decode(&reqData); err != nil {
		logger.SbiLog.Errorf("Failed to decode request body: %v", err)
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: fmt.Sprintf("Failed to decode request body: %v", err),
		})
		return
	}

	response, problemDetails := s.EnableGroupReachability(&reqData)
	if problemDetails != nil {
		sendProblemDetails(w, problemDetails)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.SbiLog.Errorf("Failed to encode response: %v", err)
	}
}

func (s *Server) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (s *Server) handleCreateUEContext(w http.ResponseWriter, r *http.Request, ueContextId string) {
	logger.SbiLog.Infof("Creating UE Context for ID: %s", ueContextId)

	createData, _, err := parseMultipartRequest(r)
	if err != nil {
		logger.SbiLog.Errorf("Failed to parse request: %v", err)
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: fmt.Sprintf("Failed to parse request: %v", err),
		})
		return
	}

	response, problemDetails := s.CreateUEContext(ueContextId, createData)
	if problemDetails != nil {
		sendProblemDetails(w, problemDetails)
		return
	}

	location := fmt.Sprintf("/namf-comm/v1/ue-contexts/%s", ueContextId)
	w.Header().Set("Location", location)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.SbiLog.Errorf("Failed to encode response: %v", err)
	}
}

func (s *Server) handleReleaseUEContext(w http.ResponseWriter, r *http.Request, ueContextId string) {
	logger.SbiLog.Infof("Releasing UE Context for ID: %s", ueContextId)

	var releaseData UEContextRelease
	if err := json.NewDecoder(r.Body).Decode(&releaseData); err != nil {
		logger.SbiLog.Errorf("Failed to decode request body: %v", err)
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: fmt.Sprintf("Failed to decode request body: %v", err),
		})
		return
	}

	if releaseData.NgapCause == nil {
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "ngapCause is required",
		})
		return
	}

	problemDetails := s.ReleaseUEContext(ueContextId, &releaseData)
	if problemDetails != nil {
		sendProblemDetails(w, problemDetails)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func sendProblemDetails(w http.ResponseWriter, problem *ProblemDetails) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(problem.Status)

	if err := json.NewEncoder(w).Encode(problem); err != nil {
		logger.SbiLog.Errorf("Failed to encode problem details: %v", err)
	}
}

func generateTransferId() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "transfer-" + fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

func parseN1N2TransferRequest(r *http.Request) (*N1N2MessageTransferReqData, map[string][]byte, error) {
	contentType := r.Header.Get("Content-Type")

	if strings.Contains(contentType, "application/json") {
		var reqData N1N2MessageTransferReqData
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
		var reqData *N1N2MessageTransferReqData
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
				var jsonData N1N2MessageTransferReqData
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
