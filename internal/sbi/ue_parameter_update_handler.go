package sbi

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gavin/amf/internal/logger"
)

type UEParameterUpdateRequest struct {
	Supi                   string `json:"supi"`
	RoutingIndicator       string `json:"routingIndicator,omitempty"`
	DisasterRoamingEnabled bool   `json:"disasterRoamingEnabled,omitempty"`
	DefaultConfiguredNSSAI string `json:"defaultConfiguredNSSAI,omitempty"`
	UpdateData             string `json:"updateData,omitempty"`
}

type UEParameterUpdateResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

func (s *Server) handleUEParameterUpdate(w http.ResponseWriter, r *http.Request) {
	logger.SbiLog.Infof("Handle UE Parameter Update: %s %s", r.Method, r.URL.Path)

	switch r.Method {
	case http.MethodPost:
		s.handleSendUEParameterUpdate(w, r)
	default:
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Method Not Allowed",
			Status: http.StatusMethodNotAllowed,
			Detail: fmt.Sprintf("Method %s not allowed on this resource", r.Method),
		})
	}
}

func (s *Server) handleSendUEParameterUpdate(w http.ResponseWriter, r *http.Request) {
	logger.SbiLog.Info("Processing UE Parameter Update Request")

	var request UEParameterUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		logger.SbiLog.Errorf("Failed to decode request body: %v", err)
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: fmt.Sprintf("Failed to decode request body: %v", err),
		})
		return
	}

	if request.Supi == "" {
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "SUPI is required",
		})
		return
	}

	ue := s.amfContext.GetUEBySupi(request.Supi)
	if ue == nil {
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Not Found",
			Status: http.StatusNotFound,
			Detail: fmt.Sprintf("UE not found for SUPI: %s", request.Supi),
		})
		return
	}

	ue.RoutingIndicator = request.RoutingIndicator
	ue.DisasterRoamingEnabled = request.DisasterRoamingEnabled

	if request.DefaultConfiguredNSSAI != "" {
		nssaiBytes, err := hex.DecodeString(request.DefaultConfiguredNSSAI)
		if err != nil {
			logger.SbiLog.Warnf("Failed to decode DefaultConfiguredNSSAI: %v", err)
		} else {
			ue.DefaultConfiguredNSSAI = nssaiBytes
		}
	}

	var updateData []byte
	if request.UpdateData != "" {
		var err error
		updateData, err = hex.DecodeString(request.UpdateData)
		if err != nil {
			sendProblemDetails(w, &ProblemDetails{
				Type:   "about:blank",
				Title:  "Bad Request",
				Status: http.StatusBadRequest,
				Detail: fmt.Sprintf("Invalid updateData hex string: %v", err),
			})
			return
		}
	} else {
		updateData = []byte{0x01, 0x00}
	}

	if s.nasHandler == nil {
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Internal Server Error",
			Status: http.StatusInternalServerError,
			Detail: "NAS handler not available",
		})
		return
	}

	if err := s.nasHandler.SendUEParameterUpdate(ue, updateData); err != nil {
		logger.SbiLog.Errorf("Failed to send UE parameter update: %v", err)
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Internal Server Error",
			Status: http.StatusInternalServerError,
			Detail: fmt.Sprintf("Failed to send UE parameter update: %v", err),
		})
		return
	}

	response := UEParameterUpdateResponse{
		Success: true,
		Message: fmt.Sprintf("UE parameter update sent to UE with SUPI: %s", request.Supi),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.SbiLog.Errorf("Failed to encode response: %v", err)
	}
}
