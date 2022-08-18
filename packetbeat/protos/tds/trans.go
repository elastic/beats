package tds

import (
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/packetbeat/procs"
	"github.com/elastic/beats/v7/packetbeat/protos/applayer"
	"github.com/elastic/elastic-agent-libs/logp"
)

type transactions struct {
	config *transactionConfig

	requests  messageList
	responses messageList

	onTransaction transactionHandler

	watcher procs.ProcessesWatcher
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
	logp.Info("trans.init()")
	trans.config = c
	trans.onTransaction = cb
}

func (trans *transactions) onMessage(
	tuple *common.IPPortTuple,
	dir uint8,
	msg *message,
) error {
	logp.Info("trans.onMessage()")
	var err error

	msg.Tuple = *tuple
	msg.Transport = applayer.TransportTCP
	msg.CmdlineTuple = trans.watcher.FindProcessesTuple(&msg.Tuple, msg.Transport)

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
	tuple *common.IPPortTuple,
	dir uint8,
	msg *message,
) error {
	logp.Info("trans.onRequest()")
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
	} else {
		trans.requests.append(msg)
	}

	if !msg.isComplete {
		return nil
	}

	if isDebug {
		debugf("request message complete")
	}

	return trans.correlate()
}

// onRequest handles response messages, merging with incomplete requests
// and adding non-merged responses into the correlation list.
func (trans *transactions) onResponse(
	tuple *common.IPPortTuple,
	dir uint8,
	msg *message,
) error {
	logp.Info("trans.onResponse()")
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
	logp.Info("trans.tryMergeRequest()")
	msg.isComplete = true
	return false, nil
}

func (trans *transactions) tryMergeResponses(prev, msg *message) (merged bool, err error) {
	logp.Info("trans.tryMergeResponses()")
	msg.isComplete = true
	return false, nil
}

func (trans *transactions) correlate() error {
	logp.Info("trans.correlate()")
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

	// merge requests with responses into transactions
	for !responses.empty() && !requests.empty() {
		resp := responses.first()
		if !resp.isComplete {
			break
		}

		requ := requests.pop()
		responses.pop()

		logp.Info("trans.correlate: request: %v", requ)
		logp.Info("trans.correlate: response: %v", resp)

		if err := trans.onTransaction(requ, resp); err != nil {
			return err
		}
	}

	return nil
}

func (ml *messageList) append(msg *message) {
	logp.Info("trans.append()")
	if ml.tail == nil {
		ml.head = msg
	} else {
		ml.tail.next = msg
	}
	msg.next = nil
	ml.tail = msg
}

func (ml *messageList) empty() bool {
	logp.Info("trans.empty()")
	return ml.head == nil
}

func (ml *messageList) pop() *message {
	logp.Info("trans.pop()")
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
	logp.Info("trans.first()")
	return ml.head
}

func (ml *messageList) last() *message {
	logp.Info("trans.last()")
	return ml.tail
}
