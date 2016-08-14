package lumberjack

import (
	"fmt"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/packetbeat/procs"
	"github.com/elastic/beats/packetbeat/protos/applayer"
)

type transactions struct {
	config *transactionConfig

	requests  messageList
	responses messageList

	active transaction

	onTransaction transactionHandler
}

type transactionConfig struct {
	transactionTimeout time.Duration
	outOfBandData      bool
}

type transactionHandler func(t transaction) error

type transaction struct {
	request  *message
	lastACK  uint32
	countACK int
}

// List of messages available for correlation
type messageList struct {
	head, tail *message
}

func (trans *transactions) init(c *transactionConfig, cb transactionHandler) {
	trans.config = c
	trans.onTransaction = cb
}

func (trans *transactions) onMessage(
	tuple *common.IpPortTuple,
	dir uint8,
	msg *message,
) error {
	var err error

	msg.Tuple = *tuple
	msg.Transport = applayer.TransportTcp
	msg.CmdlineTuple = procs.ProcWatcher.FindProcessesTuple(&msg.Tuple)

	if msg.IsRequest {
		if isDebug {
			debugf("Received request with tuple: %s", tuple)
		}
		err = trans.onRequest(tuple, dir, msg)
	} else {
		if isDebug {
			debugf("Received response with tuple: %s", tuple)
		}
		err = trans.onResponse(tuple, dir, msg)
	}

	return err
}

// onRequest handles request messages, merging with incomplete requests
// and adding non-merged requests into the correlation list.
func (trans *transactions) onRequest(
	tuple *common.IpPortTuple,
	dir uint8,
	msg *message,
) error {
	prev := trans.requests.last()

	merged, err := trans.tryMergeRequests(prev, msg)
	if err != nil {
		return err
	}
	if merged {
		if isDebug {
			debugf("request message got merged")
		}
		msg = prev

		if msg.isComplete {
			return trans.correlate()
		}
		return nil
	}

	if msg.op != opWindow {
		if !trans.config.outOfBandData {
			debugf("drop out of band message (op=%v, seq=%v)", msg.op, msg.seq)
			return nil
		}

		msg.AddNotes("out of band data message")
	}
	trans.requests.append(msg)

	if (prev != nil && prev.isComplete) || msg.isComplete {
		return trans.correlate()
	}
	return nil
}

// onRequest handles response messages, merging with incomplete requests
// and adding non-merged responses into the correlation list.
func (trans *transactions) onResponse(
	tuple *common.IpPortTuple,
	dir uint8,
	msg *message,
) error {
	prev := trans.responses.last()
	merged, err := trans.tryMergeResponses(prev, msg)
	if err != nil {
		return err
	}
	if merged {
		if isDebug {
			debugf("response message got merged")
		}
		msg = prev
	} else {
		trans.responses.append(msg)
	}

	if !msg.isComplete {
		return nil
	}

	if isDebug {
		debugf("response message complete")
	}

	return trans.correlate()
}

func (trans *transactions) tryMergeRequests(
	prev, msg *message,
) (merged bool, err error) {
	if prev == nil {
		return false, nil
	}

	if !(prev.isComplete || prev.ignore) {
		switch prev.op {
		case opWindow:
			return trans.tryMergeIntoWindow(prev, msg)
		case opData, opJSON, opCompressed:
			return trans.tryMergeOutOfBandData(prev, msg)
		}
	}

	return false, nil
}

func (trans *transactions) tryMergeIntoWindow(prev, msg *message) (bool, error) {
	switch msg.op {
	case opWindow:
		// two consecutive window messages -> something went wrong. =>
		// ignore previous window message and continue with current
		// window message
		prev.ignore = true

		msg.AddNotes("follow incomplete window")
		return false, nil

	case opCompressed:
		// batch request complete
		prev.isComplete = true
		prev.count = prev.seq
		prev.size += msg.size // payload bytes
		prev.Size += msg.Size // total number of bytes

		return true, nil

	case opJSON, opData:
		if msg.seq < prev.count+1 {
			prev.isComplete = true
			prev.AddNotes("incomplete window")

			msg.isComplete = true
			msg.AddNotes("out of band message")
			return false, nil
		}

		if prev.count+1 != msg.seq {
			prev.AddNotes(fmt.Sprintf("missing data frame from %v -> %v",
				prev.count+1, msg.seq))
		}

		prev.count++
		prev.isComplete = prev.count == prev.seq
		prev.size += msg.size // payload bytes
		prev.Size += msg.Size // total number of bytes

		return true, nil

	default:
		prev.AddNotes(fmt.Sprintf("Failed to add message with opcode %v", msg.op))
		prev.isComplete = true
		return true, nil
	}
}

func (trans *transactions) tryMergeOutOfBandData(prev, msg *message) (bool, error) {
	if prev.op != msg.op {
		prev.isComplete = true
		return false, nil
	}

	prev.count++
	prev.seq = msg.seq
	prev.size += msg.size
	prev.Size += msg.Size
	return true, nil
}

func (trans *transactions) tryMergeResponses(prev, msg *message) (merged bool, err error) {
	msg.isComplete = true
	return false, nil
}

func (trans *transactions) correlate() error {
	requests := &trans.requests
	responses := &trans.responses

	// drop responses with missing requests
	if requests.empty() {
		for !responses.empty() {
			logp.Warn("Response from unknown transaction. Ignoring.")
			responses.pop()
		}
		return nil
	}

	// drop ignored messages
	requests.dropIgnore()
	responses.dropIgnore()

	// merge responses into active transaction.
	for {
		if trans.active.request == nil {
			// start new transaction with request
			requ := requests.pop()
			if requ == nil { // no requests to correlate -> done
				break
			}
			trans.active.request = requ
		}

		if trans.active.request.ignore {
			if isDebug {
				debugf("In progress transaction canceled.")
			}
			trans.active.request = nil
			break
		}

		if !trans.active.request.isComplete {
			// waiting for request to be completed
			break
		}

		for !responses.empty() {
			resp := responses.first()
			if resp.ignore {
				// ignore and continue with next message
				responses.pop()
				continue
			}
			if !resp.isComplete {
				break
			}

			merged, finished := trans.active.mergeResponse(resp)
			if merged {
				responses.pop()
			}
			if finished {
				// publish and finalize active transaction such that a new one
				// can be handled
				err := trans.onTransaction(trans.active)
				trans.active.request = nil

				if err != nil {
					return err
				}
				break // handle next transaction
			}
		}
	}
	return nil
}

func (t *transaction) mergeResponse(resp *message) (merged, finished bool) {
	requ := t.request
}

func (ml *messageList) append(msg *message) {
	if ml.tail == nil {
		ml.head = msg
	} else {
		ml.tail.next = msg
	}
	msg.next = nil
	ml.tail = msg
}

func (ml *messageList) empty() bool {
	return ml.head == nil
}

func (ml *messageList) pop() *message {
	if ml.head == nil {
		return nil
	}

	msg := ml.head
	ml.head = ml.head.next
	if ml.head == nil {
		ml.tail = nil
	}
	return msg
}

func (ml *messageList) first() *message {
	return ml.head
}

func (ml *messageList) last() *message {
	return ml.tail
}

func (ml *messageList) dropIgnore() {
	for ml.head != nil {
		if !ml.head.ignore {
			return
		}

		ml.pop()
	}
}
