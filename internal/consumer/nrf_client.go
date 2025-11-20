package consumer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gavin/amf/internal/logger"
)

type NRFClient struct {
	nrfUri string
	client *http.Client
}

func NewNRFClient(nrfUri string) *NRFClient {
	return &NRFClient{
		nrfUri: nrfUri,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type NFProfile struct {
	NfInstanceId     string                 `json:"nfInstanceId"`
	NfType           string                 `json:"nfType"`
	NfStatus         string                 `json:"nfStatus"`
	HeartBeatTimer   int                    `json:"heartBeatTimer,omitempty"`
	PlmnList         []PlmnId               `json:"plmnList,omitempty"`
	SNssais          []SNssai               `json:"sNssais,omitempty"`
	Ipv4Addresses    []string               `json:"ipv4Addresses,omitempty"`
	AmfInfo          *AmfInfo               `json:"amfInfo,omitempty"`
	NfServices       []NFService            `json:"nfServices,omitempty"`
	Capacity         int                    `json:"capacity,omitempty"`
	Priority         int                    `json:"priority,omitempty"`
	CustomInfo       map[string]interface{} `json:"customInfo,omitempty"`
}

type PlmnId struct {
	Mcc string `json:"mcc"`
	Mnc string `json:"mnc"`
}

type SNssai struct {
	Sst int    `json:"sst"`
	Sd  string `json:"sd,omitempty"`
}

type AmfInfo struct {
	AmfSetId      string         `json:"amfSetId,omitempty"`
	AmfRegionId   string         `json:"amfRegionId,omitempty"`
	GuamiList     []GuamiInfo    `json:"guamiList,omitempty"`
	TaiList       []Tai          `json:"taiList,omitempty"`
	TaiRangeList  []TaiRange     `json:"taiRangeList,omitempty"`
	N2InterfaceAmfInfo *N2InterfaceAmfInfo `json:"n2InterfaceAmfInfo,omitempty"`
}

type GuamiInfo struct {
	PlmnId PlmnId `json:"plmnId"`
	AmfId  string `json:"amfId"`
}

type Tai struct {
	PlmnId PlmnId `json:"plmnId"`
	Tac    string `json:"tac"`
}

type TaiRange struct {
	PlmnId     PlmnId   `json:"plmnId"`
	TacRangeList []TacRange `json:"tacRangeList,omitempty"`
}

type TacRange struct {
	Start   string `json:"start"`
	End     string `json:"end,omitempty"`
	Pattern string `json:"pattern,omitempty"`
}

type N2InterfaceAmfInfo struct {
	Ipv4EndpointAddresses []string `json:"ipv4EndpointAddresses,omitempty"`
	Ipv6EndpointAddresses []string `json:"ipv6EndpointAddresses,omitempty"`
	AmfName               string   `json:"amfName,omitempty"`
}

type NFService struct {
	ServiceInstanceId string   `json:"serviceInstanceId"`
	ServiceName       string   `json:"serviceName"`
	Versions          []NFServiceVersion `json:"versions"`
	Scheme            string   `json:"scheme"`
	NfServiceStatus   string   `json:"nfServiceStatus"`
	ApiPrefix         string   `json:"apiPrefix,omitempty"`
	Ipv4Addresses     []string `json:"ipv4Addresses,omitempty"`
}

type NFServiceVersion struct {
	ApiVersionInUri string `json:"apiVersionInUri"`
	ApiFullVersion  string `json:"apiFullVersion"`
}

type SearchResult struct {
	ValidityPeriod int          `json:"validityPeriod,omitempty"`
	NfInstances    []NFProfile  `json:"nfInstances"`
	SearchId       string       `json:"searchId,omitempty"`
}

type PatchItem struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

func (c *NRFClient) RegisterNF(profile *NFProfile) (*NFProfile, error) {
	logger.ConsumerLog.Infof("Registering NF instance: %s (type: %s)", profile.NfInstanceId, profile.NfType)

	url := fmt.Sprintf("%s/nnrf-nfm/v1/nf-instances/%s", c.nrfUri, profile.NfInstanceId)

	body, err := json.Marshal(profile)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal NF profile: %w", err)
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("NF registration failed with status: %d", resp.StatusCode)
	}

	var registeredProfile NFProfile
	if err := json.NewDecoder(resp.Body).Decode(&registeredProfile); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	logger.ConsumerLog.Infof("NF instance registered successfully: %s", profile.NfInstanceId)
	return &registeredProfile, nil
}

func (c *NRFClient) UpdateNFStatus(nfInstanceId, status string) error {
	logger.ConsumerLog.Infof("Updating NF status: %s to %s", nfInstanceId, status)

	url := fmt.Sprintf("%s/nnrf-nfm/v1/nf-instances/%s", c.nrfUri, nfInstanceId)

	patchItems := []PatchItem{
		{
			Op:    "replace",
			Path:  "/nfStatus",
			Value: status,
		},
	}

	body, err := json.Marshal(patchItems)
	if err != nil {
		return fmt.Errorf("failed to marshal patch items: %w", err)
	}

	req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json-patch+json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("NF status update failed with status: %d", resp.StatusCode)
	}

	logger.ConsumerLog.Infof("NF status updated successfully: %s", nfInstanceId)
	return nil
}

