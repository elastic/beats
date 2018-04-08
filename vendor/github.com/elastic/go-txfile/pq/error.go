package pq

import "errors"

var (
	errNODelegate          = errors.New("delegate must not be nil")
	errInvalidPagesize     = errors.New("invalid page size")
	errClosed              = errors.New("queue closed")
	errNoQueueRoot         = errors.New("no queue root")
	errIncompleteQueueRoot = errors.New("incomplete queue root")
	errInvalidVersion      = errors.New("invalid queue version")
	errACKEmptyQueue       = errors.New("ack on empty queue")
	errACKTooManyEvents    = errors.New("too many events have been acked")
	errSeekPageFailed      = errors.New("failed to seek to next page")
)
