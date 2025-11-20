package consumer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gavin/amf/internal/logger"
)

type SMFClient struct {
	smfUri string
	client *http.Client
}

func NewSMFClient(smfUri string) *SMFClient {
	return &SMFClient{
		smfUri: smfUri,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type SmContextCreateData struct {
	Supi               string              `json:"supi"`
	Dnn                string              `json:"dnn"`
	SNssai             *SNssai             `json:"sNssai,omitempty"`
	PduSessionId       int32               `json:"pduSessionId"`
	ServingNfId        string              `json:"servingNfId"`
	ServingNetwork     *PlmnId             `json:"servingNetwork,omitempty"`
	RequestType        string              `json:"requestType"`
	N1SmMsg            *RefToBinaryData    `json:"n1SmMsg,omitempty"`
	AnType             string              `json:"anType"`
	RatType            string              `json:"ratType,omitempty"`
	PresenceInLadn     string              `json:"presenceInLadn,omitempty"`
	UeLocation         *UserLocation       `json:"ueLocation,omitempty"`
	UeTimeZone         string              `json:"ueTimeZone,omitempty"`
	PduSessionsActivatableInd bool         `json:"pduSessionsActivatableInd,omitempty"`
	SmContextStatusUri string              `json:"smContextStatusUri,omitempty"`
}

type SmContextCreateResponse struct {
	SmContextRef       string              `json:"smContextRef,omitempty"`
	SmContextId        string              `json:"smContextId,omitempty"`
	N2SmInfo           *RefToBinaryData    `json:"n2SmInfo,omitempty"`
	N2SmInfoType       string              `json:"n2SmInfoType,omitempty"`
	N1SmMsg            *RefToBinaryData    `json:"n1SmMsg,omitempty"`
	AllocatedEbiList   []EbiArpMapping     `json:"allocatedEbiList,omitempty"`
	SupportedFeatures  string              `json:"supportedFeatures,omitempty"`
}

type SmContextUpdateData struct {
	N2SmInfo           *RefToBinaryData    `json:"n2SmInfo,omitempty"`
	N2SmInfoType       string              `json:"n2SmInfoType,omitempty"`
	N1SmMsg            *RefToBinaryData    `json:"n1SmMsg,omitempty"`
	UeLocation         *UserLocation       `json:"ueLocation,omitempty"`
	UeTimeZone         string              `json:"ueTimeZone,omitempty"`
	AnType             string              `json:"anType,omitempty"`
	RatType            string              `json:"ratType,omitempty"`
	PresenceInLadn     string              `json:"presenceInLadn,omitempty"`
	UpCnxState         string              `json:"upCnxState,omitempty"`
	HoState            string              `json:"hoState,omitempty"`
	ToBeSwitch         bool                `json:"toBeSwitch,omitempty"`
	FailedToBeSwitch   bool                `json:"failedToBeSwitch,omitempty"`
}

type SmContextUpdateResponse struct {
	N2SmInfo           *RefToBinaryData    `json:"n2SmInfo,omitempty"`
	N2SmInfoType       string              `json:"n2SmInfoType,omitempty"`
	N1SmMsg            *RefToBinaryData    `json:"n1SmMsg,omitempty"`
	UpCnxState         string              `json:"upCnxState,omitempty"`
	HoState            string              `json:"hoState,omitempty"`
}

type SmContextReleaseData struct {
	Cause              string              `json:"cause,omitempty"`
	NgApCause          *NgApCause          `json:"ngApCause,omitempty"`
	N2SmInfo           *RefToBinaryData    `json:"n2SmInfo,omitempty"`
	N2SmInfoType       string              `json:"n2SmInfoType,omitempty"`
	UeLocation         *UserLocation       `json:"ueLocation,omitempty"`
	UeTimeZone         string              `json:"ueTimeZone,omitempty"`
}

type SmContextReleasedData struct {
	SmallDataRateStatusInd bool             `json:"smallDataRateStatusInd,omitempty"`
	ApnRateStatus          *ApnRateStatus   `json:"apnRateStatus,omitempty"`
	N2SmInfo               *RefToBinaryData `json:"n2SmInfo,omitempty"`
	N2SmInfoType           string           `json:"n2SmInfoType,omitempty"`
}

type RefToBinaryData struct {
	ContentId string `json:"contentId"`
}

type UserLocation struct {
	NrLocation    *NrLocation    `json:"nrLocation,omitempty"`
	EutraLocation *EutraLocation `json:"eutraLocation,omitempty"`
}

type NrLocation struct {
	Tai             Tai    `json:"tai"`
	Ncgi            Ncgi   `json:"ncgi"`
	AgeOfLocationInfo int  `json:"ageOfLocationInfo,omitempty"`
}

type EutraLocation struct {
	Tai             Tai    `json:"tai"`
	Ecgi            Ecgi   `json:"ecgi"`
	AgeOfLocationInfo int  `json:"ageOfLocationInfo,omitempty"`
}

type Ncgi struct {
	PlmnId PlmnId `json:"plmnId"`
	NrCellId string `json:"nrCellId"`
}

type Ecgi struct {
	PlmnId PlmnId `json:"plmnId"`
	EutraCellId string `json:"eutraCellId"`
}

type EbiArpMapping struct {
	EpsBearerId int32 `json:"epsBearerId"`
	Arp         *Arp  `json:"arp,omitempty"`
}

type Arp struct {
	PriorityLevel        int32  `json:"priorityLevel"`
	PreemptCap           string `json:"preemptCap,omitempty"`
	PreemptVuln          string `json:"preemptVuln,omitempty"`
}

type NgApCause struct {
	Group int32 `json:"group"`
	Value int32 `json:"value"`
}

type ApnRateStatus struct {
	RemainPacketsUl      int32     `json:"remainPacketsUl,omitempty"`
	RemainPacketsDl      int32     `json:"remainPacketsDl,omitempty"`
	ValidityTime         time.Time `json:"validityTime,omitempty"`
	RemainExReportsUl    int32     `json:"remainExReportsUl,omitempty"`
	RemainExReportsDl    int32     `json:"remainExReportsDl,omitempty"`
}

func (c *SMFClient) CreateSMContext(createData *SmContextCreateData) (*SmContextCreateResponse, error) {
	logger.ConsumerLog.Infof("Creating SM Context for SUPI: %s, PDU Session ID: %d", createData.Supi, createData.PduSessionId)

	body, err := json.Marshal(createData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal create data: %w", err)
	}

	url := fmt.Sprintf("%s/nsmf-pdusession/v1/sm-contexts", c.smfUri)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SM context creation failed with status: %d", resp.StatusCode)
	}

	var createResp SmContextCreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	logger.ConsumerLog.Infof("SM Context created successfully: %s", createResp.SmContextId)
	return &createResp, nil
}

func (c *SMFClient) UpdateSMContext(smContextRef string, updateData *SmContextUpdateData) (*SmContextUpdateResponse, error) {
	logger.ConsumerLog.Infof("Updating SM Context: %s", smContextRef)

	body, err := json.Marshal(updateData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal update data: %w", err)
	}

	url := fmt.Sprintf("%s/nsmf-pdusession/v1/sm-contexts/%s/modify", c.smfUri, smContextRef)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SM context update failed with status: %d", resp.StatusCode)
	}

	var updateResp SmContextUpdateResponse
	if err := json.NewDecoder(resp.Body).Decode(&updateResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	logger.ConsumerLog.Infof("SM Context updated successfully: %s", smContextRef)
	return &updateResp, nil
}

func (c *SMFClient) ReleaseSMContext(smContextRef string, releaseData *SmContextReleaseData) (*SmContextReleasedData, error) {
	logger.ConsumerLog.Infof("Releasing SM Context: %s", smContextRef)

	body, err := json.Marshal(releaseData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal release data: %w", err)
	}

	url := fmt.Sprintf("%s/nsmf-pdusession/v1/sm-contexts/%s/release", c.smfUri, smContextRef)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return nil, fmt.Errorf("SM context release failed with status: %d", resp.StatusCode)
	}

	if resp.StatusCode == http.StatusNoContent {
		logger.ConsumerLog.Infof("SM Context released successfully: %s", smContextRef)
		return nil, nil
	}

	var releasedData SmContextReleasedData
	if err := json.NewDecoder(resp.Body).Decode(&releasedData); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	logger.ConsumerLog.Infof("SM Context released successfully: %s", smContextRef)
	return &releasedData, nil
}

func (c *SMFClient) RetrieveSMContext(smContextRef string) (*SmContextRetrieveData, error) {
	logger.ConsumerLog.Infof("Retrieving SM Context: %s", smContextRef)

	url := fmt.Sprintf("%s/nsmf-pdusession/v1/sm-contexts/%s", c.smfUri, smContextRef)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SM context retrieval failed with status: %d", resp.StatusCode)
	}

	var retrieveData SmContextRetrieveData
	if err := json.NewDecoder(resp.Body).Decode(&retrieveData); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	logger.ConsumerLog.Infof("SM Context retrieved successfully: %s", smContextRef)
	return &retrieveData, nil
}

type SmContextRetrieveData struct {
	SmContextStatusUri string         `json:"smContextStatusUri,omitempty"`
	UpCnxState         string         `json:"upCnxState,omitempty"`
	HoState            string         `json:"hoState,omitempty"`
	AllocatedEbiList   []EbiArpMapping `json:"allocatedEbiList,omitempty"`
}
