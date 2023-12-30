package hmip

type state struct {
	Devices Devices `json:"devices"`
	Groups  Groups  `json:"groups"`
	Clients Clients `json:"clients"`
}

func (s state) GetDevices() Devices {
	return s.Devices
}

func (s state) GetGroups() Groups {
	return s.Groups
}

func (s state) GetClients() Clients {
	return s.Clients
}

func (s state) GetDevicesByType(deviceType string) Devices {
	var devices Devices
	for _, device := range s.Devices {
		if device.GetType() == deviceType {
			devices = append(devices, device)
		}
	}
	return devices
}

func (s state) GetGroupsByType(groupType string) Groups {
	var groups Groups
	for _, group := range s.Groups {
		if group.GetType() == groupType {
			groups = append(groups, group)
		}
	}
	return groups
}

func (s state) GetDeviceByID(deviceID string) Device {
	for _, device := range s.Devices {
		if device.GetID() == deviceID {
			return device
		}
	}
	return nil
}

func (s state) GetGroupByID(groupID string) Group {
	for _, group := range s.Groups {
		if group.GetID() == groupID {
			return group
		}
	}
	return nil
}

func (s state) GetFunctionalChannelsByType(deviceType, channelType string) FunctionalChannels {
	var channels FunctionalChannels
	for _, device := range s.GetDevicesByType(deviceType) {
		for _, channel := range device.GetFunctionalChannelsByType(channelType) {
			channels = append(channels, channel)
		}
	}
	return channels
}
