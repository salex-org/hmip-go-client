package hmip

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/avast/retry-go/v4"
	"golang.org/x/net/websocket"
	"io"
	"net/http"
	"net/url"
	"os"
	"slices"
	"time"
)

type homematic struct {
	config              *Config
	httpClient          *http.Client
	registrations       []handlerRegistration
	eventLoopRunning    bool
	eventLoopError      error
	eventLog            io.Writer
	websocketConfig     *websocket.Config
	websocketConnection *websocket.Conn
}

func GetClient() (Homematic, error) {
	config, err := GetConfig()
	if err != nil {
		return nil, err
	} else {
		return GetClientWithConfig(config)
	}
}

func GetClientWithConfig(config *Config) (Homematic, error) {
	err := config.lookupEndpoints()
	client := &homematic{
		config: config,
		httpClient: &http.Client{
			Transport: &homematicRoundTripper{
				Origin: http.DefaultTransport,
				config: config,
			},
			Timeout: 30 * time.Second,
		},
		eventLoopRunning: false,
		eventLog:         os.Stdout,
	}
	if err == nil {
		// Initialize websocket configuration
		client.websocketConfig, err = websocket.NewConfig(config.WebSocketEndpoint, "wss://localhost")
		if client.websocketConfig != nil {
			client.websocketConfig.Header.Set("AUTHTOKEN", config.AuthToken)
			client.websocketConfig.Header.Set("CLIENTAUTH", config.ClientAuthToken)
		}
	}
	return client, err
}

func (c *homematic) LoadCurrentState() (State, error) {
	requestBody, _ := json.Marshal(getStateRequest{
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
	state := state{}
	err = json.Unmarshal(responseBody, &state)
	if err != nil {
		return nil, err
	}
	return &state, nil
}

func (c *homematic) RegisterEventHandler(handler EventHandler, eventTypes ...string) {
	c.registrations = append(c.registrations, handlerRegistration{
		Handler: handler,
		Types:   eventTypes,
	})
}

func (c *homematic) SetEventLog(writer io.Writer) {
	c.eventLog = writer
}

func (c *homematic) ListenForEvents() error {
	if c.eventLoopRunning {
		return errors.New("Event loop already running")
	}
	c.eventLoopRunning = true
	return retry.Do(c.eventLoop, retry.DelayType(func(n uint, loopErr error, config *retry.Config) time.Duration {
		c.eventLoopError = loopErr
		_, _ = fmt.Fprintf(c.eventLog, "Error in event loop: %v\nTry to lookup hosts again\n", loopErr)
		err := c.config.lookupEndpoints()
		if err == nil {
			if c.websocketConfig.Location.String() != c.config.WebSocketEndpoint {
				c.websocketConfig.Location, err = url.ParseRequestURI(c.config.WebSocketEndpoint)
				if err == nil {
					_, _ = fmt.Fprintf(c.eventLog, "Switching websocket endpoint to %s\nRestarting event loop\n", c.config.WebSocketEndpoint)
					return 0
				}
			}
		}
		if err != nil {
			c.eventLoopError = err
			_, _ = fmt.Fprintf(c.eventLog, "Error during host lookup: %v\n", err)
		}
		_, _ = fmt.Fprintf(c.eventLog, "Restarting event loop in 10 minutes\n")
		return time.Minute * 10
	}), retry.Attempts(0))
}

func (c *homematic) eventLoop() error {
	c.eventLoopError = nil // Reset error cache
	var err error
	c.websocketConnection, err = websocket.DialConfig(c.websocketConfig)
	if err != nil {
		return err
	}
	defer func(conn *websocket.Conn) {
		_, _ = fmt.Fprintf(c.eventLog, "\U0001F6AB Closing connection to %s\n", conn.RemoteAddr().String())
		_ = conn.Close()
	}(c.websocketConnection)
	_, _ = fmt.Fprintf(c.eventLog, "\U0001F50C Established connection to %v\n", c.websocketConnection.RemoteAddr())
	for {
		message := pushMessage{}
		err := websocket.JSON.Receive(c.websocketConnection, &message)
		if err != nil {
			if c.eventLoopRunning {
				return err
			} else {
				return nil // Error occurred because of terminating the event loop - returning without error
			}
		}
		for _, event := range message.Events {
			for _, registration := range c.registrations {
				if len(registration.Types) == 0 || slices.Contains(registration.Types, event.GetType()) {
					registration.Handler(event, message.Origin)
				}
			}
		}
	}
}

func (c *homematic) GetEventLoopState() error {
	return c.eventLoopError
}

func (c *homematic) StopEventListening() error {
	if c.eventLoopRunning {
		c.eventLoopRunning = false
		if c.websocketConnection != nil {
			return c.websocketConnection.Close()
		}
	}
	return nil
}

type homematicRoundTripper struct {
	Origin http.RoundTripper
	config *Config
}

func (r *homematicRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	request.Header.Set("Content-Type", "application/json")
	request.Header["VERSION"] = []string{ApiVersion}
	request.Header["CLIENTAUTH"] = []string{r.config.ClientAuthToken}
	request.Header["AUTHTOKEN"] = []string{r.config.AuthToken}
	return r.Origin.RoundTrip(request)
}

type getStateRequest struct {
	ClientCharacteristics clientCharacteristics `json:"clientCharacteristics"`
}

type pushMessage struct {
	Events Events `json:"events"`
	Origin origin `json:"origin"`
}

type handlerRegistration struct {
	Handler EventHandler
	Types   []string
}
