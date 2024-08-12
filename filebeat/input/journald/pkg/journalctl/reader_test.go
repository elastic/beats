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

package journalctl

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"testing"

	"github.com/elastic/elastic-agent-libs/logp"
)

//go:embed testdata/corner-cases.json
var coredumpJSON []byte

// TestEventWithNonStringData ensures the Reader can read data that is not a
// string. There is at least one real example of that: coredumps.
// This test uses a real example captured from journalctl -o json.
//
// If needed more test cases can be added in the future
func TestEventWithNonStringData(t *testing.T) {
	testCases := []json.RawMessage{}
	if err := json.Unmarshal(coredumpJSON, &testCases); err != nil {
		t.Fatalf("could not unmarshal the contents from 'testdata/message-byte-array.json' into map[string]any: %s", err)
	}

	for idx, event := range testCases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			stdout := io.NopCloser(&bytes.Buffer{})
			stderr := io.NopCloser(&bytes.Buffer{})
			r := Reader{
				logger:   logp.L(),
				dataChan: make(chan []byte),
				errChan:  make(chan string),
				stdout:   stdout,
				stderr:   stderr,
			}

			go func() {
				r.dataChan <- []byte(event)
			}()

			_, err := r.Next(context.Background())
			if err != nil {
				t.Fatalf("did not expect an error: %s", err)
			}
		})
	}
}
