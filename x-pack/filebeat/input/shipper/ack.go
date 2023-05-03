package shipper

import "sync"

type shipperAckTracker struct {
	mutex sync.Mutex
	// the idea here is to track the ID of events that have been acked. In the old shipper,
	// we had a direct interface with the queue, and could grab the event index when we called Publish().
	// This is meant to do something similar.

	// currently, in the queue, the PeristedIndex is the oldest unacknowledged event,
	// so this shoould probably return the oldest event that hasn't been acked.
	eventIndex int64
}
