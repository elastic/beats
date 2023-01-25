package proxyqueue

// producerACKData tracks the number of events that need to be acknowledged
// from a single batch targeting a single producer.
type producerACKData struct {
	producer *producer
	count    int
}

// batchACKState stores the metadata associated with a batch of events sent to
// a consumer. When the consumer ACKs that batch, its doneChan is closed.
// The run loop for the broker checks the doneChan for the next sequential
// outstanding batch (to ensure ACKs are delivered in order) and calls the
// producer's ackHandler when appropriate.
type batchACKState struct {
	next     *batchACKState
	doneChan chan struct{}
	acks     []producerACKData
}

type pendingACKsList struct {
	head *batchACKState
	tail *batchACKState
}

func (l *pendingACKsList) append(ackState *batchACKState) {
	if l.head == nil {
		l.head = ackState
	} else {
		l.tail.next = ackState
	}
	l.tail = ackState
}

func (l *pendingACKsList) nextDoneChan() chan struct{} {
	if l.head != nil {
		return l.head.doneChan
	}
	return nil
}

func (l *pendingACKsList) pop() *batchACKState {
	ch := l.head
	if ch != nil {
		l.head = ch.next
		if l.head == nil {
			l.tail = nil
		}
		ch.next = nil
	}
	return ch
}

func acksForBatch(b *batch) []producerACKData {
	results := []producerACKData{}
	// We traverse the list back to front, so we can coalesce multiple events
	// into a single entry in the ACK data.
	for i := len(b.entries) - 1; i >= 0; i-- {
		entry := b.entries[i]
		if producer := entry.producer; producer != nil {
			if producer.producedCount > producer.consumedCount {
				results = append(results, producerACKData{
					producer: producer,
					count:    int(producer.producedCount - producer.consumedCount),
				})
				producer.consumedCount = producer.producedCount
			}
		}
	}
	return results
}
