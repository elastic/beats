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
)

// TestACKWaitClosesOnClose verifies the disk queue producer's ack-wait channel
// is open until Close and closed afterward. The disk queue persists events
// durably and does not track in-memory acknowledgments, so Close is the only
// signal to wait for, and repeated Close calls must remain safe.
func TestACKWaitClosesOnClose(t *testing.T) {
	p := &diskQueueProducer{
		done:    make(chan struct{}),
		ackWait: make(chan struct{}),
	}

	select {
	case <-p.ACKWaitChan():
		t.Fatal("ackWait must be open before Close")
	default:
	}

	p.Close()

	select {
	case <-p.ACKWaitChan():
	default:
		t.Fatal("ackWait must be closed after Close")
	}

	// Close is idempotent: a second call must not panic on the already-closed
	// channels.
	p.Close()
}
