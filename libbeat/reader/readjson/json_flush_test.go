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

package readjson

import (
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/reader"
	"github.com/elastic/beats/v7/pkg/logp/logptest"
	"github.com/elastic/beats/v7/pkg/mapstr"
)

type pendingJSONReader struct {
	pending  *reader.Message
	deadline time.Time
}

func (*pendingJSONReader) Next() (reader.Message, error)        { return reader.Message{}, io.EOF }
func (*pendingJSONReader) Close() error                         { return nil }
func (p *pendingJSONReader) PendingDeadline() (bool, time.Time) { return p.pending != nil, p.deadline }
func (p *pendingJSONReader) FlushPending(bool) (reader.Message, bool, error) {
	if p.pending == nil {
		return reader.Message{}, false, nil
	}
	msg := *p.pending
	p.pending = nil
	return msg, true, nil
}

func TestJSONReaderBufferedFlusher(t *testing.T) {
	deadline := time.Now().Add(time.Minute)
	content := "{\"a\":\n1}"
	pending := func() *reader.Message {
		return &reader.Message{Content: []byte(content), Bytes: len(content) + 1, Fields: mapstr.M{}}
	}

	t.Run("JSONReader", func(t *testing.T) {
		src := &pendingJSONReader{pending: pending(), deadline: deadline}
		r := NewJSONReader(src, &Config{}, logptest.NewTestingLogger(t, ""))

		isPending, dl := reader.PendingDeadline(r)
		assert.True(t, isPending, "pending message of the wrapped reader")
		assert.Equal(t, deadline, dl, "deadline of the wrapped reader")

		msg, ok, err := reader.FlushPending(r, true)
		require.NoError(t, err, "FlushPending")
		require.True(t, ok, "FlushPending")
		v, err := msg.Fields.GetValue("json.a")
		require.NoError(t, err, "the flushed message must be decoded")
		assert.EqualValues(t, 1, v, "json.a")
		assert.Equal(t, len(content)+1, msg.Bytes, "bytes")
	})

	// JSONParser embeds JSONReader, so verify JSONParser.FlushPending overrides
	// the promoted JSONReader.FlushPending and writes into Target rather than "json".
	t.Run("JSONParser overrides promoted JSONReader.FlushPending", func(t *testing.T) {
		src := &pendingJSONReader{pending: pending(), deadline: deadline}
		r := NewJSONParser(src, &ParserConfig{Target: "parsed"}, logptest.NewTestingLogger(t, ""))

		msg, ok, err := reader.FlushPending(r, true)
		require.NoError(t, err, "FlushPending")
		require.True(t, ok, "FlushPending")
		v, err := msg.Fields.GetValue("parsed.a")
		require.NoError(t, err, "the flushed message must be decoded into the target")
		assert.EqualValues(t, 1, v, "parsed.a")
		_, err = msg.Fields.GetValue("json")
		assert.Error(t, err, "JSONReader processing must not be applied by JSONParser")
	})
}
