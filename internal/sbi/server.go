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

	s.router.HandleFunc("/namf-comm/v1/non-ue-n2-messages/transfer", s.handleNonUeN2MessageTransfer)

	s.router.HandleFunc("/namf-comm/v1/non-ue-n2-messages/subscriptions", s.handleNonUeN2MessageSubscriptions)
	s.router.HandleFunc("/namf-comm/v1/non-ue-n2-messages/subscriptions/", s.handleNonUeN2MessageSubscription)

	s.router.HandleFunc("/namf-comm/v1/subscriptions", s.handleAMFStatusSubscriptions)
	s.router.HandleFunc("/namf-comm/v1/subscriptions/", s.handleAMFStatusSubscription)

	s.router.HandleFunc("/namf-evts/v1/subscriptions", s.handleEventSubscriptions)
	s.router.HandleFunc("/namf-evts/v1/subscriptions/", s.handleEventSubscription)

	s.router.HandleFunc("/namf-loc/v1/", s.handleLocationService)

	s.router.HandleFunc("/namf-mt/v1/ue-contexts/", s.handleMTService)

	s.router.HandleFunc("/namf-mbs-comm/v1/n2-messages/transfer", s.handleMbsN2MessageTransfer)

	s.router.HandleFunc("/namf-mbs-bc/v1/mbs-contexts", s.handleMbsBroadcastContexts)
	s.router.HandleFunc("/namf-mbs-bc/v1/mbs-contexts/", s.handleMbsBroadcastContext)

	s.router.HandleFunc("/health", s.handleHealthCheck)

	logger.SbiLog.Info("SBI routes registered")
}

func (s *Server) handleUEContexts(w http.ResponseWriter, r *http.Request) {
	logger.SbiLog.Infof("Handle UE Contexts: %s %s", r.Method, r.URL.Path)

	switch r.Method {
	case http.MethodGet:
		s.handleQueryUEContexts(w, r)
	default:
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Method Not Allowed",
			Status: http.StatusMethodNotAllowed,
			Detail: fmt.Sprintf("Method %s not allowed on this resource", r.Method),
		})
	}
}

