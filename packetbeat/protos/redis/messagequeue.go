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

package redis

import (
	"math"
)

// Message interface needs to be implemented by types in order to be stored
// in a MessageQueue.
type Message interface {
	// Size returns the size of the current element.
	Size() int
}

type listEntry struct {
	item Message
	next *listEntry
}

// MessageQueue defines a queue that automatically evicts messages based on
// the total size or number of elements contained.
type MessageQueue struct {
	head, tail *listEntry
	bytesAvail int64
	slotsAvail int32
}

// MessageQueueConfig represents the configuration for a MessageQueue.
// Setting any limit to zero disables the limit.
type MessageQueueConfig struct {
	// MaxBytes is the maximum number of bytes that can be stored in the queue.
	MaxBytes int64 `config:"queue_max_bytes"`

	// MaxMessages sets a limit on the number of messages that the queue can hold.
	MaxMessages int32 `config:"queue_max_messages"`
}

// NewMessageQueue creates a new MessageQueue with the given configuration.
func NewMessageQueue(c MessageQueueConfig) (queue MessageQueue) {
	queue.bytesAvail = c.MaxBytes
	if queue.bytesAvail <= 0 {
		queue.bytesAvail = math.MaxInt64
	}
	queue.slotsAvail = c.MaxMessages
	if queue.slotsAvail <= 0 {
		queue.slotsAvail = math.MaxInt32
	}
	return queue
}

// Append appends a new message into the queue, returning the number of
// messages evicted to make room, if any.
func (ml *MessageQueue) Append(msg Message) (evicted int) {
	size := int64(msg.Size())
	evicted = ml.adjust(size)
	ml.slotsAvail--
	ml.bytesAvail -= size
	entry := &listEntry{
		item: msg,
	}
	if ml.tail == nil {
		ml.head = entry
	} else {
		ml.tail.next = entry
	}
	ml.tail = entry
	return evicted
}

// IsEmpty returns if the MessageQueue is empty.
func (ml *MessageQueue) IsEmpty() bool {
	return ml.head == nil
}

// Pop returns the oldest message in the queue, if any.
func (ml *MessageQueue) Pop() Message {
	if ml.head == nil {
		return nil
	}

	msg := ml.head
	ml.head = msg.next
	if ml.head == nil {
		ml.tail = nil
	}
	ml.slotsAvail++
	ml.bytesAvail += int64(msg.item.Size())
	return msg.item
}

func (ml *MessageQueue) adjust(msgSize int64) (evicted int) {
	if ml.slotsAvail == 0 {
		ml.Pop()
		evicted++
	}
	for ml.bytesAvail < msgSize && !ml.IsEmpty() {
		ml.Pop()
		evicted++
	}
	return evicted
}
