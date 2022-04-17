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

package udp

import (
	"net"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/menderesk/beats/v7/filebeat/inputsource"
)

const (
	maxMessageSize = 20
	maxSocketSize  = 0
	timeout        = time.Second * 15
)

type info struct {
	message []byte
	mt      inputsource.NetworkMetadata
}

func TestReceiveEventFromUDP(t *testing.T) {
	tests := []struct {
		name     string
		message  []byte
		expected []byte
	}{
		{
			name:     "Sending a message under the MaxMessageSize limit",
			message:  []byte("Hello world"),
			expected: []byte("Hello world"),
		},
		{
			name:     "Sending a message over the MaxMessageSize limit will truncate the message",
			message:  []byte("Hello world not so nice"),
			expected: []byte("Hello world not so n"),
		},
	}

	ch := make(chan info)
	host := "localhost:0"
	config := &Config{
		Host:           host,
		MaxMessageSize: maxMessageSize,
		Timeout:        timeout,
		ReadBuffer:     maxSocketSize,
	}
	fn := func(message []byte, metadata inputsource.NetworkMetadata) {
		ch <- info{message: message, mt: metadata}
	}
	s := New(config, fn)
	err := s.Start()
	if !assert.NoError(t, err) {
		return
	}
	defer s.Stop()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			conn, err := net.Dial("udp", s.localaddress)
			if !assert.NoError(t, err) {
				return
			}
			defer conn.Close()

			_, err = conn.Write(test.message)
			if !assert.NoError(t, err) {
				return
			}
			info := <-ch
			assert.Equal(t, test.expected, info.message)
			if runtime.GOOS == "windows" {
				if len(test.expected) < len(test.message) {
					assert.Nil(t, info.mt.RemoteAddr)
					assert.True(t, info.mt.Truncated)
				} else {
					assert.NotNil(t, info.mt.RemoteAddr)
					assert.False(t, info.mt.Truncated)
				}
			} else {
				assert.NotNil(t, info.mt.RemoteAddr)
				assert.False(t, info.mt.Truncated)
			}
		})
	}
}
