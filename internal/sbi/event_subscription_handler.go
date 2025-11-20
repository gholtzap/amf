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

func (s *Server) ModifyEventSubscription(subscriptionId string, patchItems []AmfUpdateEventSubscriptionItem) (*AmfUpdatedEventSubscription, *ProblemDetails) {
	logger.SbiLog.Infof("Modifying event subscription: %s with %d patch operations", subscriptionId, len(patchItems))

	if subscriptionId == "" {
		logger.SbiLog.Error("Subscription ID cannot be empty")
		return nil, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "Subscription ID cannot be empty",
		}
	}

	subscriptionData, exists := s.amfContext.GetEventSubscription(subscriptionId)
	if !exists {
		logger.SbiLog.Warnf("Event subscription not found: %s", subscriptionId)
		return nil, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Not Found",
			Status: http.StatusNotFound,
			Detail: fmt.Sprintf("Event subscription not found: %s", subscriptionId),
		}
	}

	subscription, ok := subscriptionData.(*AmfEventSubscription)
	if !ok {
		logger.SbiLog.Error("Failed to cast subscription to AmfEventSubscription")
		return nil, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Internal Server Error",
			Status: http.StatusInternalServerError,
			Detail: "Failed to retrieve subscription data",
		}
	}

	if len(patchItems) == 0 {
		logger.SbiLog.Error("Patch items cannot be empty")
		return nil, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "Patch items cannot be empty",
		}
	}

	for _, patch := range patchItems {
		switch patch.Op {
		case "replace":
			if err := s.applyReplacePatch(subscription, patch.Path, patch.Value); err != nil {
				logger.SbiLog.Errorf("Failed to apply replace patch: %v", err)
				return nil, &ProblemDetails{
					Type:   "about:blank",
					Title:  "Bad Request",
					Status: http.StatusBadRequest,
					Detail: fmt.Sprintf("Failed to apply patch: %v", err),
				}
			}
		case "add":
			if err := s.applyAddPatch(subscription, patch.Path, patch.Value); err != nil {
				logger.SbiLog.Errorf("Failed to apply add patch: %v", err)
				return nil, &ProblemDetails{
					Type:   "about:blank",
					Title:  "Bad Request",
					Status: http.StatusBadRequest,
					Detail: fmt.Sprintf("Failed to apply patch: %v", err),
				}
			}
		case "remove":
			if err := s.applyRemovePatch(subscription, patch.Path); err != nil {
				logger.SbiLog.Errorf("Failed to apply remove patch: %v", err)
				return nil, &ProblemDetails{
					Type:   "about:blank",
					Title:  "Bad Request",
					Status: http.StatusBadRequest,
					Detail: fmt.Sprintf("Failed to apply patch: %v", err),
				}
			}
		default:
			logger.SbiLog.Errorf("Unsupported patch operation: %s", patch.Op)
			return nil, &ProblemDetails{
				Type:   "about:blank",
				Title:  "Bad Request",
				Status: http.StatusBadRequest,
				Detail: fmt.Sprintf("Unsupported patch operation: %s", patch.Op),
			}
		}
	}

	s.amfContext.StoreEventSubscription(subscriptionId, subscription)

	response := &AmfUpdatedEventSubscription{
		Subscription: subscription,
		ReportList:   []AmfEventReport{},
	}

	logger.SbiLog.Infof("Event subscription modified successfully: %s", subscriptionId)
	return response, nil
}

func (s *Server) applyReplacePatch(subscription *AmfEventSubscription, path string, value interface{}) error {
	switch path {
	case "/eventNotifyUri":
		if strVal, ok := value.(string); ok {
			subscription.EventNotifyUri = strVal
		} else {
			return fmt.Errorf("invalid value type for eventNotifyUri")
		}
	case "/subsChangeNotifyUri":
		if strVal, ok := value.(string); ok {
			subscription.SubsChangeNotifyUri = strVal
		} else {
			return fmt.Errorf("invalid value type for subsChangeNotifyUri")
		}
	case "/subsChangeNotifyCorrelationId":
		if strVal, ok := value.(string); ok {
			subscription.SubsChangeNotifyCorrelationId = strVal
		} else {
			return fmt.Errorf("invalid value type for subsChangeNotifyCorrelationId")
		}
	default:
		return fmt.Errorf("unsupported path for replace operation: %s", path)
	}
	return nil
}

func (s *Server) applyAddPatch(subscription *AmfEventSubscription, path string, value interface{}) error {
	if path == "/eventList/-" || path == "/eventList" {
		eventMap, ok := value.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid value type for eventList")
		}

		eventType, _ := eventMap["type"].(string)
		immediateFlag, _ := eventMap["immediateFlag"].(bool)

		newEvent := AmfEvent{
			Type:          eventType,
			ImmediateFlag: immediateFlag,
		}
		subscription.EventList = append(subscription.EventList, newEvent)
		return nil
	}
	return fmt.Errorf("unsupported path for add operation: %s", path)
}

func (s *Server) applyRemovePatch(subscription *AmfEventSubscription, path string) error {
	return fmt.Errorf("remove operation not yet fully implemented for path: %s", path)
}

func generateSubscriptionId() string {
	return generateTransferId()
}
