package report

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
)

// Event is the format of monitoring events.
// A separate event is required as it has to be serialized differently.
// The only difference between report.Event and beat.Event
// is Timestamp is serialized as "timestamp" in monitoring.
type Event struct {
	Timestamp time.Time     `struct:"timestamp"`
	Fields    common.MapStr `struct:",inline"`
}
