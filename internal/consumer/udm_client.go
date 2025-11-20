package consumer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gavin/amf/internal/logger"
)

type UDMClient struct {
	udmUri string
	client *http.Client
}

func NewUDMClient(udmUri string) *UDMClient {
	return &UDMClient{
		udmUri: udmUri,
		client: &http.Client{},
	}
}

type AccessAndMobilitySubscriptionData struct {
	Gpsis                       []string                `json:"gpsis,omitempty"`
	InternalGroupIds            []string                `json:"internalGroupIds,omitempty"`
	SubscribedUeAmbr            *Ambr                   `json:"subscribedUeAmbr,omitempty"`
	Nssai                       *Nssai                  `json:"nssai,omitempty"`
	RatRestrictions             []string                `json:"ratRestrictions,omitempty"`
	ForbiddenAreas              []Area                  `json:"forbiddenAreas,omitempty"`
	ServiceAreaRestriction      *ServiceAreaRestriction `json:"serviceAreaRestriction,omitempty"`
	CoreNetworkTypeRestrictions []string                `json:"coreNetworkTypeRestrictions,omitempty"`
	RfspIndex                   int                     `json:"rfspIndex,omitempty"`
	SubsRegTimer                int                     `json:"subsRegTimer,omitempty"`
	UeUsageType                 int                     `json:"ueUsageType,omitempty"`
	MpsPriority                 bool                    `json:"mpsPriority,omitempty"`
	McsPriority                 bool                    `json:"mcsPriority,omitempty"`
}

type Ambr struct {
	Uplink   string `json:"uplink"`
	Downlink string `json:"downlink"`
}

type Nssai struct {
	DefaultSingleNssais []Snssai `json:"defaultSingleNssais"`
	SingleNssais        []Snssai `json:"singleNssais,omitempty"`
}

type Snssai struct {
	Sst int    `json:"sst"`
	Sd  string `json:"sd,omitempty"`
}

type Area struct {
	Tacs []string `json:"tacs,omitempty"`
}

type ServiceAreaRestriction struct {
	RestrictionType string `json:"restrictionType"`
	Areas           []Area `json:"areas,omitempty"`
	MaxNumOfTAs     int    `json:"maxNumOfTAs,omitempty"`
}

type SmfSelectionSubscriptionData struct {
	SubscribedSnssaiInfos map[string]SnssaiInfo `json:"subscribedSnssaiInfos,omitempty"`
}

type SnssaiInfo struct {
	DnnInfos []DnnInfo `json:"dnnInfos"`
}

type DnnInfo struct {
	Dnn                 string `json:"dnn"`
	DefaultDnnIndicator bool   `json:"defaultDnnIndicator,omitempty"`
}

type Amf3GppAccessRegistration struct {
	AmfInstanceId      string   `json:"amfInstanceId"`
	Guami              string   `json:"guami"`
	RatType            string   `json:"ratType"`
	DeregCallbackUri   string   `json:"deregCallbackUri,omitempty"`
	PcscfRestorationCallbackUri string `json:"pcscfRestorationCallbackUri,omitempty"`
	InitialRegistrationInd bool `json:"initialRegistrationInd,omitempty"`
}

func (c *UDMClient) GetAccessAndMobilitySubscriptionData(supi, plmnId string) (*AccessAndMobilitySubscriptionData, error) {
	logger.ConsumerLog.Infof("Get AM Subscription Data for SUPI: %s", supi)

	url := fmt.Sprintf("%s/nudm-sdm/v2/%s/%s/am-data", c.udmUri, supi, plmnId)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get AM subscription data failed with status: %d", resp.StatusCode)
	}

	var amData AccessAndMobilitySubscriptionData
	if err := json.NewDecoder(resp.Body).Decode(&amData); err != nil {
		return nil, err
	}

	logger.ConsumerLog.Info("AM Subscription Data retrieved successfully")
	return &amData, nil
}

func (c *UDMClient) GetSmfSelectionSubscriptionData(supi, plmnId string) (*SmfSelectionSubscriptionData, error) {
	logger.ConsumerLog.Infof("Get SMF Selection Subscription Data for SUPI: %s", supi)

	url := fmt.Sprintf("%s/nudm-sdm/v2/%s/%s/smf-select-data", c.udmUri, supi, plmnId)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get SMF selection data failed with status: %d", resp.StatusCode)
	}

	var smfData SmfSelectionSubscriptionData
	if err := json.NewDecoder(resp.Body).Decode(&smfData); err != nil {
		return nil, err
	}

	logger.ConsumerLog.Info("SMF Selection Subscription Data retrieved successfully")
	return &smfData, nil
}

func (c *UDMClient) RegisterAMF(supi, amfInstanceId, guami string) error {
	logger.ConsumerLog.Infof("Register AMF for SUPI: %s", supi)

	url := fmt.Sprintf("%s/nudm-uecm/v1/%s/registrations/amf-3gpp-access", c.udmUri, supi)

	registration := Amf3GppAccessRegistration{
		AmfInstanceId:          amfInstanceId,
		Guami:                  guami,
		RatType:                "NR",
		InitialRegistrationInd: true,
	}

	body, err := json.Marshal(registration)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("AMF registration failed with status: %d", resp.StatusCode)
	}

	logger.ConsumerLog.Info("AMF registration successful")
	return nil
}
