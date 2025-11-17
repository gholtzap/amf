package sbi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

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
	s.router.HandleFunc("/namf-comm/v1/n1-n2-messages", s.handleN1N2MessageTransfer)

	s.router.HandleFunc("/namf-evts/v1/subscriptions", s.handleEventSubscriptions)
	s.router.HandleFunc("/namf-evts/v1/subscriptions/", s.handleEventSubscription)

	s.router.HandleFunc("/namf-loc/v1/provide-location-info", s.handleProvideLocationInfo)

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

func (s *Server) handleN1N2MessageTransfer(w http.ResponseWriter, r *http.Request) {
	logger.SbiLog.Infof("Handle N1N2 Message Transfer: %s %s", r.Method, r.URL.Path)

	w.WriteHeader(http.StatusNotImplemented)
}

func (s *Server) handleEventSubscriptions(w http.ResponseWriter, r *http.Request) {
	logger.SbiLog.Infof("Handle Event Subscriptions: %s %s", r.Method, r.URL.Path)

	w.WriteHeader(http.StatusNotImplemented)
}

func (s *Server) handleEventSubscription(w http.ResponseWriter, r *http.Request) {
	logger.SbiLog.Infof("Handle Event Subscription: %s %s", r.Method, r.URL.Path)

	w.WriteHeader(http.StatusNotImplemented)
}

func (s *Server) handleProvideLocationInfo(w http.ResponseWriter, r *http.Request) {
	logger.SbiLog.Infof("Handle Provide Location Info: %s %s", r.Method, r.URL.Path)

	w.WriteHeader(http.StatusNotImplemented)
}

func (s *Server) handleMTService(w http.ResponseWriter, r *http.Request) {
	logger.SbiLog.Infof("Handle MT Service: %s %s", r.Method, r.URL.Path)

	w.WriteHeader(http.StatusNotImplemented)
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
