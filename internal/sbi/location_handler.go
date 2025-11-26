package sbi

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gavin/amf/internal/context"
	"github.com/gavin/amf/internal/logger"
)

func (s *Server) ProvideLocationInfo(ueContextId string, requestData *RequestLocInfo) (*ProvideLocInfo, *ProblemDetails) {
	logger.SbiLog.Infof("Providing location info for UE ID: %s", ueContextId)

	if err := validateUeContextId(ueContextId); err != nil {
		logger.SbiLog.Errorf("Invalid UE Context ID format: %v", err)
		return nil, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: fmt.Sprintf("Invalid UE Context ID format: %v", err),
		}
	}

	ue := s.findUEContextById(ueContextId)
	if ue == nil {
		logger.SbiLog.Warnf("UE Context not found for ID: %s", ueContextId)
		return nil, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Not Found",
			Status: http.StatusNotFound,
			Detail: fmt.Sprintf("UE Context not found for ID: %s", ueContextId),
		}
	}

	response := &ProvideLocInfo{}

	if requestData.ReqCurrentLoc {
		response.CurrentLoc = true
	}

	if requestData.Req5gsLoc || requestData.ReqCurrentLoc {
		if ue.Tai.PlmnId.Mcc != "" {
			location := &UserLocation{}

			location.NrLocation = &NrLocation{
				Tai: ToSbiTai(ue.Tai),
			}

			if ue.CellId != "" {
				location.NrLocation.Ncgi = &Ncgi{
					PlmnId: &PlmnId{
						Mcc: ue.Tai.PlmnId.Mcc,
						Mnc: ue.Tai.PlmnId.Mnc,
					},
					NrCellId: ue.CellId,
				}
			}

			response.Location = location
		}
	}

	if requestData.ReqRatType {
		switch ue.AccessType {
		case "3GPP_ACCESS":
			response.RatType = "NR"
		case "NON_3GPP_ACCESS":
			response.RatType = "EUTRA"
		default:
			response.RatType = "NR"
		}
	}

	if requestData.ReqTimeZone {
		response.Timezone = "+00:00"
	}

	response.SupportedFeatures = requestData.SupportedFeatures

	logger.SbiLog.Infof("Location info provided for UE: %s", ueContextId)
	return response, nil
}

