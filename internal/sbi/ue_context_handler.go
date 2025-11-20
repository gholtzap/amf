package sbi

import (
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/gavin/amf/internal/context"
	"github.com/gavin/amf/internal/logger"
)

func (s *Server) CreateUEContext(ueContextId string, createData *UeContextCreateData) (*UeContextCreatedData, *ProblemDetails) {
	logger.SbiLog.Infof("Creating UE Context for UE ID: %s", ueContextId)

	if err := validateUeContextId(ueContextId); err != nil {
		logger.SbiLog.Errorf("Invalid UE Context ID format: %v", err)
		return nil, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: fmt.Sprintf("Invalid UE Context ID format: %v", err),
		}
	}

	if existingUe := s.findUEContextById(ueContextId); existingUe != nil {
		logger.SbiLog.Warnf("UE Context already exists for ID: %s", ueContextId)
		return nil, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Conflict",
			Status: http.StatusConflict,
			Detail: fmt.Sprintf("UE Context already exists for ID: %s", ueContextId),
		}
	}

	ue := s.createInternalUEContext(createData)
	if ue == nil {
		logger.SbiLog.Error("Failed to create internal UE context")
		return nil, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Internal Server Error",
			Status: http.StatusInternalServerError,
			Detail: "Failed to create UE context",
		}
	}

	response := buildUeContextCreatedData(ue, createData)

	logger.SbiLog.Infof("UE Context created successfully for SUPI: %s, AMF UE NGAP ID: %d", ue.Supi, ue.AmfUeNgapId)
	return response, nil
}

func (s *Server) ReleaseUEContext(ueContextId string, releaseData *UEContextRelease) *ProblemDetails {
	logger.SbiLog.Infof("Releasing UE Context for UE ID: %s", ueContextId)

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

	if releaseData.NgapCause != nil {
		logger.SbiLog.Infof("Release cause: group=%d, value=%d", releaseData.NgapCause.Group, releaseData.NgapCause.Value)
	}

	s.amfContext.DeleteUEContext(ue.AmfUeNgapId)

	logger.SbiLog.Infof("UE Context released for AMF UE NGAP ID: %d", ue.AmfUeNgapId)
	return nil
}

func (s *Server) createInternalUEContext(createData *UeContextCreateData) *context.UEContext {

	ue := s.amfContext.NewUEContext(0)
	if ue == nil {
		return nil
	}

	ue.Supi = createData.Supi
	ue.Pei = createData.Pei

	if createData.LocationInfo != nil {
		if createData.LocationInfo.NrLocation != nil && createData.LocationInfo.NrLocation.Tai != nil {
			ue.Tai = ToInternalTai(createData.LocationInfo.NrLocation.Tai)
			if createData.LocationInfo.NrLocation.Ncgi != nil {
				ue.CellId = createData.LocationInfo.NrLocation.Ncgi.NrCellId
			}
		} else if createData.LocationInfo.EutraLocation != nil && createData.LocationInfo.EutraLocation.Tai != nil {
			ue.Tai = ToInternalTai(createData.LocationInfo.EutraLocation.Tai)
			if createData.LocationInfo.EutraLocation.Ecgi != nil {
				ue.CellId = createData.LocationInfo.EutraLocation.Ecgi.EutraCellId
			}
		}
	}

	switch createData.AccessType {
	case "3GPP_ACCESS":
		ue.AccessType = context.AccessType3GPP
	case "NON_3GPP_ACCESS":
		ue.AccessType = context.AccessTypeNon3GPP
	default:
		ue.AccessType = context.AccessType3GPP
	}

	ue.RegistrationState = context.Registered
	ue.RmState = context.RmRegistered
	ue.CmState = context.CmConnected

	ue.PduSessions = make(map[int32]*context.PduSessionContext)

	return ue
}

func buildUeContextCreatedData(ue *context.UEContext, createData *UeContextCreateData) *UeContextCreatedData {
	response := &UeContextCreatedData{
		UeContext: &UeContext{
			Supi:             ue.Supi,
			Pei:              ue.Pei,
			UdmGroupId:       createData.UdmGroupId,
			AusfGroupId:      createData.AusfGroupId,
			RoutingIndicator: createData.RoutingIndicator,
		},
		PduSessionList: []PduSessionContext{},
	}

	for _, session := range ue.PduSessions {
		response.PduSessionList = append(response.PduSessionList, PduSessionContext{
			PduSessionId: session.PduSessionId,
			Dnn:          session.Dnn,
			Snssai:       ToSbiSnssai(session.Snssai),
		})
	}

	return response
}

