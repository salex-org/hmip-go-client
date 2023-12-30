package hmip

import (
	"strconv"
	"time"
)

type HomematicTimestamp struct {
	time.Time
}

func (t *HomematicTimestamp) MarshalJSON() ([]byte, error) {
	s := strconv.Itoa(int(t.Time.UnixMilli()))
	return []byte(s), nil
}

func (t *HomematicTimestamp) UnmarshalJSON(value []byte) error {
	unix, err := strconv.Atoi(string(value))
	if err != nil {
		return err
	}
	t.Time = time.UnixMilli(int64(unix))
	return nil
}