func (s *Server) ProvidePositioningInfo(ueContextId string, requestData *RequestPosInfo) (*ProvidePosInfoExt, *ProblemDetails) {
	logger.SbiLog.Infof("Providing positioning info for UE ID: %s, LCS Client Type: %s", ueContextId, requestData.LcsClientType)

	if err := validateUeContextId(ueContextId); err != nil {
		logger.SbiLog.Errorf("Invalid UE Context ID format: %v", err)
		return nil, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: fmt.Sprintf("Invalid UE Context ID format: %v", err),
		}
	}

	ue := s.findUEContextById(ueContextId)
	if ue == nil {
		logger.SbiLog.Warnf("UE Context not found for ID: %s", ueContextId)
		return nil, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Not Found",
			Status: http.StatusNotFound,
			Detail: fmt.Sprintf("UE Context not found for ID: %s", ueContextId),
		}
	}

	if requestData.Supi != "" && ue.Supi != requestData.Supi {
		logger.SbiLog.Warnf("SUPI mismatch: UE context has %s, request has %s", ue.Supi, requestData.Supi)
		return nil, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "SUPI mismatch",
		}
	}

	response := &ProvidePosInfoExt{
		SupportedFeatures: requestData.SupportedFeatures,
	}

	if ue.Tai.PlmnId.Mcc != "" {
		lat := 37.7749
		lon := -122.4194

		response.LocationEstimate = &GeographicArea{
			Point: &Point{
				Lat: lat,
				Lon: lon,
			},
		}

		response.AccuracyFulfilmentInd = "REQUESTED_ACCURACY_FULFILLED"
		response.AgeOfLocationEstimate = 5
	}

	if requestData.VelocityRequested != "" {
		response.VelocityEstimate = &VelocityEstimate{
			HSpeed:       50,
			Bearing:      90,
			HUncertainty: 10,
		}
	}

	if ue.CellId != "" {
		response.Ncgi = &Ncgi{
			PlmnId: &PlmnId{
				Mcc: ue.Tai.PlmnId.Mcc,
				Mnc: ue.Tai.PlmnId.Mnc,
			},
			NrCellId: ue.CellId,
		}
	}

	if requestData.LcsServiceType != "" {
		logger.SbiLog.Infof("LCS Service Type: %s", requestData.LcsServiceType)
	}

	if requestData.Priority != "" {
		logger.SbiLog.Infof("Priority: %s", requestData.Priority)
	}

	if requestData.LocationNotificationUri != "" {
		logger.SbiLog.Infof("Location notification URI provided: %s", requestData.LocationNotificationUri)

		if requestData.PeriodicEventInfo != nil {
			logger.SbiLog.Infof("Periodic location reporting requested: interval=%ds, max_reports=%d",
				requestData.PeriodicEventInfo.ReportingInterval, requestData.PeriodicEventInfo.MaximumNumberOfReports)

			subscriptionId := generateLocationSubscriptionId()
			subscription := &context.LocationSubscription{
				SubscriptionId:          subscriptionId,
				UeContextId:             ueContextId,
				LocationNotificationUri: requestData.LocationNotificationUri,
				EventType:               "PERIODIC",
				ReportingInterval:       requestData.PeriodicEventInfo.ReportingInterval,
				MaximumNumberOfReports:  requestData.PeriodicEventInfo.MaximumNumberOfReports,
				ReportCount:             0,
				StopTimer:               make(chan struct{}),
			}

			s.amfContext.StoreLocationSubscription(subscriptionId, subscription)

			go s.periodicLocationReporter(subscription)

			response.LdrReference = subscriptionId
		} else if requestData.AreaEventInfo != nil {
			logger.SbiLog.Infof("Area-based location reporting requested: areas=%d, occurrence=%s",
				len(requestData.AreaEventInfo.AreaDefinition), requestData.AreaEventInfo.OccurrenceInfo)

			subscriptionId := generateLocationSubscriptionId()
			subscription := &context.LocationSubscription{
				SubscriptionId:          subscriptionId,
				UeContextId:             ueContextId,
				LocationNotificationUri: requestData.LocationNotificationUri,
				EventType:               "AREA_EVENT",
				ReportCount:             0,
				StopTimer:               make(chan struct{}),
				AreaEventInfo:           requestData.AreaEventInfo,
			}

			s.amfContext.StoreLocationSubscription(subscriptionId, subscription)

			go s.areaEventLocationReporter(subscription, requestData.AreaEventInfo)

			response.LdrReference = subscriptionId
		} else if requestData.MotionEventInfo != nil {
			logger.SbiLog.Infof("Motion-based location reporting requested: linearDistance=%dm, occurrence=%s",
				requestData.MotionEventInfo.LinearDistance, requestData.MotionEventInfo.OccurrenceInfo)

			subscriptionId := generateLocationSubscriptionId()
			subscription := &context.LocationSubscription{
				SubscriptionId:          subscriptionId,
				UeContextId:             ueContextId,
				LocationNotificationUri: requestData.LocationNotificationUri,
				EventType:               "MOTION_EVENT",
				ReportCount:             0,
				StopTimer:               make(chan struct{}),
				MotionEventInfo:         requestData.MotionEventInfo,
			}

			s.amfContext.StoreLocationSubscription(subscriptionId, subscription)

			go s.motionEventLocationReporter(subscription, requestData.MotionEventInfo)

			response.LdrReference = subscriptionId
		}
	}

	logger.SbiLog.Infof("Positioning info provided for UE: %s", ueContextId)
	return response, nil
}

