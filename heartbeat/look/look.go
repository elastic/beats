// Package look defines common formatters for fields/types to be used when
// generating heartbeat events.
package look

import (
	"time"

	"github.com/elastic/beats/libbeat/common"

	"github.com/elastic/beats/heartbeat/reason"
)

// RTT formats a round-trip-time given as time.Duration into an
// event field. The duration is stored in `{"us": rtt}`.
func RTT(rtt time.Duration) common.MapStr {
	if rtt < 0 {
		rtt = 0
	}

	return common.MapStr{
		"us": rtt / (time.Microsecond / time.Nanosecond),
	}
}

// Reason formats an error into an error event field.
func Reason(err error) common.MapStr {
	if r, ok := err.(reason.Reason); ok {
		return reason.Fail(r)
	}
	return reason.FailIO(err)
}

// Timestamp converts an event timestamp into an compatible event timestamp for
// reporting.
func Timestamp(t time.Time) common.Time {
	return common.Time(t)
}

// Status creates a service status message from an error value.
func Status(err error) string {
	if err == nil {
		return "up"
	}
	return "down"
}
