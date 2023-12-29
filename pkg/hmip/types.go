package hmip

import (
	"io"
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

// ======================================================

// Homematic is the base interface to access the HomemticIP Cloud
type Homematic interface {
	LoadCurrentState() (State, error)
	RegisterEventHandler(handler EventHandler, eventTypes ...string)
	SetEventLog(writer io.Writer)
	ListenForEvents() error
	StopEventListening() error
	GetEventLoopState() error
}

// ======================================================

// State represents the current state of the HomematicIP Cloud
// with all devices, groups and clients
type State interface {
	GetDevices() Devices
	GetGroups() Groups
	GetClients() Clients
	GetDevicesByType(deviceType string) Devices
	GetGroupsByType(groupType string) Groups
	GetDeviceByID(deviceID string) Device
	GetGroupByID(groupID string) Group
	GetFunctionalChannelsByType(deviceType, channelType string) FunctionalChannels
}

// ======================================================

type Device interface {
	Stateful
	Named
	Typed
	GetModel() string
	GetSGTIN() string
	IsPermanentlyReachable() bool
	GetConnectionType() string
	GetFunctionalChannels() FunctionalChannels
	GetFunctionalChannelsByType(channelType string) FunctionalChannels
}
type Devices []Device

// ======================================================

type Group interface {
	Stateful
	Named
	Typed
}
type Groups []Group

type MetaGroup interface {
	Group
	GetIcon() string
}

// ======================================================

type Client interface {
	Typed
	Named
	GetID() string
	GetLastSeen() time.Time
	GetCreated() time.Time
}
type Clients []Client

// ======================================================

type FunctionalChannel interface {
	Typed
}
type FunctionalChannels []FunctionalChannel

type BaseDeviceChannel interface {
	FunctionalChannel
	HasLowBattery() bool
	GetRSSIValue() int
	IsUnreached() bool
	HasUnderVoltage() bool
	IsOverheated() bool
	GetGroups() []string
}

type SwitchChannel interface {
	FunctionalChannel
	Switchable
}

type SwitchMeasuringChannel interface {
	FunctionalChannel
	Switchable
	PowerConsumptionMeasuring
}

type ClimateSensorChannel interface {
	FunctionalChannel
	ClimateMeasuring
}

type SmokeDetectorChannel interface {
	FunctionalChannel
	IsChamberDegraded() bool
}

// ======================================================

type Stateful interface {
	GetID() string
	GetLastUpdated() time.Time
}

type Named interface {
	GetName() string
}

type Typed interface {
	GetType() string
}

type Switchable interface {
	IsSwitchedOn() bool
}

type PowerConsumptionMeasuring interface {
	GetCurrentPowerConsumption() float64
}

type ClimateMeasuring interface {
	GetActualTemperature() float64
	GetHumidity() int
	GetVapourAmount() float64
}

// ======================================================

type Event interface {
	Typed
}

type Events []Event

type DeviceChangedEvent interface {
	Event
	GetDevice() Device
	GetFunctionalChannels(deviceType, channelType string) FunctionalChannels
}
type GroupChangedEvent interface {
	Event
	GetGroup() Group
}

type Origin interface {
	Typed
	GetID() string
}

type HandlerRegistration struct {
	Handler EventHandler
	Types   []string
}
type EventHandler func(event Event, origin Origin)