func (s *Server) CancelLocation(ueContextId string, requestData *CancelPosInfo) *ProblemDetails {
	logger.SbiLog.Infof("Canceling location for UE ID: %s", ueContextId)

	if err := validateUeContextId(ueContextId); err != nil {
		logger.SbiLog.Errorf("Invalid UE Context ID format: %v", err)
		return &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: fmt.Sprintf("Invalid UE Context ID format: %v", err),
		}
	}

	if requestData.Supi == "" {
		logger.SbiLog.Error("SUPI is required")
		return &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "SUPI is required",
		}
	}

	if requestData.HgmlcCallBackURI == "" {
		logger.SbiLog.Error("H-GMLC Callback URI is required")
		return &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "H-GMLC Callback URI is required",
		}
	}

	if requestData.LdrReference == "" {
		logger.SbiLog.Error("LDR Reference is required")
		return &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "LDR Reference is required",
		}
	}

	ue := s.findUEContextById(ueContextId)
	if ue == nil {
		logger.SbiLog.Warnf("UE Context not found for ID: %s", ueContextId)
		return &ProblemDetails{
			Type:   "about:blank",
			Title:  "Not Found",
			Status: http.StatusNotFound,
			Detail: fmt.Sprintf("UE Context not found for ID: %s", ueContextId),
		}
	}

	if ue.Supi != requestData.Supi {
		logger.SbiLog.Warnf("SUPI mismatch: UE context has %s, request has %s", ue.Supi, requestData.Supi)
		return &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "SUPI mismatch",
		}
	}

	s.amfContext.DeleteLocationSubscription(requestData.LdrReference)

	logger.SbiLog.Infof("Location request canceled for UE: %s, LDR Reference: %s", ueContextId, requestData.LdrReference)
	return nil
}

func generateLocationSubscriptionId() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "loc-sub-" + fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

func (s *Server) periodicLocationReporter(subscription *context.LocationSubscription) {
	ticker := time.NewTicker(time.Duration(subscription.ReportingInterval) * time.Second)
	defer ticker.Stop()

	logger.SbiLog.Infof("Started periodic location reporting for subscription %s (interval: %ds, max reports: %d)",
		subscription.SubscriptionId, subscription.ReportingInterval, subscription.MaximumNumberOfReports)

	for {
		select {
		case <-ticker.C:
			ue := s.findUEContextById(subscription.UeContextId)
			if ue == nil {
				logger.SbiLog.Warnf("UE context not found for location subscription %s, stopping reporter", subscription.SubscriptionId)
				s.amfContext.DeleteLocationSubscription(subscription.SubscriptionId)
				return
			}

			notification := &NotifiedPosInfo{}

			if ue.Tai.PlmnId.Mcc != "" {
				lat := 37.7749
				lon := -122.4194

				notification.LocationEstimate = &GeographicArea{
					Point: &Point{
						Lat: lat,
						Lon: lon,
					},
				}

				notification.AccuracyFulfilmentInd = "REQUESTED_ACCURACY_FULFILLED"
				notification.AgeOfLocationEstimate = 5
			}

			if ue.CellId != "" {
				notification.Ncgi = &Ncgi{
					PlmnId: &PlmnId{
						Mcc: ue.Tai.PlmnId.Mcc,
						Mnc: ue.Tai.PlmnId.Mnc,
					},
					NrCellId: ue.CellId,
				}
			}

			notification.LdrReference = subscription.SubscriptionId

			go s.sendLocationNotification(subscription.LocationNotificationUri, notification)

			subscription.ReportCount++

			if subscription.MaximumNumberOfReports > 0 && subscription.ReportCount >= subscription.MaximumNumberOfReports {
				logger.SbiLog.Infof("Maximum reports (%d) reached for subscription %s, stopping reporter",
					subscription.MaximumNumberOfReports, subscription.SubscriptionId)
				s.amfContext.DeleteLocationSubscription(subscription.SubscriptionId)
				return
			}

		case <-subscription.StopTimer:
			logger.SbiLog.Infof("Location reporting stopped for subscription %s", subscription.SubscriptionId)
			return
		}
	}
}

