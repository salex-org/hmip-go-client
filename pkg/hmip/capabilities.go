package hmip

import "time"

type named struct {
	Name string `json:"label"`
}

func (n named) GetName() string {
	return n.Name
}

// ======================================================

type stateful struct {
	ID          string             `json:"id"`
	LastUpdated HomematicTimestamp `json:"lastStatusUpdate"`
}

func (s stateful) GetID() string {
	return s.ID
}
func (s stateful) GetLastUpdated() time.Time {
	return s.LastUpdated.Time
}

// ======================================================

type typed struct {
	Type string `json:"type"`
}

func (t typed) GetType() string {
	return t.Type
}

// ======================================================

type switchable struct {
	SwitchedOn bool `json:"on"`
}

func (s switchable) IsSwitchedOn() bool {
	return s.SwitchedOn
}

// ======================================================

type powerConsumptionMeasuring struct {
	CurrentPowerConsumption float64 `json:"currentPowerConsumption"`
}

func (pcm powerConsumptionMeasuring) GetCurrentPowerConsumption() float64 {
	return pcm.CurrentPowerConsumption
}

// ======================================================

type climateMeasuring struct {
	ActualTemperature float64 `json:"actualTemperature"`
	Humidity          int     `json:"humidity"`
	VapourAmount      float64 `json:"vaporAmount"`
}

func (cm climateMeasuring) GetActualTemperature() float64 {
	return cm.ActualTemperature
}

func (cm climateMeasuring) GetHumidity() int {
	return cm.Humidity
}

func (cm climateMeasuring) GetVapourAmount() float64 {
	return cm.VapourAmount
}
