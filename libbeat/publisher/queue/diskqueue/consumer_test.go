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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

func TestQueueGetObserver(t *testing.T) {
	reg := monitoring.NewRegistry()
	const eventCount = 50
	dq := diskQueue{
		observer: queue.NewQueueObserver(reg),
		readerLoop: &readerLoop{
			output: make(chan *readFrame, eventCount),
		},
	}
	for i := 0; i < eventCount; i++ {
		dq.readerLoop.output <- &readFrame{bytesOnDisk: 123}
	}
	_, err := dq.Get(eventCount)
	assert.NoError(t, err, "Queue Get call should succeed")
	assertRegistryUint(t, reg, "queue.consumed.events", eventCount, "Get call should report consumed events")
	assertRegistryUint(t, reg, "queue.consumed.bytes", eventCount*123, "Get call should report consumed bytes")
}

func assertRegistryUint(t *testing.T, reg *monitoring.Registry, key string, expected uint64, message string) {
	t.Helper()

	entry := reg.Get(key)
	if entry == nil {
		assert.Failf(t, message, "registry key '%v' doesn't exist", key)
		return
	}
	value, ok := reg.Get(key).(*monitoring.Uint)
	if !ok {
		assert.Failf(t, message, "registry key '%v' doesn't refer to a uint64", key)
		return
	}
	assert.Equal(t, expected, value.Get(), message)
}
