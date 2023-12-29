package hmip

import "encoding/json"

type functionalChannel struct {
	Type string `json:"functionalChannelType"`
}

func (fc functionalChannel) GetType() string {
	return fc.Type
}

// ======================================================

type baseDeviceChannel struct {
	functionalChannel
	LowBattery   bool     `json:"lowBat"`
	RSSIValue    int      `json:"rssiDeviceValue"`
	Unreached    bool     `json:"unreach"`
	UnderVoltage bool     `json:"deviceUndervoltage"`
	Overheated   bool     `json:"deviceOverheated"`
	Groups       []string `json:"groups"`
}

func (bdc baseDeviceChannel) HasLowBattery() bool {
	return bdc.LowBattery
}

func (bdc baseDeviceChannel) GetRSSIValue() int {
	return bdc.RSSIValue
}

func (bdc baseDeviceChannel) IsUnreached() bool {
	return bdc.Unreached
}

func (bdc baseDeviceChannel) HasUnderVoltage() bool {
	return bdc.UnderVoltage
}

func (bdc baseDeviceChannel) IsOverheated() bool {
	return bdc.Overheated
}

func (bdc baseDeviceChannel) GetGroups() []string {
	return bdc.Groups
}

// ======================================================

type switchChannel struct {
	functionalChannel
	switchable
}

// ======================================================

type switchMeasuringChannel struct {
	functionalChannel
	switchable
	powerConsumptionMeasuring
}

// ======================================================

type climateSensorChannel struct {
	functionalChannel
	climateMeasuring
}

// ======================================================

type smokeDetectorChannel struct {
	functionalChannel
	ChamberDegraded bool `json:"chamberDegraded"`
}

func (sdc smokeDetectorChannel) IsChamberDegraded() bool {
	return sdc.ChamberDegraded
}

// ======================================================

func (fc *FunctionalChannels) UnmarshalJSON(value []byte) error {
	var channelValues map[string]json.RawMessage
	err := json.Unmarshal(value, &channelValues)
	if err != nil {
		return err
	}
	channels := make(FunctionalChannels, 0, len(channelValues))
	for _, channelValue := range channelValues {
		var channel functionalChannel
		err = json.Unmarshal(channelValue, &channel)
		if err != nil {
			return err
		}
		switch channel.Type {
		case CHANNEL_TYPE_DEVICE_BASE:
			specialChannel := baseDeviceChannel{
				functionalChannel: channel,
			}
			err = json.Unmarshal(channelValue, &specialChannel)
			if err != nil {
				return err
			}
			channels = append(channels, &specialChannel)
		case CHANNEL_TYPE_SWITCH:
			specialChannel := switchChannel{
				functionalChannel: channel,
			}
			err = json.Unmarshal(channelValue, &specialChannel)
			if err != nil {
				return err
			}
			channels = append(channels, &specialChannel)
		case CHANNEL_TYPE_SWITCH_MEASURING:
			specialChannel := switchMeasuringChannel{
				functionalChannel: channel,
			}
			err = json.Unmarshal(channelValue, &specialChannel)
			if err != nil {
				return err
			}
			channels = append(channels, &specialChannel)
		case CHANNEL_TYPE_CLIMATE_SENSOR:
			specialChannel := climateSensorChannel{
				functionalChannel: channel,
			}
			err = json.Unmarshal(channelValue, &specialChannel)
			if err != nil {
				return err
			}
			channels = append(channels, &specialChannel)
		case CHANNEL_TYPE_SMOKE_DETECTOR:
			specialChannel := smokeDetectorChannel{
				functionalChannel: channel,
			}
			err = json.Unmarshal(channelValue, &specialChannel)
			if err != nil {
				return err
			}
			channels = append(channels, &specialChannel)
		default:
			channels = append(channels, &channel)
		}
	}
	*fc = channels
	return nil
}
