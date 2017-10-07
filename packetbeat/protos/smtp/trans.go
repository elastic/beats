package smtp

import (
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

	onTransaction transactionHandler
}

type transactionConfig struct {
	transactionTimeout time.Duration
}

type transactionHandler func(requ, resp *message) error

// List of messages available for correlation
type messageList struct {
	head, tail *message
}

func (trans *transactions) init(c *transactionConfig, cb transactionHandler) {
	trans.config = c
	trans.onTransaction = cb
}

func (trans *transactions) onMessage(
	tuple *common.IPPortTuple,
	dir uint8,
	msg *message,
) error {
	msg.Tuple = *tuple
	msg.Transport = applayer.TransportTCP
	msg.CmdlineTuple = procs.ProcWatcher.FindProcessesTuple(&msg.Tuple)

	if msg.IsRequest {
		if isDebug {
			debugf("Received request with tuple: %s", tuple)
		}
		trans.requests.append(msg)
	} else {
		if isDebug {
			debugf("Received response with tuple: %s", tuple)
		}
		trans.responses.append(msg)
	}

	return trans.correlate()
}

func (trans *transactions) correlate() error {
	requests := &trans.requests
	responses := &trans.responses

	// drop responses with missing requests, unless they are "prompt"
	// responses (220, 421, etc.)
	if requests.empty() {
		for !responses.empty() {
			resp := responses.pop()
			switch resp.statusCode {
			case 220, 421:
				if err := trans.onTransaction(nil, resp); err != nil {
					return err
				}
			default:
				logp.Warn("Response from unknown transaction. Ignoring.")
			}
		}
		return nil
	}

	// merge requests with responses into transactions
	for !responses.empty() && !requests.empty() {
		resp := responses.pop()
		requ := requests.pop()

		if err := trans.onTransaction(requ, resp); err != nil {
			return err
		}
	}

	return nil
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
