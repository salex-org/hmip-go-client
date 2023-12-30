package hmip

import (
	"encoding/json"
	"time"
)

type client struct {
	named
	typed
	ID       string             `json:"id"`
	Created  HomematicTimestamp `json:"createdAtTimestamp"`
	LastSeen HomematicTimestamp `json:"lastSeenAtTimestamp"`
}

func (c client) GetID() string {
	return c.ID
}

func (c client) GetLastSeen() time.Time {
	return c.LastSeen.Time
}

func (c client) GetCreated() time.Time {
	return c.Created.Time
}

// ======================================================

func (c *Clients) UnmarshalJSON(value []byte) error {
	var clientValues map[string]json.RawMessage
	err := json.Unmarshal(value, &clientValues)
	if err != nil {
		return err
	}
	clients := make(Clients, 0, len(clientValues))
	for _, clientValue := range clientValues {
		var client client
		err = json.Unmarshal(clientValue, &client)
		if err != nil {
			return err
		}
		clients = append(clients, &client)
	}
	*c = clients
	return nil
}