func (s *Server) findUEContextById(ueContextId string) *context.UEContext {
	var targetSupi string

	if strings.HasPrefix(ueContextId, "imsi-") {
		targetSupi = strings.TrimPrefix(ueContextId, "imsi-")
		targetSupi = "imsi-" + targetSupi
	} else if strings.HasPrefix(ueContextId, "5g-guti-") {

		guti := strings.TrimPrefix(ueContextId, "5g-guti-")
		return s.findUEContextByGuti(guti)
	} else if strings.HasPrefix(ueContextId, "nai-") {

		targetSupi = ueContextId
	} else {

		targetSupi = ueContextId
	}

	var foundUe *context.UEContext
	s.amfContext.UeContexts.Range(func(key, value interface{}) bool {
		ue := value.(*context.UEContext)
		if ue.Supi == targetSupi {
			foundUe = ue
			return false
		}
		return true
	})

	return foundUe
}

func (s *Server) findUEContextByGuti(guti string) *context.UEContext {
	var foundUe *context.UEContext
	s.amfContext.UeContexts.Range(func(key, value interface{}) bool {
		ue := value.(*context.UEContext)
		if ue.Guti == guti {
			foundUe = ue
			return false
		}
		return true
	})
	return foundUe
}

func (s *Server) handleGetUEContext(w http.ResponseWriter, r *http.Request, ueContextId string) {
	logger.SbiLog.Infof("Getting UE Context for ID: %s", ueContextId)

	ue := s.findUEContextById(ueContextId)
	if ue == nil {
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Not Found",
			Status: http.StatusNotFound,
			Detail: fmt.Sprintf("UE Context not found for ID: %s", ueContextId),
		})
		return
	}

	response := s.GetUEContext(ueContextId, ue)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.SbiLog.Errorf("Failed to encode response: %v", err)
	}
}

func (s *Server) GetUEContext(ueContextId string, ue *context.UEContext) *UeContextCreatedData {
	logger.SbiLog.Infof("Retrieving UE Context: %s", ueContextId)

	ueContext := &UeContext{
		Supi:                ue.Supi,
		SupiUnauthInd:       false,
		Pei:                 ue.Pei,
		IabOperationAllowed: false,
	}

	pduSessions := []PduSessionContext{}
	for _, pduSession := range ue.PduSessions {
		pduSessionCtx := PduSessionContext{
			PduSessionId: pduSession.PduSessionId,
			Snssai:       ToSbiSnssai(pduSession.Snssai),
			Dnn:          pduSession.Dnn,
		}
		pduSessions = append(pduSessions, pduSessionCtx)
	}

	return &UeContextCreatedData{
		UeContext:      ueContext,
		PduSessionList: pduSessions,
	}
}

func validateUeContextId(ueContextId string) error {
	if ueContextId == "" {
		return fmt.Errorf("UE Context ID cannot be empty")
	}

	validPrefixes := []string{"5g-guti-", "imsi-", "nai-", "gli-", "gci-", "imei-", "imeisv-"}
	for _, prefix := range validPrefixes {
		if strings.HasPrefix(ueContextId, prefix) {
			return nil
		}
	}

	return nil
}

func parseMultipartRequest(r *http.Request) (*UeContextCreateData, []byte, error) {
	contentType := r.Header.Get("Content-Type")

	if strings.Contains(contentType, "application/json") {
		var createData UeContextCreateData
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read request body: %w", err)
		}

		if err := json.Unmarshal(body, &createData); err != nil {
			return nil, nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
		}

		return &createData, nil, nil
	}

	if strings.Contains(contentType, "multipart/related") {
		mediaType, params, err := mime.ParseMediaType(contentType)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse content type: %w", err)
		}

		if !strings.HasPrefix(mediaType, "multipart/") {
			return nil, nil, fmt.Errorf("expected multipart content type")
		}

		boundary := params["boundary"]
		if boundary == "" {
			return nil, nil, fmt.Errorf("boundary not found in content type")
		}

		reader := multipart.NewReader(r.Body, boundary)
		var createData *UeContextCreateData
		var n2Info []byte

		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, nil, fmt.Errorf("failed to read multipart: %w", err)
			}

			partContentType := part.Header.Get("Content-Type")
			contentId := strings.Trim(part.Header.Get("Content-Id"), "<>")

			data, err := io.ReadAll(part)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to read part data: %w", err)
			}

			if strings.Contains(partContentType, "application/json") {
				var jsonData UeContextCreateData
				if err := json.Unmarshal(data, &jsonData); err != nil {
					return nil, nil, fmt.Errorf("failed to unmarshal JSON part: %w", err)
				}
				createData = &jsonData
			} else if contentId != "" && strings.Contains(partContentType, "application/vnd.3gpp.ngap") {
				n2Info = data
			}

			part.Close()
		}

		if createData == nil {
			return nil, nil, fmt.Errorf("JSON data part not found in multipart request")
		}

		return createData, n2Info, nil
	}

	return nil, nil, fmt.Errorf("unsupported Content-Type: %s", contentType)
}