func (s *Server) handleQueryUEContexts(w http.ResponseWriter, r *http.Request) {
	logger.SbiLog.Info("Querying UE Contexts")

	supi := r.URL.Query().Get("supi")
	gpsi := r.URL.Query().Get("gpsi")

	result, problemDetails := s.QueryUEContexts(supi, gpsi)
	if problemDetails != nil {
		sendProblemDetails(w, problemDetails)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(result); err != nil {
		logger.SbiLog.Errorf("Failed to encode response: %v", err)
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
		if len(parts) == 2 {
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
		if len(parts) == 3 && parts[2] == "subscriptions" {
			if r.Method == http.MethodPost {
				s.handleN1N2MessageSubscribe(w, r, ueContextId)
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
		if len(parts) == 4 && parts[2] == "subscriptions" {
			subscriptionId := parts[3]
			if r.Method == http.MethodDelete {
				s.handleN1N2MessageUnSubscribe(w, ueContextId, subscriptionId)
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
	}

	if len(parts) > 1 && parts[1] == "assign-ebi" {
		if r.Method == http.MethodPost {
			s.handleAssignEbi(w, r, ueContextId)
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

	if len(parts) > 1 && parts[1] == "transfer" {
		if r.Method == http.MethodPost {
			s.handleUEContextTransfer(w, r, ueContextId)
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

	if len(parts) > 1 && parts[1] == "transfer-update" {
		if r.Method == http.MethodPost {
			s.handleRegistrationStatusUpdate(w, r, ueContextId)
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

	if len(parts) > 1 && parts[1] == "relocate" {
		if r.Method == http.MethodPost {
			s.handleRelocateUEContext(w, r, ueContextId)
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

	if len(parts) > 1 && parts[1] == "cancel-relocate" {
		if r.Method == http.MethodPost {
			s.handleCancelRelocateUEContext(w, r, ueContextId)
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
	case http.MethodGet:
		s.handleGetUEContext(w, r, ueContextId)
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
	case http.MethodPatch:
		s.handleModifyEventSubscription(w, r, subscriptionId)
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

func (s *Server) handleModifyEventSubscription(w http.ResponseWriter, r *http.Request, subscriptionId string) {
	logger.SbiLog.Infof("Modifying event subscription: %s", subscriptionId)

	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json-patch+json" {
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Unsupported Media Type",
			Status: http.StatusUnsupportedMediaType,
			Detail: "Content-Type must be application/json-patch+json",
		})
		return
	}

	var patchItems []AmfUpdateEventSubscriptionItem
	if err := json.NewDecoder(r.Body).Decode(&patchItems); err != nil {
		logger.SbiLog.Errorf("Failed to decode request body: %v", err)
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: fmt.Sprintf("Failed to decode request body: %v", err),
		})
		return
	}

	response, problemDetails := s.ModifyEventSubscription(subscriptionId, patchItems)
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
	} else if operation == "provide-pos-info" && r.Method == http.MethodPost {
		s.handleProvidePositioningInfo(w, r, ueContextId)
	} else if operation == "cancel-pos-info" && r.Method == http.MethodPost {
		s.handleCancelPositioningInfo(w, r, ueContextId)
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

func (s *Server) handleProvidePositioningInfo(w http.ResponseWriter, r *http.Request, ueContextId string) {
	logger.SbiLog.Infof("Provide positioning info for UE: %s", ueContextId)

	var requestData RequestPosInfo
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

	response, problemDetails := s.ProvidePositioningInfo(ueContextId, &requestData)
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

func (s *Server) handleCancelPositioningInfo(w http.ResponseWriter, r *http.Request, ueContextId string) {
	logger.SbiLog.Infof("Cancel positioning info for UE: %s", ueContextId)

	var requestData CancelPosInfo
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

	problemDetails := s.CancelLocation(ueContextId, &requestData)
	if problemDetails != nil {
		sendProblemDetails(w, problemDetails)
		return
	}

	w.WriteHeader(http.StatusNoContent)
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

func (s *Server) handleNonUeN2MessageTransfer(w http.ResponseWriter, r *http.Request) {
	logger.SbiLog.Infof("Handle Non-UE N2 Message Transfer: %s %s", r.Method, r.URL.Path)

	if r.Method != http.MethodPost {
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Method Not Allowed",
			Status: http.StatusMethodNotAllowed,
			Detail: fmt.Sprintf("Method %s not allowed on this resource", r.Method),
		})
		return
	}

	transferData, binaryParts, err := parseN2TransferRequest(r)
	if err != nil {
		logger.SbiLog.Errorf("Failed to parse Non-UE N2 transfer request: %v", err)
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: fmt.Sprintf("Failed to parse request: %v", err),
		})
		return
	}

	response, problemDetails := s.NonUeN2MessageTransfer(transferData, binaryParts)
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

func (s *Server) handleMbsN2MessageTransfer(w http.ResponseWriter, r *http.Request) {
	logger.SbiLog.Infof("Handle MBS N2 Message Transfer: %s %s", r.Method, r.URL.Path)

	if r.Method != http.MethodPost {
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Method Not Allowed",
			Status: http.StatusMethodNotAllowed,
			Detail: fmt.Sprintf("Method %s not allowed on this resource", r.Method),
		})
		return
	}

	transferData, binaryParts, err := parseMbsN2TransferRequest(r)
	if err != nil {
		logger.SbiLog.Errorf("Failed to parse MBS N2 transfer request: %v", err)
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: fmt.Sprintf("Failed to parse request: %v", err),
		})
		return
	}

	response, problemDetails := s.MbsN2MessageTransfer(transferData, binaryParts)
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

func (s *Server) handleMbsBroadcastContexts(w http.ResponseWriter, r *http.Request) {
	logger.SbiLog.Infof("Handle MBS Broadcast Contexts: %s %s", r.Method, r.URL.Path)

	switch r.Method {
	case http.MethodPost:
		s.handleCreateMbsBroadcastContext(w, r)
	default:
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Method Not Allowed",
			Status: http.StatusMethodNotAllowed,
			Detail: fmt.Sprintf("Method %s not allowed on this resource", r.Method),
		})
	}
}

func (s *Server) handleMbsBroadcastContext(w http.ResponseWriter, r *http.Request) {
	logger.SbiLog.Infof("Handle MBS Broadcast Context: %s %s", r.Method, r.URL.Path)

	path := r.URL.Path
	prefix := "/namf-mbs-bc/v1/mbs-contexts/"
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
			Detail: "MBS Context Reference is required",
		})
		return
	}

	parts := strings.Split(pathAfterPrefix, "/")
	mbsContextRef := parts[0]

	if len(parts) > 1 && parts[1] == "update" {
		if r.Method == http.MethodPost {
			s.handleUpdateMbsBroadcastContext(w, r, mbsContextRef)
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
	case http.MethodDelete:
		s.handleDeleteMbsBroadcastContext(w, mbsContextRef)
	default:
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Method Not Allowed",
			Status: http.StatusMethodNotAllowed,
			Detail: fmt.Sprintf("Method %s not allowed on this resource", r.Method),
		})
	}
}

func (s *Server) handleCreateMbsBroadcastContext(w http.ResponseWriter, r *http.Request) {
	logger.SbiLog.Info("Creating MBS Broadcast Context")

	createData, binaryParts, err := parseMbsBroadcastCreateRequest(r)
	if err != nil {
		logger.SbiLog.Errorf("Failed to parse MBS Broadcast create request: %v", err)
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: fmt.Sprintf("Failed to parse request: %v", err),
		})
		return
	}

	response, problemDetails := s.CreateMbsBroadcastContext(createData, binaryParts)
	if problemDetails != nil {
		sendProblemDetails(w, problemDetails)
		return
	}

	mbsContextRef := generateMbsContextRef()
	location := fmt.Sprintf("/namf-mbs-bc/v1/mbs-contexts/%s", mbsContextRef)
	w.Header().Set("Location", location)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.SbiLog.Errorf("Failed to encode response: %v", err)
	}
}

