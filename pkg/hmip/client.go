package hmip

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/avast/retry-go/v4"
	"github.com/google/uuid"
	"github.com/opencontainers/go-digest"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	LookupEndpoint = "https://lookup.homematic.com:48335/getHost"
	ApiVersion     = "12"
)

type HostsLookupRequest struct {
	AccessPointSGTIN      string                `json:"id"`
	ClientCharacteristics ClientCharacteristics `json:"clientCharacteristics"`
}

type HostsLookupResponse struct {
	RestEndpoint      string `json:"urlREST"`
	WebSocketEndpoint string `json:"urlWebSocket"`
}

type RegisterClientRequest struct {
	DeviceID         string `json:"deviceId"`
	DeviceName       string `json:"deviceName,omitempty"`
	AccessPointSGTIN string `json:"sgtin,omitempty"`
	AuthToken        string `json:"authToken,omitempty""`
}

type GetAuthTokenResponse struct {
	AuthToken string `json:"authToken"`
}

type ConfirmAuthTokenResponse struct {
	ClientID string `json:"clientId"`
}

type ClientCharacteristics struct {
	APIVersion         string `json:"apiVersion"`
	ClientName         string `json:"applicationIdentifier"`
	ClientVersion      string `json:"applicationVersion"`
	DeviceManufacturer string `json:"deviceManufacturer"`
	DeviceType         string `json:"deviceType"`
	Language           string `json:"language"`
	OSType             string `json:"osType"`
	OSVersion          string `json:"osVersion"`
}

type State struct {
}

type Client struct {
	config *Config
}

type Config struct {
	AccessPointSGTIN  string
	ClientName        string
	ClientVersion     string
	RestEndpoint      string
	WebSocketEndpoint string
	LookupEndpoint    string
	PIN               string
	DeviceID          string
	ClientID          string
	ClientAuthToken   string
	AuthToken         string
}

type HomematicRoundTripper struct {
	Origin http.RoundTripper
	config *Config
}

func (r *HomematicRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	request.Header.Set("Content-Type", "application/json")
	request.Header["VERSION"] = []string{ApiVersion}
	request.Header["CLIENTAUTH"] = []string{r.config.ClientAuthToken}
	request.Header["AUTHTOKEN"] = []string{r.config.AuthToken}
	return r.Origin.RoundTrip(request)
}

func GetConfig() (*Config, error) {
	config := Config{
		LookupEndpoint: LookupEndpoint,
	}
	return &config, nil
}

func GetClientWithConfig(config *Config) (*Client, error) {
	client := Client{
		config: config,
	}
	return &client, nil
}

func GetClient() (*Client, error) {
	config, err := GetConfig()
	if err != nil {
		return nil, err
	} else {
		return GetClientWithConfig(config)
	}
}

func (c *Config) RegisterClient(handshakeCallback func()) error {
	err := c.lookupEndpoints()
	if err != nil {
		return err
	}
	c.createClientAuthToken()
	c.createDeviceID()
	httpClient := &http.Client{
		Transport: &HomematicRoundTripper{
			Origin: http.DefaultTransport,
			config: c,
		},
		Timeout: 30 * time.Second,
	}
	err = c.connectionRequest(httpClient)
	if err != nil {
		return err
	}
	handshakeCallback()
	err = c.requestAcknowledge(httpClient)
	if err != nil {
		return err
	}
	err = c.requestAuthToken(httpClient)
	if err != nil {
		return err
	}
	return c.confirmAuthToken(httpClient)
}

func (c *Client) LoadCurrentState() (*State, error) {
	state := State{}
	return &state, nil
}

func (c *Config) createClientAuthToken() {
	digest := digest.SHA512.FromBytes([]byte(c.AccessPointSGTIN + "jiLpVitHvWnIGD1yo7MA"))
	c.ClientAuthToken = strings.ToUpper(digest.Hex())
}

func (c *Config) createDeviceID() {
	c.DeviceID = uuid.New().String()
}

