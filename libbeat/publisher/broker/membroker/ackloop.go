package membroker

// ackLoop implements the brokers asynchronous ACK worker.
// Multiple concurrent ACKs from consecutive published batches will be batched up by the
// worker, to reduce the number of signals to return to the producer and the
// broker event loop.
// Producer ACKs are run in the ackLoop go-routine.
type ackLoop struct {
	broker *Broker
	sig    chan batchAckRequest
	lst    chanList

	totalACK   uint64
	totalSched uint64

	batchesSched uint64
	batchesACKed uint64
}

func (l *ackLoop) run() {
	var (
		// log = l.broker.logger

		// Buffer up acked event counter in acked. If acked > 0, acks will be set to
		// the broker.acks channel for sending the ACKs while potentially receiving
		// new batches from the broker event loop.
		// This concurrent bidirectionaly communication pattern requiring 'select'
		// ensures we can not have any deadlock between the event loop and the ack
		// loop, as the ack loop will not block on any channel
		acked int
		acks  chan int
	)

	for {
		select {
		case <-l.broker.done:
			// TODO: handle pending ACKs?
			// TODO: panic on pending batches?
			return

		case acks <- acked:
			acks, acked = nil, 0

		case lst := <-l.broker.scheduledACKs:
			count, events := lst.count()
			l.lst.concat(&lst)

			// log.Debugf("ackloop: scheduledACKs count=%v events=%v\n", count, events)
			l.batchesSched += uint64(count)
			l.totalSched += uint64(events)

		case <-l.sig:
			acked += l.handleBatchSig()
			acks = l.broker.acks
		}

		// log.Debug("ackloop INFO")
		// log.Debug("ackloop:   total events scheduled = ", l.totalSched)
		// log.Debug("ackloop:   total events ack = ", l.totalACK)
		// log.Debug("ackloop:   total batches scheduled = ", l.batchesSched)
		// log.Debug("ackloop:   total batches ack = ", l.batchesACKed)

		l.sig = l.lst.channel()
		// if l.sig == nil {
		// 	log.Debug("ackloop: no ack scheduled")
		// } else {
		// 	log.Debug("ackloop: schedule ack: ", l.lst.head.seq)
		// }
	}
}

// handleBatchSig collects and handles a batch ACK/Cancel signal. handleBatchSig
// is run by the ackLoop.
func (l *ackLoop) handleBatchSig() int {
	acks := l.lst.pop()
	l.broker.logger.Debugf("ackloop: receive ack [%v: %v, %v]", acks.seq, acks.start, acks.count)
	start := acks.start
	count := acks.count
	l.batchesACKed++
	releaseACKChan(acks)

	done := false
	// collect pending ACKs
	for !l.lst.empty() && !done {
		acks := l.lst.front()
		select {
		case <-acks.ch:
			l.broker.logger.Debugf("ackloop: receive ack [%v: %v, %v]", acks.seq, acks.start, acks.count)

			count += acks.count
			l.batchesACKed++
			releaseACKChan(l.lst.pop())

		default:
			done = true
		}
	}

	// report acks to waiting clients
	states := l.broker.buf.buf.clients
	l.broker.reportACK(states, start, count)

	// return final ACK to EventLoop, in order to clean up internal buffer
	l.broker.logger.Debug("ackloop: return ack to broker loop:", count)

	l.totalACK += uint64(count)
	l.broker.logger.Debug("ackloop:  done send ack")
	return count
}
