package consumer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gavin/amf/internal/logger"
)

type AUSFClient struct {
	ausfUri string
	client  *http.Client
}

func NewAUSFClient(ausfUri string) *AUSFClient {
	return &AUSFClient{
		ausfUri: ausfUri,
		client:  &http.Client{},
	}
}

type UEAuthenticationRequest struct {
	Supi                  string                 `json:"supi"`
	ServingNetworkName    string                 `json:"servingNetworkName"`
	ResynchronizationInfo *ResynchronizationInfo `json:"resynchronizationInfo,omitempty"`
}

type ResynchronizationInfo struct {
	Rand string `json:"rand"`
	Auts string `json:"auts"`
}

type UEAuthenticationResponse struct {
	AuthType             string            `json:"authType"`
	Links                map[string]string `json:"_links"`
	AuthenticationVector interface{}       `json:"authenticationVector,omitempty"`
}

type UEAuthenticationConfirmation struct {
	Res string `json:"res"`
}

type UEAuthenticationConfirmationResponse struct {
	AuthResult string `json:"authResult"`
	Supi       string `json:"supi,omitempty"`
	Kseaf      string `json:"kseaf,omitempty"`
}

func (c *AUSFClient) RequestAuthentication(supi, servingNetworkName string) (*UEAuthenticationResponse, error) {
	logger.ConsumerLog.Infof("Request Authentication for SUPI: %s", supi)

	reqBody := UEAuthenticationRequest{
		Supi:               supi,
		ServingNetworkName: servingNetworkName,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/nausf-auth/v1/ue-authentications", c.ausfUri)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("authentication request failed with status: %d", resp.StatusCode)
	}

	var authResp UEAuthenticationResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return nil, err
	}

	logger.ConsumerLog.Info("Authentication request successful")
	return &authResp, nil
}

func (c *AUSFClient) ConfirmAuthentication(authCtxId, res string) (*UEAuthenticationConfirmationResponse, error) {
	logger.ConsumerLog.Infof("Confirm Authentication for authCtxId: %s", authCtxId)

	reqBody := UEAuthenticationConfirmation{
		Res: res,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/nausf-auth/v1/ue-authentications/%s/5g-aka-confirmation", c.ausfUri, authCtxId)
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("authentication confirmation failed with status: %d", resp.StatusCode)
	}

	var confirmResp UEAuthenticationConfirmationResponse
	if err := json.NewDecoder(resp.Body).Decode(&confirmResp); err != nil {
		return nil, err
	}

	logger.ConsumerLog.Info("Authentication confirmation successful")
	return &confirmResp, nil
}
