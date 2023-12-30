package hmip

import "encoding/json"

type event struct {
	Type string `json:"pushEventType"`
}

func (e event) GetType() string {
	return e.Type
}

// ======================================================

type deviceChangedEvent struct {
	event
	Device device `json:"device"`
}

func (dce deviceChangedEvent) GetDevice() Device {
	return dce.Device
}

func (dce deviceChangedEvent) GetFunctionalChannels(deviceType, channelType string) FunctionalChannels {
	var channels FunctionalChannels
	if dce.Device.GetType() == deviceType {
		for _, channel := range dce.Device.GetFunctionalChannels() {
			if channel.GetType() == channelType {
				channels = append(channels, channel)
			}
		}
	}
	return channels
}

// ======================================================

type groupChangedEvent struct {
	event
	Group group `json:"group"`
}

func (gce groupChangedEvent) GetGroup() Group {
	return gce.Group
}

// ======================================================

func (e *Events) UnmarshalJSON(value []byte) error {
	var eventValues map[string]json.RawMessage
	err := json.Unmarshal(value, &eventValues)
	if err != nil {
		return err
	}
	events := make(Events, 0, len(eventValues))
	for _, eventValue := range eventValues {
		var event event
		err = json.Unmarshal(eventValue, &event)
		if err != nil {
			return err
		}
		switch event.Type {
		case EVENT_TYPE_DEVICE_CHANGED:
			specialEvent := deviceChangedEvent{
				event: event,
			}
			err = json.Unmarshal(eventValue, &specialEvent)
			if err != nil {
				return err
			}
			events = append(events, &specialEvent)
		case EVENT_TYPE_GROUP_CHANGED:
			specialEvent := groupChangedEvent{
				event: event,
			}
			err = json.Unmarshal(eventValue, &specialEvent)
			if err != nil {
				return err
			}
			events = append(events, &specialEvent)
		default:
			events = append(events, &event)
		}
	}
	*e = events
	return nil
}
