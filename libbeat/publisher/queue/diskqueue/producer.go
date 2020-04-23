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

type diskQueueProducer struct {
	// The disk queue that created this producer.
	queue *diskQueue

	// The configuration this producer was created with.
	config queue.ProducerConfig
}

//
// diskQueueProducer implementation of the queue.Producer interface
//

func (producer *diskQueueProducer) Publish(event publisher.Event) bool {
	panic("TODO: not implemented")
}

func (producer *diskQueueProducer) TryPublish(event publisher.Event) bool {
	panic("TODO: not implemented")
}

func (producer *diskQueueProducer) Cancel() int {
	panic("TODO: not implemented")
}
