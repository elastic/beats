package mode

import "time"

// FailOverConnectionMode connects to at most one host by random and swap to
// another host (by random) if currently active host becomes unavailable. If no
// connection is available, the mode blocks until a new connection can be established.
type FailOverConnectionMode struct {
	*SingleConnectionMode
}

// NewFailOverConnectionMode creates a new failover connection mode leveraging
// only one connection at once. If connection becomes unavailable, the mode will
// try to connect to another configured connection.
func NewFailOverConnectionMode(
	clients []ProtocolClient,
	maxAttempts int,
	waitRetry, timeout, maxWaitRetry time.Duration,
) (*FailOverConnectionMode, error) {
	mode, err := NewSingleConnectionMode(NewFailoverClient(clients),
		maxAttempts, waitRetry, timeout, maxWaitRetry)
	if err != nil {
		return nil, err
	}

	return &FailOverConnectionMode{mode}, nil
}