func (s *Server) handleDeleteMbsBroadcastContext(w http.ResponseWriter, mbsContextRef string) {
	logger.SbiLog.Infof("Deleting MBS Broadcast Context: %s", mbsContextRef)

	problemDetails := s.DeleteMbsBroadcastContext(mbsContextRef)
	if problemDetails != nil {
		sendProblemDetails(w, problemDetails)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleUpdateMbsBroadcastContext(w http.ResponseWriter, r *http.Request, mbsContextRef string) {
	logger.SbiLog.Infof("Updating MBS Broadcast Context: %s", mbsContextRef)

	updateData, binaryParts, err := parseMbsBroadcastUpdateRequest(r)
	if err != nil {
		logger.SbiLog.Errorf("Failed to parse MBS Broadcast update request: %v", err)
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: fmt.Sprintf("Failed to parse request: %v", err),
		})
		return
	}

	response, problemDetails := s.UpdateMbsBroadcastContext(mbsContextRef, updateData, binaryParts)
	if problemDetails != nil {
		sendProblemDetails(w, problemDetails)
		return
	}

	if response == nil {
		w.WriteHeader(http.StatusNoContent)
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

func (s *Server) handleAssignEbi(w http.ResponseWriter, r *http.Request, ueContextId string) {
	logger.SbiLog.Infof("Assign EBI for UE: %s", ueContextId)

	var assignData AssignEbiData
	if err := json.NewDecoder(r.Body).Decode(&assignData); err != nil {
		logger.SbiLog.Errorf("Failed to decode request body: %v", err)
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: fmt.Sprintf("Failed to decode request body: %v", err),
		})
		return
	}

	response, problemDetails := s.AssignEbi(ueContextId, &assignData)
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

func (s *Server) handleUEContextTransfer(w http.ResponseWriter, r *http.Request, ueContextId string) {
	logger.SbiLog.Infof("Handle UE Context Transfer for UE: %s", ueContextId)

	transferData, binaryParts, err := parseUEContextTransferRequest(r)
	if err != nil {
		logger.SbiLog.Errorf("Failed to parse UE context transfer request: %v", err)
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: fmt.Sprintf("Failed to parse request: %v", err),
		})
		return
	}

	response, problemDetails := s.UEContextTransfer(ueContextId, transferData, binaryParts)
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

func (s *Server) handleRegistrationStatusUpdate(w http.ResponseWriter, r *http.Request, ueContextId string) {
	logger.SbiLog.Infof("Handle Registration Status Update for UE: %s", ueContextId)

	var reqData UeRegStatusUpdateReqData
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

	response, problemDetails := s.RegistrationStatusUpdate(ueContextId, &reqData)
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

func (s *Server) handleRelocateUEContext(w http.ResponseWriter, r *http.Request, ueContextId string) {
	logger.SbiLog.Infof("Handle Relocate UE Context for UE: %s", ueContextId)

	relocateData, binaryParts, err := parseRelocateRequest(r)
	if err != nil {
		logger.SbiLog.Errorf("Failed to parse relocate request: %v", err)
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: fmt.Sprintf("Failed to parse request: %v", err),
		})
		return
	}

	response, problemDetails := s.RelocateUEContext(ueContextId, relocateData, binaryParts)
	if problemDetails != nil {
		sendProblemDetails(w, problemDetails)
		return
	}

	location := fmt.Sprintf("/namf-comm/v1/ue-contexts/%s/relocate", ueContextId)
	w.Header().Set("Location", location)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.SbiLog.Errorf("Failed to encode response: %v", err)
	}
}

