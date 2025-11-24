package sbi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gavin/amf/internal/logger"
)

func (s *Server) handleAMFStatusSubscriptions(w http.ResponseWriter, r *http.Request) {
	logger.SbiLog.Infof("Handle AMF Status Subscriptions: %s %s", r.Method, r.URL.Path)

	switch r.Method {
	case http.MethodPost:
		s.handleCreateAMFStatusSubscription(w, r)
	default:
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Method Not Allowed",
			Status: http.StatusMethodNotAllowed,
			Detail: fmt.Sprintf("Method %s not allowed on this resource", r.Method),
		})
	}
}

func (s *Server) handleAMFStatusSubscription(w http.ResponseWriter, r *http.Request) {
	logger.SbiLog.Infof("Handle AMF Status Subscription: %s %s", r.Method, r.URL.Path)

	path := r.URL.Path
	prefix := "/namf-comm/v1/subscriptions/"
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
		s.handleDeleteAMFStatusSubscription(w, subscriptionId)
	case http.MethodPut:
		s.handleUpdateAMFStatusSubscription(w, r, subscriptionId)
	default:
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Method Not Allowed",
			Status: http.StatusMethodNotAllowed,
			Detail: fmt.Sprintf("Method %s not allowed on this resource", r.Method),
		})
	}
}

func (s *Server) handleCreateAMFStatusSubscription(w http.ResponseWriter, r *http.Request) {
	logger.SbiLog.Info("Creating AMF Status Change subscription")

	var createData SubscriptionData
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

	response, problemDetails := s.CreateAMFStatusSubscription(&createData)
	if problemDetails != nil {
		sendProblemDetails(w, problemDetails)
		return
	}

	location := fmt.Sprintf("/namf-comm/v1/subscriptions/%s", generateSubscriptionId())
	w.Header().Set("Location", location)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.SbiLog.Errorf("Failed to encode response: %v", err)
	}
}

func (s *Server) handleDeleteAMFStatusSubscription(w http.ResponseWriter, subscriptionId string) {
	logger.SbiLog.Infof("Deleting AMF Status Change subscription: %s", subscriptionId)

	problemDetails := s.DeleteAMFStatusSubscription(subscriptionId)
	if problemDetails != nil {
		sendProblemDetails(w, problemDetails)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleUpdateAMFStatusSubscription(w http.ResponseWriter, r *http.Request, subscriptionId string) {
	logger.SbiLog.Infof("Updating AMF Status Change subscription: %s", subscriptionId)

	var updateData SubscriptionData
	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		logger.SbiLog.Errorf("Failed to decode request body: %v", err)
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: fmt.Sprintf("Failed to decode request body: %v", err),
		})
		return
	}

	response, problemDetails := s.UpdateAMFStatusSubscription(subscriptionId, &updateData)
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

func (s *Server) CreateAMFStatusSubscription(createData *SubscriptionData) (*SubscriptionData, *ProblemDetails) {
	logger.SbiLog.Info("Creating AMF status change subscription")

	if createData.AmfStatusUri == "" {
		logger.SbiLog.Error("AMF status URI is required")
		return nil, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "AMF status URI is required",
		}
	}

	subscriptionId := generateSubscriptionId()
	s.amfContext.StoreAMFStatusSubscription(subscriptionId, createData)

	logger.SbiLog.Infof("AMF status change subscription created successfully: %s", subscriptionId)
	return createData, nil
}

func (s *Server) DeleteAMFStatusSubscription(subscriptionId string) *ProblemDetails {
	logger.SbiLog.Infof("Deleting AMF status change subscription: %s", subscriptionId)

	if subscriptionId == "" {
		logger.SbiLog.Error("Subscription ID cannot be empty")
		return &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "Subscription ID cannot be empty",
		}
	}

	_, exists := s.amfContext.GetAMFStatusSubscription(subscriptionId)
	if !exists {
		logger.SbiLog.Warnf("AMF status change subscription not found: %s", subscriptionId)
		return &ProblemDetails{
			Type:   "about:blank",
			Title:  "Not Found",
			Status: http.StatusNotFound,
			Detail: fmt.Sprintf("AMF status change subscription not found: %s", subscriptionId),
		}
	}

	s.amfContext.DeleteAMFStatusSubscription(subscriptionId)

	logger.SbiLog.Infof("AMF status change subscription deleted successfully: %s", subscriptionId)
	return nil
}

