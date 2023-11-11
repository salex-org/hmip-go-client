package hmip

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/avast/retry-go/v4"
	"io"
	"net/http"
	"time"
)

type Client interface {
	LoadCurrentState() (*State, error)
}

type clientImpl struct {
	config     *Config
	httpClient *http.Client
}

func GetClient() (Client, error) {
	config, err := GetConfig()
	if err != nil {
		return nil, err
	} else {
		return GetClientWithConfig(config)
	}
}

func GetClientWithConfig(config *Config) (Client, error) {
	_ = config.lookupEndpoints()
	client := &clientImpl{
		config: config,
		httpClient: &http.Client{
			Transport: &HomematicRoundTripper{
				Origin: http.DefaultTransport,
				config: config,
			},
			Timeout: 30 * time.Second,
		},
	}
	return client, nil
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

func (c *clientImpl) LoadCurrentState() (*State, error) {
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