func (c *Config) connectionRequest(httpClient *http.Client) error {
	requestBody, _ := json.Marshal(RegisterClientRequest{
		DeviceID:         c.DeviceID,
		DeviceName:       c.ClientName,
		AccessPointSGTIN: c.AccessPointSGTIN,
	})
	request, requestErr := http.NewRequest("POST", c.RestEndpoint+"/hmip/auth/connectionRequest", bytes.NewReader(requestBody))
	if requestErr != nil {
		return requestErr
	}
	response, responseErr := httpClient.Do(request)
	if responseErr != nil {
		return responseErr
	}
	if response.StatusCode != 200 {
		return errors.New(fmt.Sprintf("Error on connection request (%s)", response.Status))
	}
	return nil
}

func (c *Config) requestAcknowledge(httpClient *http.Client) error {
	requestBody, _ := json.Marshal(RegisterClientRequest{
		DeviceID: c.DeviceID,
	})
	return retry.Do(func() error {
		request, requestErr := http.NewRequest("POST", c.RestEndpoint+"/hmip/auth/isRequestAcknowledged", bytes.NewReader(requestBody))
		if requestErr != nil {
			return retry.Unrecoverable(requestErr)
		}
		response, responseErr := httpClient.Do(request)
		if responseErr != nil {
			return retry.Unrecoverable(responseErr)
		}
		if response.StatusCode == 200 {
			return nil
		}
		responseErr = errors.New(fmt.Sprintf("Error on waiting for acknowledge of new client (%s)", response.Status))
		if response.StatusCode == 400 {
			return responseErr
		}
		return retry.Unrecoverable(requestErr)
	}, retry.Delay(3*time.Second), retry.DelayType(retry.FixedDelay), retry.Attempts(20))
}

func (c *Config) requestAuthToken(httpClient *http.Client) error {
	requestBody, _ := json.Marshal(RegisterClientRequest{
		DeviceID: c.DeviceID,
	})
	request, requestErr := http.NewRequest("POST", c.RestEndpoint+"/hmip/auth/requestAuthToken", bytes.NewReader(requestBody))
	if requestErr != nil {
		return requestErr
	}
	response, responseErr := httpClient.Do(request)
	if responseErr != nil {
		return responseErr
	}
	defer response.Body.Close()
	responseBody, _ := io.ReadAll(response.Body)
	result := GetAuthTokenResponse{}
	responseErr = json.Unmarshal(responseBody, &result)
	if responseErr != nil {
		return responseErr
	}
	c.AuthToken = result.AuthToken
	return nil
}

func (c *Config) confirmAuthToken(httpClient *http.Client) error {
	requestBody, _ := json.Marshal(RegisterClientRequest{
		DeviceID:  c.DeviceID,
		AuthToken: c.AuthToken,
	})
	request, requestErr := http.NewRequest("POST", c.RestEndpoint+"/hmip/auth/confirmAuthToken", bytes.NewReader(requestBody))
	if requestErr != nil {
		return requestErr
	}
	response, responseErr := httpClient.Do(request)
	if responseErr != nil {
		return responseErr
	}
	defer response.Body.Close()
	responseBody, _ := io.ReadAll(response.Body)
	result := ConfirmAuthTokenResponse{}
	responseErr = json.Unmarshal(responseBody, &result)
	if responseErr != nil {
		return responseErr
	}
	c.ClientID = result.ClientID
	return nil
}

func (c *Config) lookupEndpoints() error {
	requestBody, _ := json.Marshal(HostsLookupRequest{
		AccessPointSGTIN: c.AccessPointSGTIN,
		ClientCharacteristics: ClientCharacteristics{
			APIVersion:         ApiVersion,
			ClientName:         c.ClientName,
			ClientVersion:      c.ClientVersion,
			DeviceManufacturer: "",
			DeviceType:         "Computer",
			Language:           "de-DE",
			OSType:             "linux",
			OSVersion:          "",
		},
	})
	response, err := http.Post(LookupEndpoint, "application/json", bytes.NewReader(requestBody))
	if err != nil {
		return err
	}
	if response.StatusCode != 200 {
		return errors.New(fmt.Sprintf("Error on endpoint lookup (%s)", response.Status))
	}
	defer response.Body.Close()
	responseBody, _ := io.ReadAll(response.Body)
	result := HostsLookupResponse{}
	json.Unmarshal(responseBody, &result)
	c.RestEndpoint = result.RestEndpoint
	c.WebSocketEndpoint = result.WebSocketEndpoint
	return nil
}
