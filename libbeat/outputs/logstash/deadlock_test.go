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

package logstash

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp"
)

// flaky?
func TestDeadlockListener(t *testing.T) {
	const timeout = 5 * time.Millisecond
	log := logp.NewLogger("test")
	listener := newDeadlockListener(log, timeout)

	// Verify that the listener doesn't trigger when receiving regular acks
	for i := 0; i < 5; i++ {
		time.Sleep(timeout / 2)
		listener.ack(1)
	}
	select {
	case <-listener.doneChan:
		require.Fail(t, "Deadlock listener should not trigger unless there is no progress for the configured time interval")
	case <-time.After(timeout / 2):
	}

	// Verify that the listener does trigger when the acks stop
	select {
	case <-time.After(timeout):
		require.Fail(t, "Deadlock listener should trigger when there is no progress for the configured time interval")
	case <-listener.doneChan:
	}
}
