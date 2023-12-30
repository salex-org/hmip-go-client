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

// Homematic is the base interface to access the HomemticIP Cloud.
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
// with all devices, groups and clients.
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

// Device represents the current state of a device.
// Specific information is stored in the functional channels.
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

// Group represents the current state of a group.
type Group interface {
	Stateful
	Named
	Typed
}
type Groups []Group

// MetaGroup represents the current state of a group with GROUP_TYPE_META.
type MetaGroup interface {
	Group
	GetIcon() string
}

// ======================================================

// Client represents a registered client.
type Client interface {
	Typed
	Named
	GetID() string
	GetLastSeen() time.Time
	GetCreated() time.Time
}
type Clients []Client

// ======================================================

// FunctionalChannel represents a functional channel as
// part of a device. It is extended in special sub interfaces.
type FunctionalChannel interface {
	Typed
}
type FunctionalChannels []FunctionalChannel

// BaseDeviceChannel is a special functional channel for type CHANNEL_TYPE_DEVICE_BASE
// containing base device information available in all kinds of devices.
type BaseDeviceChannel interface {
	FunctionalChannel
	HasLowBattery() bool
	GetRSSIValue() int
	IsUnreached() bool
	HasUnderVoltage() bool
	IsOverheated() bool
	GetGroups() []string
}

// SwitchChannel is a special functional channel for type CHANNEL_TYPE_SWITCH
// containing the switch state of switching devices.
type SwitchChannel interface {
	FunctionalChannel
	Switchable
}

// SwitchMeasuringChannel is a special functional channel for type CHANNEL_TYPE_SWITCH_MEASURING
// containing the switch state and the current power consumption of switching and measuring devices.
type SwitchMeasuringChannel interface {
	FunctionalChannel
	Switchable
	PowerConsumptionMeasuring
}

// ClimateSensorChannel is a special functional channel for type CHANNEL_TYPE_CLIMATE_SENSOR
// containing the measuring data of climate measuring devices.
type ClimateSensorChannel interface {
	FunctionalChannel
	ClimateMeasuring
}

// SmokeDetectorChannel is a special functional channel for type CHANNEL_TYPE_SMOKE_DETECTOR
// containing information regarding the chamber state of smoke detecting devices.
type SmokeDetectorChannel interface {
	FunctionalChannel
	IsChamberDegraded() bool
}

// ======================================================

// Stateful is a capability implemented by all interfaces representing data
// which has a status that gets updated periodically.
type Stateful interface {
	GetID() string
	GetLastUpdated() time.Time
}

// Named is a capability implemented by all interfaces representing data
// which has a name.
type Named interface {
	GetName() string
}

// Typed is a capability implemented by all interfaces representing data
// which has a type.
type Typed interface {
	GetType() string
}

// Switchable is a capability implemented by all interfaces representing data
// which contains a switch status.
type Switchable interface {
	IsSwitchedOn() bool
}

// PowerConsumptionMeasuring is a capability implemented by all interfaces
// representing data which contains a current power consumption.
type PowerConsumptionMeasuring interface {
	GetCurrentPowerConsumption() float64
}

// ClimateMeasuring is a capability implemented by all interfaces representing data
// which contains climate measuring information.
type ClimateMeasuring interface {
	GetActualTemperature() float64
	GetHumidity() int
	GetVapourAmount() float64
}

// ======================================================

// Event represents an event received by a WebSocket connection.
// It is extended in special sub interfaces.
type Event interface {
	Typed
}
type Events []Event

// DeviceChangedEvent is a special functional channel for type EVENT_TYPE_DEVICE_CHANGED
// containing the updated status of a Device.
type DeviceChangedEvent interface {
	Event
	GetDevice() Device
	GetFunctionalChannels(deviceType, channelType string) FunctionalChannels
}

// GroupChangedEvent is a special functional channel for type EVENT_TYPE_GROUP_CHANGED
// containing the updated status of a Group.
type GroupChangedEvent interface {
	Event
	GetGroup() Group
}

// Origin represents the origin of an event received by a WebSocket connection.
type Origin interface {
	Typed
	GetID() string
}

// EventHandler represents and function type used to register a WebSocket event
// handler (see Homematic.RegisterEventHandler)
type EventHandler func(event Event, origin Origin)
