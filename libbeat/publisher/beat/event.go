package beat

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
)

// Event is the common event format shared by all beats.
// Every event must have a timestamp and provide encodable Fields in `Fields`.
// The `Meta`-fields can be used to pass additional meta-data to the outputs.
// Output can optionally publish a subset of Meta, or ignore Meta.
type Event struct {
	Timestamp time.Time
	Meta      common.MapStr
	Fields    common.MapStr
}

func (e *Event) GetValue(key string) (interface{}, error) {
	if key == "@timestamp" {
		return e.Timestamp, nil
	}
	return e.Fields.GetValue(key)
}
