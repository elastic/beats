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

package diskqueue

import (
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
)

type diskQueueConsumer struct {
}

type diskQueueBatch struct {
}

//
// diskQueueConsumer implementation of the queue.Consumer interface
//

func (consumer *diskQueueConsumer) Get(eventCount int) (queue.Batch, error) {
	panic("TODO: not implemented")
}

func (consumer *diskQueueConsumer) Close() error {
	panic("TODO: not implemented")
}

//
// diskQueueBatch implementation of the queue.Batch interface
//

func (batch *diskQueueBatch) Events() []publisher.Event {
	panic("TODO: not implemented")
}

func (batch *diskQueueBatch) ACK() {
	panic("TODO: not implemented")
}
