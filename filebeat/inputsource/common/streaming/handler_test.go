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

package streaming

import (
	"bufio"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/filebeat/inputsource"
	"github.com/elastic/beats/v7/libbeat/common/cfgtype"
	"github.com/elastic/elastic-agent-libs/logp"
)

// TestSplitHandlerDeepCopy tests that the SplitHandlerFactory creates a handler
// that provides deep copies of the scanned data to the consumer callback.
func TestSplitHandlerDeepCopy(t *testing.T) {
	logger := logp.NewNopLogger()

	var receivedMessages []string
	var bufferSnapshots [][]byte

	// Simulate a consumer callback:
	// 1. Stores the received message as a string for verification.
	// 2. Keeps a reference to the raw byte slice.
	// 3. Overwrites the slice to detect whether future callbacks receive
	//    shared memory instead of independent copies.
	networkFunc := func(data []byte, metadata inputsource.NetworkMetadata) {
		receivedMessages = append(receivedMessages, string(data))
		bufferSnapshots = append(bufferSnapshots, data)

		for i := range data {
			data[i] = 'X'
		}
	}

	metadataFunc := func(conn net.Conn) inputsource.NetworkMetadata {
		return inputsource.NetworkMetadata{}
	}

	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	header := "PREFIX:"
	smallMsg := header + "small_event_data"
	largeMsg := header + strings.Repeat("large_event_data_that_fills_buffer_", 10) + "end"
	expectedMessages := []string{smallMsg, largeMsg, smallMsg}

	input := strings.Join(expectedMessages, "\n") + "\n"

	go func() {
		defer client.Close()
		_, err := client.Write([]byte(input))
		require.NoError(t, err)
	}()

	factory := SplitHandlerFactory(inputsource.FamilyTCP, logger, metadataFunc, networkFunc, bufio.ScanLines)
	config := ListenerConfig{
		MaxMessageSize: cfgtype.ByteSize(2048),
		Timeout:        1 * time.Second,
	}
	handler := factory(config)

	err := handler(t.Context(), server)
	assert.NoError(t, err)

	require.Len(t, receivedMessages, len(expectedMessages), "should receive all messages")

	for i, received := range receivedMessages {
		assert.Equal(t, expectedMessages[i], received, "message %d should be received intact", i)
	}

	// Verify that each callback received an independent copy of the buffer.
	for i := range bufferSnapshots {
		assert.Equal(t,
			strings.Repeat("X", len(expectedMessages[i])),
			string(bufferSnapshots[i]),
			"buffer %d should be fully mutated inside the callback", i,
		)
	}
}
