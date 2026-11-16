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

package cassandra

import (
	"strings"
	"testing"
)

func TestReadInetTruncatedPayload(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{name: "IPv4 truncated", data: []byte{4, 0x7f}},        // size=4, only 1 byte follows
		{name: "IPv6 truncated", data: []byte{16, 0x20, 0x01}}, // size=16, only 2 bytes follow
		{name: "IPv4 empty", data: []byte{4}},                  // size=4, zero bytes follow
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data := make([]byte, len(tc.data))
			copy(data, tc.data)
			decoder := ByteArrayDecoder{Data: &data}

			var panicVal any
			func() {
				defer func() { panicVal = recover() }()
				_, _ = decoder.ReadInet()
			}()

			if panicVal == nil {
				t.Fatal("expected panic on truncated inet payload, got none")
			}
			// The panic should be our controlled error message, not a runtime
			// index-out-of-bounds panic.
			err, ok := panicVal.(error)
			if !ok {
				t.Fatalf("panic value should be an error, got %T: %v", panicVal, panicVal)
			}
			if !strings.Contains(err.Error(), "not enough bytes") {
				t.Errorf("unexpected panic message: %v", err)
			}
		})
	}
}
