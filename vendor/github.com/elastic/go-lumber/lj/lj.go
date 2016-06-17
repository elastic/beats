// Package lj implements common lumberjack types and functions.
package lj

// Batch is an ACK-able batch of events as has been received by lumberjack
// server implemenentations. Batches must be ACKed, for the server
// implementations returning an ACK to it's clients.
type Batch struct {
	Events []interface{}
	ack    chan struct{}
}

// NewBatch creates a new ACK-able batch.
func NewBatch(evts []interface{}) *Batch {
	return &Batch{evts, make(chan struct{})}
}

// ACK acknowledges a batch initiating propagation of ACK to clients.
func (b *Batch) ACK() {
	close(b.ack)
}

// Await returns a channel for waiting for a batch to be ACKed.
func (b *Batch) Await() <-chan struct{} {
	return b.ack
}
