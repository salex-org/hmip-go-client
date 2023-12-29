package hmip

type state struct {
	Devices map[string]device `json:"devices"`
	Groups  Groups            `json:"groups"`
	Clients map[string]client `json:"clients"`
}

func (s state) GetDevices() Devices {
	var devices Devices
	for _, device := range s.Devices {
		devices = append(devices, device)
	}
	return devices
}

func (s state) GetGroups() Groups {
	return s.Groups
}

func (s state) GetClients() Clients {
	var clients Clients
	for _, client := range s.Clients {
		clients = append(clients, client)
	}
	return clients
}

func (s state) GetDevicesByType(deviceType string) Devices {
	var devices Devices
	for _, device := range s.Devices {
		if device.Type == deviceType {
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
		if device.ID == deviceID {
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
