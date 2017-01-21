// Package look defines common formatters for fields/types to be used when
// generating custom events.
package look

import (
	"time"

	"github.com/elastic/beats/libbeat/common"

	"github.com/elastic/beats/heartbeat/reason"
)

func RTT(rtt time.Duration) common.MapStr {
	return common.MapStr{
		"us": rtt / (time.Microsecond / time.Nanosecond),
	}
}

func Reason(err error) common.MapStr {
	if r, ok := err.(reason.Reason); ok {
		return reason.Fail(r)
	}
	return reason.FailIO(err)
}

func Timestamp(t time.Time) common.Time {
	return common.Time(t)
}
