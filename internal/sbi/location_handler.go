package sbi

import (
	"fmt"
	"net/http"

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

	logger.SbiLog.Infof("Location request canceled for UE: %s, LDR Reference: %s", ueContextId, requestData.LdrReference)
	return nil
}