func (s *Server) handleCancelRelocateUEContext(w http.ResponseWriter, r *http.Request, ueContextId string) {
	logger.SbiLog.Infof("Handle Cancel Relocate UE Context for UE: %s", ueContextId)

	cancelData, binaryParts, err := parseCancelRelocateRequest(r)
	if err != nil {
		logger.SbiLog.Errorf("Failed to parse cancel relocate request: %v", err)
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: fmt.Sprintf("Failed to parse request: %v", err),
		})
		return
	}

	problemDetails := s.CancelRelocateUEContext(ueContextId, cancelData, binaryParts)
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

func (s *Server) handleN1N2MessageSubscribe(w http.ResponseWriter, r *http.Request, ueContextId string) {
	logger.SbiLog.Infof("Handle N1N2 Message Subscribe for UE: %s", ueContextId)

	var reqData UeN1N2InfoSubscriptionCreateData
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

	response, problemDetails := s.N1N2MessageSubscribe(ueContextId, &reqData)
	if problemDetails != nil {
		sendProblemDetails(w, problemDetails)
		return
	}

	location := fmt.Sprintf("/namf-comm/v1/ue-contexts/%s/n1-n2-messages/subscriptions/%s", ueContextId, response.N1n2NotifySubscriptionId)
	w.Header().Set("Location", location)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.SbiLog.Errorf("Failed to encode response: %v", err)
	}
}

func (s *Server) handleN1N2MessageUnSubscribe(w http.ResponseWriter, ueContextId string, subscriptionId string) {
	logger.SbiLog.Infof("Handle N1N2 Message UnSubscribe for UE: %s, Subscription: %s", ueContextId, subscriptionId)

	problemDetails := s.N1N2MessageUnSubscribe(ueContextId, subscriptionId)
	if problemDetails != nil {
		sendProblemDetails(w, problemDetails)
		return
	}

	w.WriteHeader(http.StatusNoContent)
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

func parseN2TransferRequest(r *http.Request) (*N2InformationTransferReqData, map[string][]byte, error) {
	contentType := r.Header.Get("Content-Type")

	if strings.Contains(contentType, "application/json") {
		var reqData N2InformationTransferReqData
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
		var reqData *N2InformationTransferReqData
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
				var jsonData N2InformationTransferReqData
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

func parseMbsN2TransferRequest(r *http.Request) (*MbsN2MessageTransferReqData, map[string][]byte, error) {
	contentType := r.Header.Get("Content-Type")

	if strings.Contains(contentType, "application/json") {
		var reqData MbsN2MessageTransferReqData
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
		var reqData *MbsN2MessageTransferReqData
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
				var jsonData MbsN2MessageTransferReqData
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

func parseUEContextTransferRequest(r *http.Request) (*UeContextTransferReqData, map[string][]byte, error) {
	contentType := r.Header.Get("Content-Type")

	if strings.Contains(contentType, "application/json") {
		var reqData UeContextTransferReqData
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
		var reqData *UeContextTransferReqData
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
				var jsonData UeContextTransferReqData
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

func parseRelocateRequest(r *http.Request) (*UeContextRelocateData, map[string][]byte, error) {
	contentType := r.Header.Get("Content-Type")

	if strings.Contains(contentType, "application/json") {
		var reqData UeContextRelocateData
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
		var reqData *UeContextRelocateData
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
				var jsonData UeContextRelocateData
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

func parseCancelRelocateRequest(r *http.Request) (*UeContextCancelRelocateData, map[string][]byte, error) {
	contentType := r.Header.Get("Content-Type")

	if strings.Contains(contentType, "application/json") {
		var reqData UeContextCancelRelocateData
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
		var reqData *UeContextCancelRelocateData
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
				var jsonData UeContextCancelRelocateData
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