func (s *Server) AssignEbi(ueContextId string, assignData *AssignEbiData) (*AssignedEbiData, *ProblemDetails) {
	logger.SbiLog.Infof("Assigning EBI for UE: %s, PDU Session ID: %d", ueContextId, assignData.PduSessionId)

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

	session, exists := ue.PduSessions[assignData.PduSessionId]
	if !exists {
		logger.SbiLog.Warnf("PDU Session %d not found for UE: %s", assignData.PduSessionId, ueContextId)
		return nil, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Not Found",
			Status: http.StatusNotFound,
			Detail: fmt.Sprintf("PDU Session %d not found", assignData.PduSessionId),
		}
	}

	response := &AssignedEbiData{
		PduSessionId:    assignData.PduSessionId,
		AssignedEbiList: []EbiArpMapping{},
		FailedArpList:   []Arp{},
		ReleasedEbiList: assignData.ReleasedEbiList,
	}

	if session.AllocatedEbis == nil {
		session.AllocatedEbis = make(map[int32]int32)
	}

	nextEbi := s.getNextAvailableEbi(session)

	for _, arp := range assignData.ArpList {
		if nextEbi > 15 {
			logger.SbiLog.Warnf("No more EBI available, adding to failed list")
			response.FailedArpList = append(response.FailedArpList, arp)
			continue
		}

		mapping := EbiArpMapping{
			EpsBearerId: nextEbi,
			Arp:         &arp,
		}
		response.AssignedEbiList = append(response.AssignedEbiList, mapping)

		session.AllocatedEbis[nextEbi] = arp.PriorityLevel

		nextEbi++
	}

	for _, releasedEbi := range assignData.ReleasedEbiList {
		delete(session.AllocatedEbis, releasedEbi)
		logger.SbiLog.Infof("Released EBI %d for PDU Session %d", releasedEbi, assignData.PduSessionId)
	}

	for _, modifiedEbi := range assignData.ModifiedEbiList {
		if modifiedEbi.Arp != nil {
			session.AllocatedEbis[modifiedEbi.EpsBearerId] = modifiedEbi.Arp.PriorityLevel
			logger.SbiLog.Infof("Modified EBI %d for PDU Session %d", modifiedEbi.EpsBearerId, assignData.PduSessionId)
		}
	}

	logger.SbiLog.Infof("EBI assignment complete. Assigned: %d, Failed: %d, Released: %d",
		len(response.AssignedEbiList), len(response.FailedArpList), len(response.ReleasedEbiList))

	return response, nil
}

func (s *Server) getNextAvailableEbi(session *context.PduSessionContext) int32 {
	if session.AllocatedEbis == nil {
		return 5
	}

	for ebi := int32(5); ebi <= 15; ebi++ {
		if _, exists := session.AllocatedEbis[ebi]; !exists {
			return ebi
		}
	}

	return 16
}

