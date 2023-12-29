package hmip

type device struct {
	stateful
	named
	typed
	Model                string             `json:"modelType"`
	SGTIN                string             `json:"serializedGlobalTradeItemNumber"`
	FunctionalChannels   FunctionalChannels `json:"functionalChannels"`
	PermanentlyReachable bool               `json:"permanentlyReachable"`
	ConnectionType       string             `json:"connectionType"`
}

func (d device) GetModel() string {
	return d.Model
}

func (d device) GetSGTIN() string {
	return d.SGTIN
}

func (d device) IsPermanentlyReachable() bool {
	return d.PermanentlyReachable
}

func (d device) GetConnectionType() string {
	return d.ConnectionType
}

func (d device) GetFunctionalChannels() FunctionalChannels {
	var channels FunctionalChannels
	for _, channel := range d.FunctionalChannels {
		channels = append(channels, channel)
	}
	return channels
}

func (d device) GetFunctionalChannelsByType(channelType string) FunctionalChannels {
	var channels FunctionalChannels
	for _, channel := range d.FunctionalChannels {
		if channel.GetType() == channelType {
			channels = append(channels, channel)
		}
	}
	return channels
}
