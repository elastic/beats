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

package pipeline

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
)

// queueReader is a standalone stateless helper goroutine to dispatch
// reads of the queue without blocking eventConsumer's main loop.
type queueReader struct {
	req  chan queueReaderRequest // "give me a batch for this target"
	resp chan *ttlBatch          // "here is your batch, or nil"
}

type queueReaderRequest struct {
	queue      queue.Queue
	retryer    retryer
	batchSize  int
	timeToLive int
}

func makeQueueReader() queueReader {
	qr := queueReader{
		req:  make(chan queueReaderRequest, 1),
		resp: make(chan *ttlBatch),
	}
	return qr
}

func (qr *queueReader) run(logger *logp.Logger) {
	logger.Debug("pipeline event consumer queue reader: start")
	for {
		fmt.Printf("queueReader run loop\n")
		req, ok := <-qr.req
		if !ok {
			fmt.Printf("queueReader.req closed, ending run loop\n")
			// The request channel is closed, we're shutting down
			logger.Debug("pipeline event consumer queue reader: stop")
			return
		}
		fmt.Printf("queueReader got read request\n")
		queueBatch, _ := req.queue.Get(req.batchSize)
		fmt.Printf("queueReader finished reading queue\n")
		var batch *ttlBatch
		if queueBatch != nil {
			batch = newBatch(req.retryer, queueBatch, req.timeToLive)
		}
		select {
		case qr.resp <- batch:
		case <-qr.req:
			// If the request channel unblocks before we've sent our response,
			// it means we're shutting down and the pending request can be
			// discarded.
			fmt.Printf("queue shut down before sending read response\n")
			logger.Debug("pipeline event consumer queue reader: stop")
			return
		}
	}
}
