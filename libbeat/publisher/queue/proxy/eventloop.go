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

package proxyqueue

func (b *broker) newBatch() *ProxiedBatch {
	return &ProxiedBatch{
		queue:    b,
		doneChan: make(chan batchDoneMsg),
	}
}

func (b *broker) run() {
	var (
		pendingBatch = b.newBatch()
		pendingACKs  pendingACKsList
	)

	for {
		var pushChan chan pushRequest
		// Push requests are enabled if the pending batch isn't yet full.
		if len(pendingBatch.entries) < b.batchSize {
			pushChan = b.pushChan
		}

		var getChan chan getRequest
		// Get requests are enabled if the current pending batch is nonempty.
		if len(pendingBatch.entries) > 0 {
			getChan = b.getChan
		}

		select {
		case <-b.done:
			return

		case req := <-pushChan: // producer pushing new event
			b.handlePushRequest(&req)

		case req := <-getChan: // consumer asking for next batch
			b.handleGetRequest(&req)

		case <-pendingACKs.nextDoneChan():
			// TODO: propagate ACKs
		}
	}
}

func (b *broker) handlePushRequest(req *pushRequest) {
	req.responseChan <- b.nextEntryID
	/*l.buf.insert(queueEntry{
		event:    req.event,
		id:       b.nextEntryID,
		producer: req.producer,
	})*/
	b.nextEntryID++
}

func (b *broker) handleGetRequest(req *getRequest) {
	/*start, buf := l.buf.reserve(req.entryCount)
	count := len(buf)
	if count == 0 {
		panic("empty batch returned")
	}

	ackCH := newBatchACKState(start, count, l.buf.entries)

	req.responseChan <- getResponse{ackCH.doneChan, buf}
	l.pendingACKs.append(ackCH)*/
}
