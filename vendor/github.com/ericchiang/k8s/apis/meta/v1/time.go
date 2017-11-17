package v1

import (
	"encoding/json"
	"time"
)

// JSON marshaling logic for the Time type. Need to make
// third party resources JSON work.

func (t Time) MarshalJSON() ([]byte, error) {
	var seconds, nanos int64
	if t.Seconds != nil {
		seconds = *t.Seconds
	}
	if t.Nanos != nil {
		nanos = int64(*t.Nanos)
	}
	return json.Marshal(time.Unix(seconds, nanos))
}

func (t *Time) UnmarshalJSON(p []byte) error {
	var t1 time.Time
	if err := json.Unmarshal(p, &t1); err != nil {
		return err
	}
	seconds := t1.Unix()
	nanos := int32(t1.UnixNano())
	t.Seconds = &seconds
	t.Nanos = &nanos
	return nil
}