func (s *Server) UpdateAMFStatusSubscription(subscriptionId string, updateData *SubscriptionData) (*SubscriptionData, *ProblemDetails) {
	logger.SbiLog.Infof("Updating AMF status change subscription: %s", subscriptionId)

	if subscriptionId == "" {
		logger.SbiLog.Error("Subscription ID cannot be empty")
		return nil, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "Subscription ID cannot be empty",
		}
	}

	if updateData.AmfStatusUri == "" {
		logger.SbiLog.Error("AMF status URI is required")
		return nil, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "AMF status URI is required",
		}
	}

	_, exists := s.amfContext.GetAMFStatusSubscription(subscriptionId)
	if !exists {
		logger.SbiLog.Warnf("AMF status change subscription not found: %s", subscriptionId)
		return nil, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Not Found",
			Status: http.StatusNotFound,
			Detail: fmt.Sprintf("AMF status change subscription not found: %s", subscriptionId),
		}
	}

	s.amfContext.StoreAMFStatusSubscription(subscriptionId, updateData)

	logger.SbiLog.Infof("AMF status change subscription updated successfully: %s", subscriptionId)
	return updateData, nil
}

func (s *Server) NotifyAMFStatusChange(statusChange string, targetAmfRemoval string, targetAmfFailure string) {
	logger.SbiLog.Infof("Notifying AMF status change: %s", statusChange)

	guamiList := []Guami{}
	for _, guami := range s.amfContext.ServedGuami {
		guamiList = append(guamiList, Guami{
			PlmnId: &PlmnId{
				Mcc: guami.PlmnId.Mcc,
				Mnc: guami.PlmnId.Mnc,
			},
			AmfId: guami.AmfId,
		})
	}

	notification := &AmfStatusChangeNotification{
		AmfStatusInfoList: []AmfStatusInfo{
			{
				GuamiList:        guamiList,
				StatusChange:     statusChange,
				TargetAmfRemoval: targetAmfRemoval,
				TargetAmfFailure: targetAmfFailure,
			},
		},
	}

	subscriptions := s.amfContext.GetAllAMFStatusSubscriptions()

	for subscriptionId, subscription := range subscriptions {
		subData, ok := subscription.(*SubscriptionData)
		if !ok {
			logger.SbiLog.Warnf("Invalid subscription data format for ID: %s", subscriptionId)
			continue
		}

		go s.sendAMFStatusNotification(subData.AmfStatusUri, notification)
	}

	logger.SbiLog.Infof("Sent AMF status change notifications to %d subscribers", len(subscriptions))
}

func (s *Server) sendAMFStatusNotification(callbackUri string, notification *AmfStatusChangeNotification) {
	logger.SbiLog.Infof("Sending AMF status notification to: %s", callbackUri)

	jsonData, err := json.Marshal(notification)
	if err != nil {
		logger.SbiLog.Errorf("Failed to marshal notification: %v", err)
		return
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest(http.MethodPost, callbackUri, bytes.NewBuffer(jsonData))
	if err != nil {
		logger.SbiLog.Errorf("Failed to create HTTP request: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		logger.SbiLog.Errorf("Failed to send notification to %s: %v", callbackUri, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		logger.SbiLog.Infof("Successfully sent AMF status notification to: %s (status: %d)", callbackUri, resp.StatusCode)
	} else {
		logger.SbiLog.Warnf("Failed to send notification to %s, status code: %d", callbackUri, resp.StatusCode)
	}
}
