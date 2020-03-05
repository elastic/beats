package events

import (
	"fmt"

	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system/socket/state"
)

// Event is the interface that all the deserialized events from the ring-buffer
// have to conform to in order to be processed by state.
type Event interface {
	fmt.Stringer
	Update(*state.State)
}
