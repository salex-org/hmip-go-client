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
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	LookupEndpoint = "https://lookup.homematic.com:48335/getHost"
	ApiVersion     = "12"
	DeviceType     = "Computer"
	Language       = "de-DE"
	OSType         = "linux"

	EnvVarNameAccessPointSGTIN = "HMIP_AP_SGTIN"
	EnvVarNamePIN              = "HMIP_PIN"
	EnvVarNameClientId         = "HMIP_CLIENT_ID"
	EnvVarNameClientName       = "HMIP_CLIENT_NAME"
	EnvVarNameDeviceId         = "HMIP_DEVICE_ID"
	EnvVarNameClientAuthToken  = "HMIP_CLIENT_AUTH_TOKEN"
	EnvVarNameAuthToken        = "HMIP_AUTH_TOKEN"
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
	AuthToken        string `json:"authToken,omitempty"`
}

type GetAuthTokenResponse struct {
	AuthToken string `json:"authToken"`
}

type ConfirmAuthTokenResponse struct {
	ClientID string `json:"clientId"`
}

type GetStateRequest struct {
	ClientCharacteristics ClientCharacteristics `json:"clientCharacteristics"`
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
	Devices map[string]Device             `json:"devices"`
	Clients map[string]ClientRegistration `json:"clients"`
}

type Device struct {
	ID               string                       `json:"id"`
	Name             string                       `json:"label"`
	Type             string                       `json:"type"`
	Model            string                       `json:"modelType"`
	SGTIN            string                       `json:"serializedGlobalTradeItemNumber"`
	Channels         map[string]FunctionalChannel `json:"functionalChannels"`
	LastStatusUpdate HomematicTimestamp           `json:"lastStatusUpdate"`
}

type FunctionalChannel struct {
	Type         string  `json:"functionalChannelType"`
	Temperature  float64 `json:"actualTemperature,omitempty"`
	Humidity     int     `json:"humidity,omitempty"`
	VapourAmount float64 `json:"vaporAmount,omitempty"`
}

type ClientRegistration struct {
	ID       string             `json:"id"`
	Name     string             `json:"label"`
	Created  HomematicTimestamp `json:"createdAtTimestamp"`
	LastSeen HomematicTimestamp `json:"lastSeenAtTimestamp"`
}

type Client struct {
	config     *Config
	httpClient *http.Client
}

type Config struct {
	AccessPointSGTIN  string
	ClientName        string
	RestEndpoint      string
	WebSocketEndpoint string
	LookupEndpoint    string
	PIN               string
	DeviceID          string
	ClientID          string
	ClientAuthToken   string
	AuthToken         string
}

type HomematicTimestamp time.Time

func (t *HomematicTimestamp) MarshalJSON() ([]byte, error) {
	s := strconv.Itoa(int(time.Time(*t).Unix()))
	return []byte(s), nil
}

func (t *HomematicTimestamp) UnmarshalJSON(value []byte) error {
	unix, err := strconv.Atoi(string(value))
	if err != nil {
		return err
	}
	*t = HomematicTimestamp(time.Unix(int64(unix), 0))
	return nil
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
		LookupEndpoint:   LookupEndpoint,
		AccessPointSGTIN: os.Getenv(EnvVarNameAccessPointSGTIN),
		PIN:              os.Getenv(EnvVarNamePIN),
		ClientID:         os.Getenv(EnvVarNameClientId),
		ClientName:       os.Getenv(EnvVarNameClientName),
		ClientAuthToken:  os.Getenv(EnvVarNameClientAuthToken),
		DeviceID:         os.Getenv(EnvVarNameDeviceId),
		AuthToken:        os.Getenv(EnvVarNameAuthToken),
	}
	return &config, nil
}

func GetClientWithConfig(config *Config) (*Client, error) {
	_ = config.lookupEndpoints()
	client := Client{
		config: config,
		httpClient: &http.Client{
			Transport: &HomematicRoundTripper{
				Origin: http.DefaultTransport,
				config: config,
			},
			Timeout: 30 * time.Second,
		},
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
	requestBody, _ := json.Marshal(GetStateRequest{
		ClientCharacteristics: c.config.getClientCharacteristics(),
	})
	var response *http.Response
	err := retry.Do(func() error {
		request, err := http.NewRequest("POST", c.config.RestEndpoint+"/hmip/home/getCurrentState", bytes.NewReader(requestBody))
		if err != nil {
			return retry.Unrecoverable(err)
		}
		response, err = c.httpClient.Do(request)
		if err != nil {
			return err
		}
		if response.StatusCode != 200 {
			return errors.New(fmt.Sprintf("Error on reading state (%s)", response.Status))
		}
		return nil
	}, retry.OnRetry(func(_ uint, _ error) {
		_ = c.config.lookupEndpoints()
	}), retry.Attempts(2))
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(response.Body)
	responseBody, _ := io.ReadAll(response.Body)
	state := State{}
	err = json.Unmarshal(responseBody, &state)
	if err != nil {
		return nil, err
	}
	return &state, nil
}

func (c *Config) createClientAuthToken() {
	tokenDigest := digest.SHA512.FromBytes([]byte(c.AccessPointSGTIN + "jiLpVitHvWnIGD1yo7MA"))
	c.ClientAuthToken = strings.ToUpper(tokenDigest.Hex())
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
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(response.Body)
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
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(response.Body)
	responseBody, _ := io.ReadAll(response.Body)
	result := ConfirmAuthTokenResponse{}
	responseErr = json.Unmarshal(responseBody, &result)
	if responseErr != nil {
		return responseErr
	}
	c.ClientID = result.ClientID
	return nil
}

func (c *Config) getClientCharacteristics() ClientCharacteristics {
	return ClientCharacteristics{
		APIVersion: ApiVersion,
		ClientName: c.ClientName,
		DeviceType: DeviceType,
		Language:   Language,
		OSType:     OSType,
	}
}

func (c *Config) lookupEndpoints() error {
	requestBody, _ := json.Marshal(HostsLookupRequest{
		AccessPointSGTIN:      c.AccessPointSGTIN,
		ClientCharacteristics: c.getClientCharacteristics(),
	})
	response, err := http.Post(c.LookupEndpoint, "application/json", bytes.NewReader(requestBody))
	if err != nil {
		return err
	}
	if response.StatusCode != 200 {
		return errors.New(fmt.Sprintf("Error on endpoint lookup (%s)", response.Status))
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(response.Body)
	responseBody, _ := io.ReadAll(response.Body)
	result := HostsLookupResponse{}
	err = json.Unmarshal(responseBody, &result)
	if err != nil {
		return err
	}
	c.RestEndpoint = result.RestEndpoint
	c.WebSocketEndpoint = result.WebSocketEndpoint
	return nil
}
