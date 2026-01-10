package hmip

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/opencontainers/go-digest"
)

const (
	LookupEndpoint = "https://lookup.homematic.com:48335/getHost"
	ApiVersion     = "12"
	DeviceType     = "Computer"
	Language       = "de-DE"
	OSType         = runtime.GOOS

	EnvVarNameAccessPointSGTIN = "HMIP_AP_SGTIN"
	EnvVarNamePIN              = "HMIP_PIN"
	EnvVarNameClientId         = "HMIP_CLIENT_ID"
	EnvVarNameClientName       = "HMIP_CLIENT_NAME"
	EnvVarNameDeviceId         = "HMIP_DEVICE_ID"
	EnvVarNameClientAuthToken  = "HMIP_CLIENT_AUTH_TOKEN"
	EnvVarNameAuthToken        = "HMIP_AUTH_TOKEN"
)

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

func (c *Config) RegisterClient(handshakeCallback func()) error {
	err := c.lookupEndpoints()
	if err != nil {
		return err
	}
	c.createClientAuthToken()
	c.createDeviceID()
	httpClient := &http.Client{
		Transport: &homematicRoundTripper{
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

// ======================================================

func (c *Config) createClientAuthToken() {
	tokenDigest := digest.SHA512.FromBytes([]byte(c.getTrimmedAccessPointSGTIN() + "jiLpVitHvWnIGD1yo7MA"))
	c.ClientAuthToken = strings.ToUpper(tokenDigest.Hex())
}

func (c *Config) createDeviceID() {
	c.DeviceID = uuid.New().String()
}

func (c *Config) connectionRequest(httpClient *http.Client) error {
	requestBody, _ := json.Marshal(registerClientRequest{
		DeviceID:         c.DeviceID,
		DeviceName:       c.ClientName,
		AccessPointSGTIN: c.getTrimmedAccessPointSGTIN(),
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
	requestBody, _ := json.Marshal(registerClientRequest{
		DeviceID: c.DeviceID,
	})

	const (
		attempts = 20
		delay    = 3 * time.Second
	)

	var lastErr error

	for i := 0; i < attempts; i++ {
		request, err := http.NewRequest(
			http.MethodPost,
			c.RestEndpoint+"/hmip/auth/isRequestAcknowledged",
			bytes.NewReader(requestBody),
		)
		if err != nil {
			// unrecoverable
			return err
		}

		response, err := httpClient.Do(request)
		if err != nil {
			// unrecoverable
			return err
		}
		response.Body.Close()

		if response.StatusCode == http.StatusOK {
			return nil
		}

		lastErr = fmt.Errorf(
			"error on waiting for acknowledge of new client (%s)",
			response.Status,
		)

		if response.StatusCode != http.StatusBadRequest {
			// unrecoverable
			return lastErr
		}

		// Retry nur bei 400 (Bad Request) nach dem delay
		time.Sleep(delay)
	}

	return lastErr
}

func (c *Config) requestAuthToken(httpClient *http.Client) error {
	requestBody, _ := json.Marshal(registerClientRequest{
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
	result := getAuthTokenResponse{}
	responseErr = json.Unmarshal(responseBody, &result)
	if responseErr != nil {
		return responseErr
	}
	c.AuthToken = result.AuthToken
	return nil
}

func (c *Config) confirmAuthToken(httpClient *http.Client) error {
	requestBody, _ := json.Marshal(registerClientRequest{
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
	result := confirmAuthTokenResponse{}
	responseErr = json.Unmarshal(responseBody, &result)
	if responseErr != nil {
		return responseErr
	}
	c.ClientID = result.ClientID
	return nil
}

func (c *Config) getClientCharacteristics() clientCharacteristics {
	return clientCharacteristics{
		APIVersion: ApiVersion,
		ClientName: c.ClientName,
		DeviceType: DeviceType,
		Language:   Language,
		OSType:     OSType,
	}
}

func (c *Config) lookupEndpoints() error {
	requestBody, _ := json.Marshal(hostsLookupRequest{
		AccessPointSGTIN:      c.getTrimmedAccessPointSGTIN(),
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
	result := hostsLookupResponse{}
	err = json.Unmarshal(responseBody, &result)
	if err != nil {
		return err
	}
	c.RestEndpoint = result.RestEndpoint
	c.WebSocketEndpoint = result.WebSocketEndpoint
	return nil
}

func (c *Config) getTrimmedAccessPointSGTIN() string {
	return strings.ReplaceAll(strings.ToUpper(c.AccessPointSGTIN), "-", "")
}

// ======================================================

type hostsLookupRequest struct {
	AccessPointSGTIN      string                `json:"id"`
	ClientCharacteristics clientCharacteristics `json:"clientCharacteristics"`
}

type hostsLookupResponse struct {
	RestEndpoint      string `json:"urlREST"`
	WebSocketEndpoint string `json:"urlWebSocket"`
}

type registerClientRequest struct {
	DeviceID         string `json:"deviceId"`
	DeviceName       string `json:"deviceName"`
	AccessPointSGTIN string `json:"sgtin"`
	AuthToken        string `json:"authToken"`
}

type getAuthTokenResponse struct {
	AuthToken string `json:"authToken"`
}

type confirmAuthTokenResponse struct {
	ClientID string `json:"clientId"`
}

type clientCharacteristics struct {
	APIVersion         string `json:"apiVersion"`
	ClientName         string `json:"applicationIdentifier"`
	ClientVersion      string `json:"applicationVersion"`
	DeviceManufacturer string `json:"deviceManufacturer"`
	DeviceType         string `json:"deviceType"`
	Language           string `json:"language"`
	OSType             string `json:"osType"`
	OSVersion          string `json:"osVersion"`
}
