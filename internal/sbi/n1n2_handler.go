package sbi

import (
	"fmt"
	"net/http"

	"github.com/gavin/amf/internal/context"
	"github.com/gavin/amf/internal/logger"
)

func (s *Server) N1N2MessageTransfer(ueContextId string, reqData *N1N2MessageTransferReqData, binaryParts map[string][]byte) (*N1N2MessageTransferRspData, *ProblemDetails) {
	logger.SbiLog.Infof("Processing N1N2 Message Transfer for UE: %s", ueContextId)

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

	if ue.CmState != context.CmConnected {
		logger.SbiLog.Warnf("UE %s not in CM-CONNECTED state", ueContextId)
		return &N1N2MessageTransferRspData{
			Cause: "ATTEMPTING_TO_REACH_UE",
		}, nil
	}

	if reqData.N1MessageContainer != nil && reqData.N1MessageContainer.N1MessageContent != nil {
		logger.SbiLog.Infof("Processing N1 message for UE %s", ueContextId)
		contentId := reqData.N1MessageContainer.N1MessageContent.ContentId
		if binaryData, ok := binaryParts[contentId]; ok {
			logger.SbiLog.Infof("Found N1 message binary data, size: %d bytes", len(binaryData))
		}
	}

	if reqData.N2InfoContainer != nil {
		logger.SbiLog.Infof("Processing N2 information for UE %s", ueContextId)

		if reqData.N2InfoContainer.SmInfo != nil {
			logger.SbiLog.Infof("N2 SM Information for PDU Session ID: %d", reqData.N2InfoContainer.SmInfo.PduSessionId)

			if reqData.N2InfoContainer.SmInfo.N2InfoContent != nil {
				contentId := reqData.N2InfoContainer.SmInfo.N2InfoContent.ContentId
				if binaryData, ok := binaryParts[contentId]; ok {
					logger.SbiLog.Infof("Found N2 SM info binary data, size: %d bytes", len(binaryData))
				}
			}
		}

		if reqData.N2InfoContainer.RanInfo != nil && reqData.N2InfoContainer.RanInfo.N2InfoContent != nil {
			logger.SbiLog.Infof("Processing N2 RAN information")
			contentId := reqData.N2InfoContainer.RanInfo.N2InfoContent.ContentId
			if binaryData, ok := binaryParts[contentId]; ok {
				logger.SbiLog.Infof("Found N2 RAN info binary data, size: %d bytes", len(binaryData))
			}
		}
	}

	if reqData.PduSessionId != 0 {
		logger.SbiLog.Infof("PDU Session ID specified: %d", reqData.PduSessionId)
		if _, exists := ue.PduSessions[reqData.PduSessionId]; !exists {
			logger.SbiLog.Warnf("PDU Session %d not found for UE %s", reqData.PduSessionId, ueContextId)
			return nil, &ProblemDetails{
				Type:   "about:blank",
				Title:  "Not Found",
				Status: http.StatusNotFound,
				Detail: fmt.Sprintf("PDU Session %d not found", reqData.PduSessionId),
			}
		}
	}

	logger.SbiLog.Infof("N1N2 Message Transfer initiated successfully for UE: %s", ueContextId)

	return &N1N2MessageTransferRspData{
		Cause: "N1_N2_TRANSFER_INITIATED",
	}, nil
}

func (s *Server) N1N2MessageSubscribe(ueContextId string, reqData *UeN1N2InfoSubscriptionCreateData) (*UeN1N2InfoSubscriptionCreatedData, *ProblemDetails) {
	logger.SbiLog.Infof("Creating N1N2 Message Subscription for UE: %s", ueContextId)

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

	if reqData.N1MessageClass == "" && reqData.N2InformationClass == "" {
		return nil, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "Either N1 Message Class or N2 Information Class must be specified",
		}
	}

	if reqData.N1MessageClass != "" && reqData.N1NotifyCallbackUri == "" {
		return nil, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "N1 Notify Callback URI is required when N1 Message Class is specified",
		}
	}

	if reqData.N2InformationClass != "" && reqData.N2NotifyCallbackUri == "" {
		return nil, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "N2 Notify Callback URI is required when N2 Information Class is specified",
		}
	}

	subscriptionId := generateSubscriptionId()

	subscription := &context.N1N2Subscription{
		SubscriptionId:      subscriptionId,
		UeContextId:         ueContextId,
		N1MessageClass:      reqData.N1MessageClass,
		N1NotifyCallbackUri: reqData.N1NotifyCallbackUri,
		N2InformationClass:  reqData.N2InformationClass,
		N2NotifyCallbackUri: reqData.N2NotifyCallbackUri,
		NfId:                reqData.NfId,
	}

	s.amfContext.AddN1N2Subscription(subscription)

	logger.SbiLog.Infof("N1N2 Subscription created with ID: %s for UE: %s", subscriptionId, ueContextId)

	return &UeN1N2InfoSubscriptionCreatedData{
		N1n2NotifySubscriptionId: subscriptionId,
		SupportedFeatures:        reqData.SupportedFeatures,
	}, nil
}
