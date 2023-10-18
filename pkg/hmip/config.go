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
	"runtime"
	"strings"
	"time"
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

func (c *Config) createClientAuthToken() {
	tokenDigest := digest.SHA512.FromBytes([]byte(c.getTrimmedAccessPointSGTIN() + "jiLpVitHvWnIGD1yo7MA"))
	c.ClientAuthToken = strings.ToUpper(tokenDigest.Hex())
}

func (c *Config) createDeviceID() {
	c.DeviceID = uuid.New().String()
}

func (c *Config) connectionRequest(httpClient *http.Client) error {
	requestBody, _ := json.Marshal(RegisterClientRequest{
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
	result := HostsLookupResponse{}
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
