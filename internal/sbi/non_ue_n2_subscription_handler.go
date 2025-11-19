package sbi

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gavin/amf/internal/logger"
)

func (s *Server) handleNonUeN2MessageSubscriptions(w http.ResponseWriter, r *http.Request) {
	logger.SbiLog.Infof("Handle Non-UE N2 Message Subscriptions: %s %s", r.Method, r.URL.Path)

	switch r.Method {
	case http.MethodPost:
		s.handleCreateNonUeN2MessageSubscription(w, r)
	default:
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Method Not Allowed",
			Status: http.StatusMethodNotAllowed,
			Detail: fmt.Sprintf("Method %s not allowed on this resource", r.Method),
		})
	}
}

func (s *Server) handleNonUeN2MessageSubscription(w http.ResponseWriter, r *http.Request) {
	logger.SbiLog.Infof("Handle Non-UE N2 Message Subscription: %s %s", r.Method, r.URL.Path)

	path := r.URL.Path
	prefix := "/namf-comm/v1/non-ue-n2-messages/subscriptions/"
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
		s.handleDeleteNonUeN2MessageSubscription(w, subscriptionId)
	default:
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Method Not Allowed",
			Status: http.StatusMethodNotAllowed,
			Detail: fmt.Sprintf("Method %s not allowed on this resource", r.Method),
		})
	}
}

func (s *Server) handleCreateNonUeN2MessageSubscription(w http.ResponseWriter, r *http.Request) {
	logger.SbiLog.Info("Creating Non-UE N2 Message Subscription")

	var createData NonUeN2InfoSubscriptionCreateData
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

	response, problemDetails := s.CreateNonUeN2MessageSubscription(&createData)
	if problemDetails != nil {
		sendProblemDetails(w, problemDetails)
		return
	}

	location := fmt.Sprintf("/namf-comm/v1/non-ue-n2-messages/subscriptions/%s", response.N2NotifySubscriptionId)
	w.Header().Set("Location", location)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.SbiLog.Errorf("Failed to encode response: %v", err)
	}
}

func (s *Server) handleDeleteNonUeN2MessageSubscription(w http.ResponseWriter, subscriptionId string) {
	logger.SbiLog.Infof("Deleting Non-UE N2 Message Subscription: %s", subscriptionId)

	problemDetails := s.DeleteNonUeN2MessageSubscription(subscriptionId)
	if problemDetails != nil {
		sendProblemDetails(w, problemDetails)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) CreateNonUeN2MessageSubscription(createData *NonUeN2InfoSubscriptionCreateData) (*NonUeN2InfoSubscriptionCreatedData, *ProblemDetails) {
	if createData.N2InformationClass == "" {
		return nil, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "n2InformationClass is required",
		}
	}

	if createData.N2NotifyCallbackUri == "" {
		return nil, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "n2NotifyCallbackUri is required",
		}
	}

	validClasses := map[string]bool{
		"SM":       true,
		"NRPPa":    true,
		"PWS":      true,
		"PWS-BCAL": true,
		"PWS-RF":   true,
		"RAN":      true,
	}

	if !validClasses[createData.N2InformationClass] {
		return nil, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: fmt.Sprintf("Invalid n2InformationClass: %s", createData.N2InformationClass),
		}
	}

	subscriptionId := generateN2SubscriptionId()

	s.amfContext.NonUeN2InfoSubscriptions.Store(subscriptionId, createData)

	logger.SbiLog.Infof("Created Non-UE N2 Info subscription %s for class %s", subscriptionId, createData.N2InformationClass)

	return &NonUeN2InfoSubscriptionCreatedData{
		N2NotifySubscriptionId: subscriptionId,
		SupportedFeatures:      createData.SupportedFeatures,
		N2InformationClass:     createData.N2InformationClass,
	}, nil
}

func (s *Server) DeleteNonUeN2MessageSubscription(subscriptionId string) *ProblemDetails {
	if _, ok := s.amfContext.NonUeN2InfoSubscriptions.Load(subscriptionId); !ok {
		return &ProblemDetails{
			Type:   "about:blank",
			Title:  "Not Found",
			Status: http.StatusNotFound,
			Detail: fmt.Sprintf("Subscription %s not found", subscriptionId),
		}
	}

	s.amfContext.NonUeN2InfoSubscriptions.Delete(subscriptionId)

	logger.SbiLog.Infof("Deleted Non-UE N2 Info subscription %s", subscriptionId)

	return nil
}

func generateN2SubscriptionId() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "n2-sub-" + fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}