func (s *Server) sendLocationNotification(uri string, notification *NotifiedPosInfo) {
	body, err := json.Marshal(notification)
	if err != nil {
		logger.SbiLog.Errorf("Failed to marshal location notification: %v", err)
		return
	}

	req, err := http.NewRequest(http.MethodPost, uri, bytes.NewBuffer(body))
	if err != nil {
		logger.SbiLog.Errorf("Failed to create location notification request: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.SbiLog.Errorf("Failed to send location notification to %s: %v", uri, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		logger.SbiLog.Infof("Location notification sent successfully to %s (status: %d)", uri, resp.StatusCode)
	} else {
		logger.SbiLog.Warnf("Location notification failed with status %d from %s", resp.StatusCode, uri)
	}
}

func (s *Server) areaEventLocationReporter(subscription *context.LocationSubscription, areaEventInfo *AreaEventInfo) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	logger.SbiLog.Infof("Started area-based location reporting for subscription %s", subscription.SubscriptionId)

	for {
		select {
		case <-ticker.C:
			ue := s.findUEContextById(subscription.UeContextId)
			if ue == nil {
				logger.SbiLog.Warnf("UE context not found for location subscription %s, stopping reporter", subscription.SubscriptionId)
				s.amfContext.DeleteLocationSubscription(subscription.SubscriptionId)
				return
			}

			lat := 37.7749
			lon := -122.4194
			currentLocation := &context.Point{Lat: lat, Lon: lon}

			wasInside := false
			if subscription.LastKnownLocation != nil {
				wasInside = isPointInAreas(subscription.LastKnownLocation, areaEventInfo.AreaDefinition)
			}
			isInside := isPointInAreas(currentLocation, areaEventInfo.AreaDefinition)

			shouldNotify := false
			if areaEventInfo.OccurrenceInfo == "ONE_TIME" {
				if !wasInside && isInside {
					shouldNotify = true
				}
			} else {
				if wasInside != isInside {
					shouldNotify = true
				}
			}

			if shouldNotify {
				notification := &NotifiedPosInfo{
					LocationEstimate: &GeographicArea{
						Point: &Point{
							Lat: lat,
							Lon: lon,
						},
					},
					AccuracyFulfilmentInd: "REQUESTED_ACCURACY_FULFILLED",
					AgeOfLocationEstimate: 5,
					LdrReference:          subscription.SubscriptionId,
				}

				if ue.CellId != "" {
					notification.Ncgi = &Ncgi{
						PlmnId: &PlmnId{
							Mcc: ue.Tai.PlmnId.Mcc,
							Mnc: ue.Tai.PlmnId.Mnc,
						},
						NrCellId: ue.CellId,
					}
				}

				go s.sendLocationNotification(subscription.LocationNotificationUri, notification)

				subscription.ReportCount++
				subscription.LastKnownLocation = currentLocation
				subscription.LastReportTime = time.Now().Unix()

				if areaEventInfo.MaximumNumberOfReports > 0 && subscription.ReportCount >= areaEventInfo.MaximumNumberOfReports {
					logger.SbiLog.Infof("Maximum reports (%d) reached for subscription %s, stopping reporter",
						areaEventInfo.MaximumNumberOfReports, subscription.SubscriptionId)
					s.amfContext.DeleteLocationSubscription(subscription.SubscriptionId)
					return
				}

				if areaEventInfo.OccurrenceInfo == "ONE_TIME" {
					logger.SbiLog.Infof("One-time area event triggered for subscription %s, stopping reporter", subscription.SubscriptionId)
					s.amfContext.DeleteLocationSubscription(subscription.SubscriptionId)
					return
				}
			} else {
				subscription.LastKnownLocation = currentLocation
			}

		case <-subscription.StopTimer:
			logger.SbiLog.Infof("Area-based location reporting stopped for subscription %s", subscription.SubscriptionId)
			return
		}
	}
}