func (c *NRFClient) SendHeartbeat(nfInstanceId string) error {
	logger.ConsumerLog.Infof("Sending heartbeat for NF: %s", nfInstanceId)

	url := fmt.Sprintf("%s/nnrf-nfm/v1/nf-instances/%s", c.nrfUri, nfInstanceId)

	patchItems := []PatchItem{
		{
			Op:    "replace",
			Path:  "/nfStatus",
			Value: "REGISTERED",
		},
	}

	body, err := json.Marshal(patchItems)
	if err != nil {
		return fmt.Errorf("failed to marshal patch items: %w", err)
	}

	req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json-patch+json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("heartbeat failed with status: %d", resp.StatusCode)
	}

	logger.ConsumerLog.Debugf("Heartbeat sent successfully for NF: %s", nfInstanceId)
	return nil
}

func (c *NRFClient) DeregisterNF(nfInstanceId string) error {
	logger.ConsumerLog.Infof("Deregistering NF instance: %s", nfInstanceId)

	url := fmt.Sprintf("%s/nnrf-nfm/v1/nf-instances/%s", c.nrfUri, nfInstanceId)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("NF deregistration failed with status: %d", resp.StatusCode)
	}

	logger.ConsumerLog.Infof("NF instance deregistered successfully: %s", nfInstanceId)
	return nil
}

func (c *NRFClient) DiscoverNF(targetNfType string, requesterNfType string, plmnId *PlmnId, snssais []SNssai) (*SearchResult, error) {
	logger.ConsumerLog.Infof("Discovering NF instances of type: %s", targetNfType)

	url := fmt.Sprintf("%s/nnrf-disc/v1/nf-instances?target-nf-type=%s&requester-nf-type=%s",
		c.nrfUri, targetNfType, requesterNfType)

	if plmnId != nil {
		url += fmt.Sprintf("&target-plmn-list=%s-%s", plmnId.Mcc, plmnId.Mnc)
	}

	for _, snssai := range snssais {
		url += fmt.Sprintf("&snssais={\"sst\":%d", snssai.Sst)
		if snssai.Sd != "" {
			url += fmt.Sprintf(",\"sd\":\"%s\"", snssai.Sd)
		}
		url += "}"
	}

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
		return nil, fmt.Errorf("NF discovery failed with status: %d", resp.StatusCode)
	}

	var result SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	logger.ConsumerLog.Infof("Discovered %d NF instances of type %s", len(result.NfInstances), targetNfType)
	return &result, nil
}

func (c *NRFClient) GetNFProfile(nfInstanceId string) (*NFProfile, error) {
	logger.ConsumerLog.Infof("Getting NF profile for: %s", nfInstanceId)

	url := fmt.Sprintf("%s/nnrf-nfm/v1/nf-instances/%s", c.nrfUri, nfInstanceId)

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
		return nil, fmt.Errorf("get NF profile failed with status: %d", resp.StatusCode)
	}

	var profile NFProfile
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	logger.ConsumerLog.Infof("NF profile retrieved successfully: %s", nfInstanceId)
	return &profile, nil
}
