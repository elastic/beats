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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/logp"
)

func TestNoBatchAssemblyOnNilTarget(t *testing.T) {
	// Create a minimal struct with only the channels we need. Batch assembly
	// is triggered determinstically (i.e. no selects) at the start of each
	// iteration of the run loop, so this way we can test synchronously
	// instead of starting up the full goroutine and relying on a timeout,
	// which can cause flakiness on CI. (This test does not pass without the
	// code change to check for a nil channel.)
	c := &eventConsumer{
		logger: logp.NewLogger("eventConsumer test"),
		queueReader: queueReader{
			req: make(chan queueReaderRequest, 1),
		},
		done: make(chan struct{}),
	}

	// Close immediately so the run loop returns
	close(c.done)

	c.run()

	// Make sure no read request was sent
	_, ok := <-c.queueReader.req
	assert.False(t, ok, "The queue reader shouldn't get a read request when the target is nil")
}
