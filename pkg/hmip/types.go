package hmip

import (
	"strconv"
	"time"
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
	DeviceName       string `json:"deviceName"`
	AccessPointSGTIN string `json:"sgtin"`
	AuthToken        string `json:"authToken"`
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
	Type                    string  `json:"functionalChannelType"`
	Temperature             float64 `json:"actualTemperature"`
	Humidity                int     `json:"humidity"`
	VapourAmount            float64 `json:"vaporAmount"`
	SwitchedOn              bool    `json:"on"`
	CurrentPowerConsumption float64 `json:"currentPowerConsumption"`
}

type ClientRegistration struct {
	ID       string             `json:"id"`
	Name     string             `json:"label"`
	Created  HomematicTimestamp `json:"createdAtTimestamp"`
	LastSeen HomematicTimestamp `json:"lastSeenAtTimestamp"`
}

type HomematicTimestamp struct {
	time.Time
}

func (t *HomematicTimestamp) MarshalJSON() ([]byte, error) {
	s := strconv.Itoa(int(t.Time.UnixMilli()))
	return []byte(s), nil
}

func (t *HomematicTimestamp) UnmarshalJSON(value []byte) error {
	unix, err := strconv.Atoi(string(value))
	if err != nil {
		return err
	}
	t.Time = time.UnixMilli(int64(unix))
	return nil
}
