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
