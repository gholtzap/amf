package sbi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gavin/amf/internal/logger"
)

const (
	EventTypeLocationReport        = "LOCATION_REPORT"
	EventTypePresenceInAoi         = "PRESENCE_IN_AOI_REPORT"
	EventTypeTimezoneReport        = "TIMEZONE_REPORT"
	EventTypeAccessTypeReport      = "ACCESS_TYPE_REPORT"
	EventTypeRegistrationState     = "REGISTRATION_STATE_REPORT"
	EventTypeConnectivityState     = "CONNECTIVITY_STATE_REPORT"
	EventTypeReachability          = "REACHABILITY_REPORT"
	EventTypeCommunicationFailure  = "COMMUNICATION_FAILURE_REPORT"
	EventTypeUeMobility            = "UE_MOBILITY_REPORT"
	EventTypeLossOfConnectivity    = "LOSS_OF_CONNECTIVITY"
)

const (
	RmStateRegistered    = "REGISTERED"
	RmStateDeregistered  = "DEREGISTERED"
	CmStateConnected     = "CONNECTED"
	CmStateIdle          = "IDLE"
	ReachabilityReachable   = "REACHABLE"
	ReachabilityUnreachable = "UNREACHABLE"
)

type EventNotifier struct {
	httpClient *http.Client
}

func NewEventNotifier() *EventNotifier {
	return &EventNotifier{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type AmfEventNotification struct {
	NotifyCorrelationId           string           `json:"notifyCorrelationId"`
	SubsChangeNotifyCorrelationId string           `json:"subsChangeNotifyCorrelationId,omitempty"`
	ReportList                    []AmfEventReport `json:"reportList"`
}

func (n *EventNotifier) SendEventNotification(notifyUri string, notification *AmfEventNotification) error {
	logger.SbiLog.Infof("Sending event notification to %s", notifyUri)

	jsonData, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %v", err)
	}

	req, err := http.NewRequest("POST", notifyUri, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		logger.SbiLog.Errorf("Failed to send event notification: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		logger.SbiLog.Warnf("Event notification returned non-success status: %d", resp.StatusCode)
		return fmt.Errorf("notification failed with status: %d", resp.StatusCode)
	}

	logger.SbiLog.Infof("Event notification sent successfully to %s", notifyUri)
	return nil
}

func (s *Server) NotifyEvent(eventType string, supi string, additionalData map[string]interface{}) {
	NotifyEventForContext(s.amfContext, eventType, supi, additionalData)
}

func NotifyEventForContext(amfContext interface{}, eventType string, supi string, additionalData map[string]interface{}) {
	type EventSubscriptionGetter interface {
		GetAllEventSubscriptions() map[string]interface{}
	}

	getter, ok := amfContext.(EventSubscriptionGetter)
	if !ok {
		return
	}

	subscriptions := getter.GetAllEventSubscriptions()

	timestamp := time.Now().UTC().Format(time.RFC3339)

	for subscriptionId, subscriptionData := range subscriptions {
		subscription, ok := subscriptionData.(*AmfEventSubscription)
		if !ok {
			continue
		}

		shouldNotify := false
		for _, event := range subscription.EventList {
			if event.Type == eventType {
				shouldNotify = true
				break
			}
		}

		if !shouldNotify {
			continue
		}

		if subscription.Supi != "" && subscription.Supi != supi {
			continue
		}

		report := AmfEventReport{
			Type:      eventType,
			TimeStamp: timestamp,
			Supi:      supi,
		}

		if additionalData != nil {
			if rmState, ok := additionalData["rmState"].(string); ok {
				report.RmInfoList = []RmInfo{{RmState: rmState, AccessType: "3GPP_ACCESS"}}
			}
			if cmState, ok := additionalData["cmState"].(string); ok {
				report.CmInfoList = []CmInfo{{CmState: cmState, AccessType: "3GPP_ACCESS"}}
			}
			if reachability, ok := additionalData["reachability"].(string); ok {
				report.Reachability = reachability
			}
			if pei, ok := additionalData["pei"].(string); ok {
				report.Pei = pei
			}
			if timezone, ok := additionalData["timezone"].(string); ok {
				report.Timezone = timezone
			}
			if location, ok := additionalData["location"].(*UserLocation); ok {
				report.Location = location
			}
		}

		notification := &AmfEventNotification{
			NotifyCorrelationId: subscription.NotifyCorrelationId,
			ReportList:          []AmfEventReport{report},
		}

		notifier := NewEventNotifier()
		go func(uri string, notif *AmfEventNotification, subId string) {
			if err := notifier.SendEventNotification(uri, notif); err != nil {
				logger.SbiLog.Errorf("Failed to send notification for subscription %s: %v", subId, err)
			}
		}(subscription.EventNotifyUri, notification, subscriptionId)
	}
}
