package hmip

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
