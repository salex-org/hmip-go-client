package hmip

import (
	"strconv"
	"time"
)

const (
	EVENT_TYPE_DEVICE_CHANGED = "DEVICE_CHANGED"
	EVENT_TYPE_GROUP_CHANGED  = "GROUP_CHANGED"
	EVENT_TYPE_HOME_CHANGED   = "HOME_CHANGED"

	CHANNEL_TYPE_DEVICE_BASE       = "DEVICE_BASE"
	CHANNEL_TYPE_SWITCH            = "SWITCH_CHANNEL"
	CHANNEL_TYPE_SWITCH_MEASURING  = "SWITCH_MEASURING_CHANNEL"
	CHANNEL_TYPE_CLIMATE_SENSOR    = "CLIMATE_SENSOR_CHANNEL"
	CHANNEL_TYPE_ACCESS_CONTROLLER = "ACCESS_CONTROLLER_CHANNEL"
	CHANNEL_TYPE_SMOKE_DETECTOR    = "SMOKE_DETECTOR_CHANNEL"

	DEVICE_TYPE_TEMPERATURE_HUMIDITY_SENSOR_OUTDOOR = "TEMPERATURE_HUMIDITY_SENSOR_OUTDOOR"
	DEVICE_TYPE_PLUGABLE_SWITCH                     = "PLUGABLE_SWITCH"
	DEVICE_TYPE_PLUGABLE_SWITCH_MEASURING           = "PLUGABLE_SWITCH_MEASURING"
	DEVICE_TYPE_HOME_CONTROL_ACCESS_POINT           = "HOME_CONTROL_ACCESS_POINT"
	DEVICE_TYPE_SMOKE_DETECTOR                      = "SMOKE_DETECTOR"

	CONNECTION_TYPE_RF  = "HMIP_RF"
	CONNECTION_TYPE_LAN = "HMIP_LAN"

	GROUP_TYPE_META        = "META"
	GROUP_TYPE_ENVIRONMENT = "ENVIRONMENT"

	ORIGIN_TYPE_DEVICE = "DEVICE"
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
	Groups  map[string]Group              `json:"groups"`
	Clients map[string]ClientRegistration `json:"clients"`
}

type Group struct {
	ID   string `json:"id"`
	Name string `json:"label"`
	Type string `json:"type"`
}

type Device struct {
	ID                   string                       `json:"id"`
	Name                 string                       `json:"label"`
	Type                 string                       `json:"type"`
	Model                string                       `json:"modelType"`
	SGTIN                string                       `json:"serializedGlobalTradeItemNumber"`
	Channels             map[string]FunctionalChannel `json:"functionalChannels"`
	LastStatusUpdate     HomematicTimestamp           `json:"lastStatusUpdate"`
	PermanentlyReachable bool                         `json:"permanentlyReachable"`
	ConnectionType       string                       `json:"connectionType"`
}

type FunctionalChannel struct {
	Type                    string  `json:"functionalChannelType"`
	Temperature             float64 `json:"actualTemperature"`
	Humidity                int     `json:"humidity"`
	VapourAmount            float64 `json:"vaporAmount"`
	SwitchedOn              bool    `json:"on"`
	CurrentPowerConsumption float64 `json:"currentPowerConsumption"`
	LowBattery              bool    `json:"lowBat"`
	RSSIValue               int     `json:"rssiDeviceValue"`
	Unreached               bool    `json:"unreach"`
	Undervoltage            bool    `json:"deviceUndervoltage"`
	Overheated              bool    `json:"deviceOverheated"`
	ChamberDegraded         bool    `json:"chamberDegraded"`
}

type ClientRegistration struct {
	ID       string             `json:"id"`
	Name     string             `json:"label"`
	Created  HomematicTimestamp `json:"createdAtTimestamp"`
	LastSeen HomematicTimestamp `json:"lastSeenAtTimestamp"`
}

type PushMessage struct {
	Events map[string]Event `json:"events"`
	Origin Origin           `json:"origin"`
}

type Origin struct {
	Type string `json:"originType"`
	ID   string `json:"id"`
}

type Event struct {
	Type   string  `json:"pushEventType"`
	Device *Device `json:"device,omitempty"`
	Group  *Group  `json:"group,omitempty"`
}

type HandlerRegistration struct {
	Handler EventHandler
	Types   []string
}

type EventHandler func(event Event, origin Origin)

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
