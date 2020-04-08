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
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
)

type diskQueue struct {
}

// NewQueue returns a disk-based queue configured with the given logger
// and settings.
func NewQueue(
	logger *logp.Logger,
	ackListener queue.ACKListener,
) queue.Queue {

	return &diskQueue{}
}

//
// diskQueue mplementation of the queue.Queue interface
//

func (dq *diskQueue) Close() error {
	return nil
}

func (dq *diskQueue) BufferConfig() queue.BufferConfig {
	return queue.BufferConfig{Events: 0}
}

func (dq *diskQueue) Producer(cfg queue.ProducerConfig) queue.Producer {
	return &diskQueueProducer{
		queue:  dq,
		config: cfg,
	}
}

func (dq *diskQueue) Consumer() queue.Consumer {
	return &diskQueueConsumer{}
}
