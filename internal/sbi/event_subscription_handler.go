package sbi

import (
	"fmt"
	"net/http"

	"github.com/gavin/amf/internal/logger"
)

func (s *Server) CreateEventSubscription(createData *AmfCreateEventSubscription) (*AmfCreatedEventSubscription, *ProblemDetails) {
	logger.SbiLog.Info("Creating event subscription")

	if createData.Subscription == nil {
		logger.SbiLog.Error("Subscription data is required")
		return nil, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "Subscription data is required",
		}
	}

	subscription := createData.Subscription

	if len(subscription.EventList) == 0 {
		logger.SbiLog.Error("Event list cannot be empty")
		return nil, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "Event list cannot be empty",
		}
	}

	if subscription.EventNotifyUri == "" {
		logger.SbiLog.Error("Event notify URI is required")
		return nil, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "Event notify URI is required",
		}
	}

	if subscription.NotifyCorrelationId == "" {
		logger.SbiLog.Error("Notify correlation ID is required")
		return nil, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "Notify correlation ID is required",
		}
	}

	if subscription.NfId == "" {
		logger.SbiLog.Error("NF ID is required")
		return nil, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "NF ID is required",
		}
	}

	subscriptionId := generateSubscriptionId()

	s.amfContext.StoreEventSubscription(subscriptionId, subscription)

	response := &AmfCreatedEventSubscription{
		Subscription:      subscription,
		SubscriptionId:    subscriptionId,
		SupportedFeatures: createData.SupportedFeatures,
	}

	logger.SbiLog.Infof("Event subscription created successfully: %s", subscriptionId)
	return response, nil
}

func (s *Server) DeleteEventSubscription(subscriptionId string) *ProblemDetails {
	logger.SbiLog.Infof("Deleting event subscription: %s", subscriptionId)

	if subscriptionId == "" {
		logger.SbiLog.Error("Subscription ID cannot be empty")
		return &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "Subscription ID cannot be empty",
		}
	}

	_, exists := s.amfContext.GetEventSubscription(subscriptionId)
	if !exists {
		logger.SbiLog.Warnf("Event subscription not found: %s", subscriptionId)
		return &ProblemDetails{
			Type:   "about:blank",
			Title:  "Not Found",
			Status: http.StatusNotFound,
			Detail: fmt.Sprintf("Event subscription not found: %s", subscriptionId),
		}
	}

	s.amfContext.DeleteEventSubscription(subscriptionId)

	logger.SbiLog.Infof("Event subscription deleted successfully: %s", subscriptionId)
	return nil
}

func generateSubscriptionId() string {
	return generateTransferId()
}