func (s *Server) motionEventLocationReporter(subscription *context.LocationSubscription, motionEventInfo *MotionEventInfo) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	logger.SbiLog.Infof("Started motion-based location reporting for subscription %s (linear distance: %dm)",
		subscription.SubscriptionId, motionEventInfo.LinearDistance)

	for {
		select {
		case <-ticker.C:
			ue := s.findUEContextById(subscription.UeContextId)
			if ue == nil {
				logger.SbiLog.Warnf("UE context not found for location subscription %s, stopping reporter", subscription.SubscriptionId)
				s.amfContext.DeleteLocationSubscription(subscription.SubscriptionId)
				return
			}

			lat := 37.7749
			lon := -122.4194
			currentLocation := &context.Point{Lat: lat, Lon: lon}

			shouldNotify := false
			if subscription.LastKnownLocation != nil {
				distance := calculateDistance(subscription.LastKnownLocation, currentLocation)
				if distance >= float64(motionEventInfo.LinearDistance) {
					shouldNotify = true
					logger.SbiLog.Infof("Motion threshold exceeded: %.2fm (threshold: %dm)", distance, motionEventInfo.LinearDistance)
				}
			} else {
				subscription.LastKnownLocation = currentLocation
			}

			if shouldNotify {
				notification := &NotifiedPosInfo{
					LocationEstimate: &GeographicArea{
						Point: &Point{
							Lat: lat,
							Lon: lon,
						},
					},
					AccuracyFulfilmentInd: "REQUESTED_ACCURACY_FULFILLED",
					AgeOfLocationEstimate: 5,
					LdrReference:          subscription.SubscriptionId,
				}

				if ue.CellId != "" {
					notification.Ncgi = &Ncgi{
						PlmnId: &PlmnId{
							Mcc: ue.Tai.PlmnId.Mcc,
							Mnc: ue.Tai.PlmnId.Mnc,
						},
						NrCellId: ue.CellId,
					}
				}

				go s.sendLocationNotification(subscription.LocationNotificationUri, notification)

				subscription.ReportCount++
				subscription.LastKnownLocation = currentLocation
				subscription.LastReportTime = time.Now().Unix()

				if motionEventInfo.MaximumNumberOfReports > 0 && subscription.ReportCount >= motionEventInfo.MaximumNumberOfReports {
					logger.SbiLog.Infof("Maximum reports (%d) reached for subscription %s, stopping reporter",
						motionEventInfo.MaximumNumberOfReports, subscription.SubscriptionId)
					s.amfContext.DeleteLocationSubscription(subscription.SubscriptionId)
					return
				}

				if motionEventInfo.OccurrenceInfo == "ONE_TIME" {
					logger.SbiLog.Infof("One-time motion event triggered for subscription %s, stopping reporter", subscription.SubscriptionId)
					s.amfContext.DeleteLocationSubscription(subscription.SubscriptionId)
					return
				}
			}

		case <-subscription.StopTimer:
			logger.SbiLog.Infof("Motion-based location reporting stopped for subscription %s", subscription.SubscriptionId)
			return
		}
	}
}

func isPointInAreas(point *context.Point, areas []GeographicArea) bool {
	for _, area := range areas {
		if area.Polygon != nil && len(area.Polygon) > 0 {
			if isPointInPolygon(point, area.Polygon) {
				return true
			}
		}
		if area.Point != nil {
			if point.Lat == area.Point.Lat && point.Lon == area.Point.Lon {
				return true
			}
		}
	}
	return false
}

func isPointInPolygon(point *context.Point, polygon []Point) bool {
	if len(polygon) < 3 {
		return false
	}

	inside := false
	j := len(polygon) - 1

	for i := 0; i < len(polygon); i++ {
		xi, yi := polygon[i].Lon, polygon[i].Lat
		xj, yj := polygon[j].Lon, polygon[j].Lat

		intersect := ((yi > point.Lat) != (yj > point.Lat)) &&
			(point.Lon < (xj-xi)*(point.Lat-yi)/(yj-yi)+xi)

		if intersect {
			inside = !inside
		}
		j = i
	}

	return inside
}

func calculateDistance(p1, p2 *context.Point) float64 {
	const earthRadius = 6371000.0

	lat1 := p1.Lat * 3.141592653589793 / 180.0
	lat2 := p2.Lat * 3.141592653589793 / 180.0
	deltaLat := (p2.Lat - p1.Lat) * 3.141592653589793 / 180.0
	deltaLon := (p2.Lon - p1.Lon) * 3.141592653589793 / 180.0

	sinDeltaLat := deltaLat / 2.0
	sinDeltaLon := deltaLon / 2.0
	a := sinDeltaLat*sinDeltaLat + (1.0-lat1*lat1/2.0)*(1.0-lat2*lat2/2.0)*sinDeltaLon*sinDeltaLon

	if a < 0 {
		a = 0
	}
	if a > 1 {
		a = 1
	}

	c := 2.0 * earthRadius * (a / (1.0 - a))

	return c
}
