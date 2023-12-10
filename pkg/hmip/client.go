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

type Client interface {
	LoadCurrentState() (*State, error)
	RegisterEventHandler(handler EventHandler, eventTypes ...string)
	SetEventLog(writer io.Writer)
	ListenForEvents() error
	StopEventListening() error
}

type clientImpl struct {
	config              *Config
	httpClient          *http.Client
	registrations       []HandlerRegistration
	eventLoopRunning    bool
	eventLog            io.Writer
	websocketConfig     *websocket.Config
	websocketConnection *websocket.Conn
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
	err := config.lookupEndpoints()
	client := &clientImpl{
		config: config,
		httpClient: &http.Client{
			Transport: &HomematicRoundTripper{
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

func (c *clientImpl) RegisterEventHandler(handler EventHandler, eventTypes ...string) {
	c.registrations = append(c.registrations, HandlerRegistration{
		Handler: handler,
		Types:   eventTypes,
	})
}

func (c *clientImpl) SetEventLog(writer io.Writer) {
	c.eventLog = writer
}

func (c *clientImpl) ListenForEvents() error {
	if c.eventLoopRunning {
		return errors.New("Event loop already running")
	}
	c.eventLoopRunning = true
	return retry.Do(c.eventLoop, retry.DelayType(func(n uint, loopErr error, config *retry.Config) time.Duration {
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
			_, _ = fmt.Fprintf(c.eventLog, "Error during host lookup: %v\n", err)
		}
		_, _ = fmt.Fprintf(c.eventLog, "Restarting event loop in 10 minutes\n")
		return time.Minute * 10
	}), retry.Attempts(0))
}

func (c *clientImpl) eventLoop() error {
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
		message := PushMessage{}
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
				if len(registration.Types) == 0 || slices.Contains(registration.Types, event.Type) {
					registration.Handler(event, message.Origin)
				}
			}
		}
	}
}

func (c *clientImpl) StopEventListening() error {
	if c.eventLoopRunning {
		c.eventLoopRunning = false
		if c.websocketConnection != nil {
			return c.websocketConnection.Close()
		}
	}
	return nil
}

func (e *Event) GetFunctionalChannels(deviceType, channelType string) []FunctionalChannel {
	var channels []FunctionalChannel
	if e.Device != nil {
		if e.Device.Type == deviceType {
			for _, channel := range e.Device.Channels {
				if channel.Type == channelType {
					channels = append(channels, channel)
				}
			}
		}
	}
	return channels
}

func (s State) GetFunctionalChannelsByType(deviceType, channelType string) []FunctionalChannel {
	var channels []FunctionalChannel
	for _, device := range s.GetDevicesByType(deviceType) {
		for _, channel := range device.GetFunctionalChannelsByType(channelType) {
			channels = append(channels, channel)
		}
	}
	return channels
}

func (d Device) GetFunctionalChannelsByType(channelType string) []FunctionalChannel {
	var channels []FunctionalChannel
	for _, channel := range d.Channels {
		if channel.Type == channelType {
			channels = append(channels, channel)
		}
	}
	return channels
}

func (s State) GetDevicesByType(deviceType string) []Device {
	var devices []Device
	for _, device := range s.Devices {
		if device.Type == deviceType {
			devices = append(devices, device)
		}
	}
	return devices
}

func (s State) GetGroupsByType(groupType string) []Group {
	var groups []Group
	for _, group := range s.Groups {
		if group.Type == groupType {
			groups = append(groups, group)
		}
	}
	return groups
}

func (s State) GetDeviceByID(deviceID string) *Device {
	for _, device := range s.Devices {
		if device.ID == deviceID {
			return &device
		}
	}
	return nil
}

func (s State) GetGroupByID(groupID string) *Group {
	for _, group := range s.Groups {
		if group.ID == groupID {
			return &group
		}
	}
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
