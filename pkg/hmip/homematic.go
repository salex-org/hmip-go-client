package hmip

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type homematic struct {
	config              *Config
	httpClient          *http.Client
	registrations       []handlerRegistration
	eventLoopMutex      sync.Mutex
	eventLoopContext    context.Context
	eventLoopCancelFunc context.CancelFunc
	eventLoopError      error
	eventLog            io.Writer
	websocketEndpoint   string
	websocketDialer     *websocket.Dialer
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
		eventLog: os.Stdout,
	}
	if err == nil {
		// Initialize websocket configuration
		client.websocketEndpoint = config.WebSocketEndpoint
	}
	return client, err
}

func (c *homematic) LoadCurrentState() (State, error) {
	requestBody, _ := json.Marshal(getStateRequest{
		ClientCharacteristics: c.config.getClientCharacteristics(),
	})
	var lastErr error
	const attempts = 2
	for i := 0; i < attempts; i++ {
		request, err := http.NewRequest(
			http.MethodPost,
			c.config.RestEndpoint+"/hmip/home/getCurrentState",
			bytes.NewReader(requestBody),
		)
		if err != nil {
			return nil, err
		}

		response, err := c.httpClient.Do(request)
		if err != nil {
			lastErr = err
		} else {
			defer response.Body.Close()

			if response.StatusCode != http.StatusOK {
				lastErr = fmt.Errorf("error on reading state (%s)", response.Status)
			} else {
				responseBody, err := io.ReadAll(response.Body)
				if err != nil {
					return nil, err
				}

				state := state{}
				if err := json.Unmarshal(responseBody, &state); err != nil {
					return nil, err
				}
				return &state, nil
			}
		}

		// Vor dem nächsten Versuch neuen Lookup machen
		_ = c.config.lookupEndpoints()
	}

	return nil, lastErr
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
	c.eventLoopMutex.Lock()
	if c.eventLoopContext != nil {
		c.eventLoopMutex.Unlock()
		return fmt.Errorf("Event loop already running")
	}
	c.eventLoopContext, c.eventLoopCancelFunc = context.WithCancel(context.Background())
	c.eventLoopError = nil
	c.eventLoopMutex.Unlock()

	for {
		if err := c.eventLoop(); err != nil {
			c.eventLoopMutex.Lock()
			c.eventLoopError = err
			c.eventLoopMutex.Unlock()

			// "Echter" Fehler nur, wenn der EventLoopContext nich beendet wurde
			select {
			case <-c.eventLoopContext.Done():
			default:
				_, _ = fmt.Fprintf(c.eventLog, "Error in event loop: %v\nTry to lookup hosts again\n", err)
				err := c.config.lookupEndpoints()
				if err == nil {
					if c.websocketEndpoint != c.config.WebSocketEndpoint {
						c.websocketEndpoint = c.config.WebSocketEndpoint
						_, _ = fmt.Fprintf(c.eventLog, "Switching websocket endpoint to %s\nRestarting event loop\n", c.config.WebSocketEndpoint)
						continue
					}
				}
				if err != nil {
					c.eventLoopError = err
					_, _ = fmt.Fprintf(c.eventLog, "Error during host lookup: %v\n", err)
				}
				_, _ = fmt.Fprintf(c.eventLog, "Restarting event loop in 10 minutes\n")
			}

			timer := time.NewTimer(10 * time.Minute)
			select {
			case <-timer.C:

				c.eventLoopMutex.Lock()
				c.eventLoopError = nil
				c.eventLoopMutex.Unlock()

				continue
			case <-c.eventLoopContext.Done():
				if !timer.Stop() {
					<-timer.C
				}

				c.eventLoopMutex.Lock()
				c.eventLoopError = nil
				c.eventLoopCancelFunc = nil
				c.eventLoopContext = nil
				c.eventLoopMutex.Unlock()

				return nil
			}
		}
	}
}

func (c *homematic) eventLoop() error {
	c.eventLoopError = nil // Reset error cache
	var err error
	websocketHeader := http.Header{}
	websocketHeader.Set("AUTHTOKEN", c.config.AuthToken)
	websocketHeader.Set("CLIENTAUTH", c.config.ClientAuthToken)
	c.websocketConnection, _, err = c.websocketDialer.Dial(c.websocketEndpoint, websocketHeader)
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
		err := c.websocketConnection.ReadJSON(&message)
		if err != nil {
			return err
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
	c.eventLoopMutex.Lock()
	defer c.eventLoopMutex.Unlock()

	if c.eventLoopCancelFunc != nil {
		c.eventLoopCancelFunc()
		if c.websocketConnection != nil {
			err := c.websocketConnection.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				return err
			}

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