func (s *Server) QueryUEContexts(supi string, gpsi string) (*UeContextSearchResult, *ProblemDetails) {
	logger.SbiLog.Infof("Querying UE Contexts - SUPI: %s, GPSI: %s", supi, gpsi)

	result := &UeContextSearchResult{
		UeContexts: []SearchedUeContext{},
		TotalCount: 0,
	}

	s.amfContext.UeContexts.Range(func(key, value interface{}) bool {
		ue := value.(*context.UEContext)

		if supi != "" && ue.Supi != supi {
			return true
		}

		var ueContextId string
		if ue.Supi != "" {
			ueContextId = ue.Supi
		} else if ue.Guti != "" {
			ueContextId = "5g-guti-" + ue.Guti
		} else {
			ueContextId = fmt.Sprintf("amf-ue-ngap-id-%d", ue.AmfUeNgapId)
		}

		accessType := ""
		switch ue.AccessType {
		case context.AccessType3GPP:
			accessType = "3GPP_ACCESS"
		case context.AccessTypeNon3GPP:
			accessType = "NON_3GPP_ACCESS"
		}

		var tai *Tai
		if ue.Tai.PlmnId.Mcc != "" {
			tai = &Tai{
				PlmnId: &PlmnId{
					Mcc: ue.Tai.PlmnId.Mcc,
					Mnc: ue.Tai.PlmnId.Mnc,
				},
				Tac: ue.Tai.Tac,
			}
		}

		ueInfo := SearchedUeContext{
			UeContextId:     ueContextId,
			Supi:            ue.Supi,
			AmfUeNgapId:     ue.AmfUeNgapId,
			Pei:             ue.Pei,
			AccessType:      accessType,
			CmState:         string(ue.CmState),
			RmState:         string(ue.RmState),
			Tai:             tai,
			PduSessionCount: len(ue.PduSessions),
		}

		result.UeContexts = append(result.UeContexts, ueInfo)
		result.TotalCount++

		return true
	})

	logger.SbiLog.Infof("Found %d UE contexts", result.TotalCount)
	return result, nil
}

func (s *Server) UEContextTransfer(ueContextId string, reqData *UeContextTransferReqData, binaryParts map[string][]byte) (*UeContextTransferRspData, *ProblemDetails) {
	logger.SbiLog.Infof("UE Context Transfer for UE ID: %s, Reason: %s", ueContextId, reqData.Reason)

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

	var ueRadioCapability *N2InfoContent
	if n1Content, exists := binaryParts["ueRadioCapability"]; exists && len(n1Content) > 0 {
		ueRadioCapability = &N2InfoContent{
			NgapData: &RefToBinaryData{
				ContentId: "ueRadioCapability",
			},
		}
	}

	response := &UeContextTransferRspData{
		UeContext: &UeContext{
			Supi:             ue.Supi,
			Pei:              ue.Pei,
			UdmGroupId:       "",
			AusfGroupId:      "",
			RoutingIndicator: "",
		},
		UeRadioCapability: ueRadioCapability,
		SupportedFeatures: reqData.SupportedFeatures,
	}

	logger.SbiLog.Infof("UE Context Transfer successful for UE: %s", ue.Supi)
	return response, nil
}

func (s *Server) RegistrationStatusUpdate(ueContextId string, reqData *UeRegStatusUpdateReqData) (*UeRegStatusUpdateRspData, *ProblemDetails) {
	logger.SbiLog.Infof("Registration Status Update for UE ID: %s, Transfer Status: %s", ueContextId, reqData.TransferStatus)

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

	if reqData.TransferStatus == string(UeContextTransferStatusTransferred) {
		logger.SbiLog.Infof("UE Context successfully transferred for UE: %s", ue.Supi)

		for _, sessionId := range reqData.ToReleaseSessionList {
			if session, exists := ue.PduSessions[sessionId]; exists {
				logger.SbiLog.Infof("Releasing PDU Session %d for UE: %s", sessionId, ue.Supi)
				delete(ue.PduSessions, sessionId)
				logger.SbiLog.Infof("Released PDU Session %d (DNN: %s) for UE: %s", sessionId, session.Dnn, ue.Supi)
			} else {
				logger.SbiLog.Warnf("PDU Session %d not found for UE: %s", sessionId, ue.Supi)
			}
		}

		for _, smfChangeInfo := range reqData.SmfChangeInfoList {
			logger.SbiLog.Infof("SMF change indication: %s for PDU Sessions: %v", smfChangeInfo.SmfChangeInd, smfChangeInfo.PduSessionIdList)
		}

		if reqData.PcfReselectedInd {
			logger.SbiLog.Infof("PCF reselection indicated for UE: %s", ue.Supi)
		}

		response := &UeRegStatusUpdateRspData{
			RegStatusTransferComplete: true,
		}

		logger.SbiLog.Infof("Registration status update completed successfully for UE: %s", ue.Supi)
		return response, nil
	}

	if reqData.TransferStatus == string(UeContextTransferStatusNotTransferred) {
		logger.SbiLog.Warnf("UE Context transfer not completed for UE: %s", ue.Supi)

		response := &UeRegStatusUpdateRspData{
			RegStatusTransferComplete: false,
		}

		return response, nil
	}

	logger.SbiLog.Errorf("Invalid transfer status: %s", reqData.TransferStatus)
	return nil, &ProblemDetails{
		Type:   "about:blank",
		Title:  "Bad Request",
		Status: http.StatusBadRequest,
		Detail: fmt.Sprintf("Invalid transfer status: %s", reqData.TransferStatus),
	}
}

