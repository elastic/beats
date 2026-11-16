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

//go:build linux

package auditd

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

// pendingReader holds a message to flush, like a multiline reader.
type pendingReader struct {
	pending  *reader.Message
	deadline time.Time
}

func (*pendingReader) Next() (reader.Message, error) { return reader.Message{}, io.EOF }
func (*pendingReader) Close() error                  { return nil }

func (p *pendingReader) PendingDeadline() (bool, time.Time) { return p.pending != nil, p.deadline }

func (p *pendingReader) FlushPending(bool) (reader.Message, bool, error) {
	if p.pending == nil {
		return reader.Message{}, false, nil
	}
	msg := *p.pending
	p.pending = nil
	return msg, true, nil
}

func TestParserBufferedFlusher(t *testing.T) {
	const line = `type=SYSCALL msg=audit(1485893834.891:18877199): arch=c000003e syscall=59 success=yes exit=0 a0=7f095d0a4b88 items=2 ppid=1234 pid=5678 auid=1000 uid=0 gid=0 comm="ls" exe="/bin/ls" key=(null)`
	deadline := time.Now().Add(time.Minute)

	newPending := func() *pendingReader {
		return &pendingReader{
			pending:  &reader.Message{Content: []byte(line), Bytes: len(line) + 1, Fields: mapstr.M{}},
			deadline: deadline,
		}
	}

	tests := map[string]struct {
		mode Mode
		// whether the flushed message is parsed by the mode
		parsed bool
	}{
		"parse": {mode: ModeParse, parsed: true},
		"none":  {mode: ModeNone},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Mode = tc.mode
			p := NewParser(newPending(), cfg, logptest.NewTestingLogger(t, ""))

			pending, dl := reader.PendingDeadline(p)
			assert.True(t, pending, "pending message of the wrapped reader")
			assert.Equal(t, deadline, dl, "deadline of the wrapped reader")

			msg, ok, err := reader.FlushPending(p, true)
			require.NoError(t, err, "FlushPending")
			require.True(t, ok, "FlushPending")
			assert.Equal(t, line, string(msg.Content), "content")
			assert.Equal(t, len(line)+1, msg.Bytes, "bytes")

			v, parsed := auditdLogField(msg.Fields, "syscall")
			assert.Equal(t, tc.parsed, parsed, "the flushed message is parsed only in parse mode")
			if tc.parsed {
				assert.Equal(t, "execve", v, "auditd.log.syscall")
				assert.Equal(t, int64(1485893834), msg.Ts.Unix(), "timestamp")
			}

			_, ok, err = reader.FlushPending(p, true)
			require.NoError(t, err, "second FlushPending")
			assert.False(t, ok, "nothing left to flush")
		})
	}

	t.Run("coalesce does not delegate", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Mode = ModeCoalesce
		p := NewParser(newPending(), cfg, logptest.NewTestingLogger(t, ""))
		t.Cleanup(func() { _ = p.Close() })

		pending, _ := reader.PendingDeadline(p)
		assert.False(t, pending, "the coalescing parser must not expose the reader below it")
		_, ok, err := reader.FlushPending(p, true)
		require.NoError(t, err, "FlushPending")
		assert.False(t, ok, "the coalescing parser must not flush the reader below it")
	})
}
