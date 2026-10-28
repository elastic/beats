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

//go:build !integration

package readfile

import (
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/reader/readfile/encoding"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

// TestLineReaderNextAfterClose checks that Next returns os.ErrClosed once the
// reader is closed, whatever is left in the input buffer. Close recycles
// tempBuffer, so decoding a buffered line afterwards used to spin forever.
func TestLineReaderNextAfterClose(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{name: "complete lines buffered", input: "first\nsecond\nthird\n"},
		{name: "partial line buffered", input: "first\nsecond"},
		{name: "nothing buffered", input: "first\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newPlainLineReader(t, tc.input)

			line, n, err := nextWithTimeout(t, r)
			require.NoError(t, err, "first Next")
			assert.Equal(t, "first\n", string(line))
			assert.Equal(t, len("first\n"), n)

			require.NoError(t, r.Close())

			line, n, err = nextWithTimeout(t, r)
			assert.ErrorIs(t, err, os.ErrClosed)
			assert.Nil(t, line)
			assert.Zero(t, n)
		})
	}
}

// newPlainLineReader returns a LineReader over input using the plain codec.
func newPlainLineReader(t *testing.T, input string) *LineReader {
	t.Helper()
	codec, err := encoding.Plain(nil)
	require.NoError(t, err)
	r, err := NewLineReader(io.NopCloser(strings.NewReader(input)), Config{
		Codec:      codec,
		BufferSize: 1024,
		Terminator: LineFeed,
		MaxBytes:   unlimited,
	}, logptest.NewTestingLogger(t, ""))
	require.NoError(t, err)
	return r
}

// nextWithTimeout calls r.Next and fails the test if it does not return, so a
// LineReader stuck in decode fails the test instead of hanging the binary.
func nextWithTimeout(t *testing.T, r *LineReader) ([]byte, int, error) {
	t.Helper()
	type result struct {
		line []byte
		n    int
		err  error
	}
	done := make(chan result, 1)
	go func() {
		line, n, err := r.Next()
		done <- result{line, n, err}
	}()
	select {
	case res := <-done:
		return res.line, res.n, res.err
	case <-time.After(5 * time.Second):
		require.FailNow(t, "LineReader.Next did not return")
		return nil, 0, nil
	}
}
