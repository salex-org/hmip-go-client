package hmip

type origin struct {
	Type string `json:"originType"`
	ID   string `json:"id"`
}

func (o origin) GetType() string {
	return o.Type
}

func (o origin) GetID() string {
	return o.ID
}
