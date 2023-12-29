package hmip

import "time"

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
