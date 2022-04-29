// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package cassandra

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

func (trans *transactions) init(c *transactionConfig, watcher procs.ProcessesWatcher, cb transactionHandler) {
	trans.config = c
	trans.watcher = watcher
	trans.onTransaction = cb
}

func (trans *transactions) onMessage(
	tuple *common.IPPortTuple,
	dir uint8,
	msg *message,
) error {
	var err error
	msg.Tuple = *tuple
	msg.Transport = applayer.TransportTCP
	msg.CmdlineTuple = trans.watcher.FindProcessesTupleTCP(&msg.Tuple)

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
	prev := trans.requests.last()
	merged, err := trans.tryMergeRequests(prev, msg)
	if err != nil {
		return err
	}
	if merged {
		msg = prev
	} else {
		trans.requests.append(msg)
	}

	if !msg.isComplete {
		return nil
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
	prev := trans.responses.last()
	merged, err := trans.tryMergeResponses(prev, msg)
	if err != nil {
		return err
	}
	if merged {
		msg = prev
	} else {
		trans.responses.append(msg)
	}

	if !msg.isComplete {
		return nil
	}

	return trans.correlate()
}

func (trans *transactions) tryMergeRequests(
	prev, msg *message,
) (merged bool, err error) {
	msg.isComplete = true
	return false, nil
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

			// if the response is EVENT, which pushed from server, we can accept that
			resp := responses.first()
			if !resp.isComplete {
				break
			}

			if resp.header["op"] == "EVENT" {
				if isDebug {
					logp.Debug("cassandra", "server pushed message,%v", resp.header)
				}

				responses.pop()

				if err := trans.onTransaction(nil, resp); err != nil {
					return err
				}

				return nil
			}

			logp.Warn("Response from unknown transaction. Ignoring. %v", resp.header)
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