func (s *Server) RelocateUEContext(ueContextId string, relocateData *UeContextRelocateData, binaryParts map[string][]byte) (*UeContextRelocatedData, *ProblemDetails) {
	logger.SbiLog.Infof("Relocating UE Context for UE ID: %s", ueContextId)

	if relocateData.UeContext == nil {
		logger.SbiLog.Error("UE Context is required in relocate request")
		return nil, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "UE Context is required",
		}
	}

	if relocateData.TargetId == nil {
		logger.SbiLog.Error("Target ID is required in relocate request")
		return nil, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "Target ID is required",
		}
	}

	if relocateData.SourceToTargetData == nil {
		logger.SbiLog.Error("Source to target data is required in relocate request")
		return nil, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "Source to target data is required",
		}
	}

	ue := s.findUEContextById(ueContextId)
	if ue == nil {
		logger.SbiLog.Warnf("UE Context not found for ID: %s, creating new context", ueContextId)

		ue = s.amfContext.NewUEContext(0)
		if ue == nil {
			logger.SbiLog.Error("Failed to create new UE context")
			return nil, &ProblemDetails{
				Type:   "about:blank",
				Title:  "Internal Server Error",
				Status: http.StatusInternalServerError,
				Detail: "Failed to create UE context",
			}
		}
	}

	if relocateData.UeContext.Supi != "" {
		ue.Supi = relocateData.UeContext.Supi
	}
	if relocateData.UeContext.Pei != "" {
		ue.Pei = relocateData.UeContext.Pei
	}

	if relocateData.TargetId.Tai != nil {
		ue.Tai = ToInternalTai(relocateData.TargetId.Tai)
		logger.SbiLog.Infof("Updated TAI for UE: MCC=%s, MNC=%s, TAC=%s",
			ue.Tai.PlmnId.Mcc, ue.Tai.PlmnId.Mnc, ue.Tai.Tac)
	}

	for _, pduSessionInfo := range relocateData.PduSessionList {
		if pduSessionInfo.PduSessionId > 0 {
			if session, exists := ue.PduSessions[pduSessionInfo.PduSessionId]; exists {
				logger.SbiLog.Infof("Updating PDU Session %d for relocated UE", pduSessionInfo.PduSessionId)
				if pduSessionInfo.SNssai != nil {
					session.Snssai = ToInternalSnssai(pduSessionInfo.SNssai)
				}
			} else {
				logger.SbiLog.Infof("Creating new PDU Session %d for relocated UE", pduSessionInfo.PduSessionId)
				newSession := &context.PduSessionContext{
					PduSessionId: pduSessionInfo.PduSessionId,
					Snssai:       ToInternalSnssai(pduSessionInfo.SNssai),
				}
				ue.PduSessions[pduSessionInfo.PduSessionId] = newSession
			}
		}
	}

	if relocateData.NgapCause != nil {
		logger.SbiLog.Infof("Relocation cause: group=%d, value=%d",
			relocateData.NgapCause.Group, relocateData.NgapCause.Value)
	}

	response := &UeContextRelocatedData{
		UeContext: &UeContext{
			Supi:             ue.Supi,
			Pei:              ue.Pei,
			UdmGroupId:       relocateData.UeContext.UdmGroupId,
			AusfGroupId:      relocateData.UeContext.AusfGroupId,
			RoutingIndicator: relocateData.UeContext.RoutingIndicator,
		},
	}

	logger.SbiLog.Infof("UE Context relocated successfully for UE: %s", ue.Supi)
	return response, nil
}

func (s *Server) CancelRelocateUEContext(ueContextId string, cancelData *UeContextCancelRelocateData, binaryParts map[string][]byte) *ProblemDetails {
	logger.SbiLog.Infof("Cancelling UE Context Relocation for UE ID: %s", ueContextId)

	if cancelData.RelocationCancelRequest == nil {
		logger.SbiLog.Error("Relocation cancel request is required")
		return &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "Relocation cancel request is required",
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

	if cancelData.Supi != "" && ue.Supi != cancelData.Supi {
		logger.SbiLog.Warnf("SUPI mismatch: expected %s, got %s", ue.Supi, cancelData.Supi)
		return &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "SUPI mismatch",
		}
	}

	logger.SbiLog.Infof("Relocation cancelled for UE: %s", ue.Supi)
	return nil
}
