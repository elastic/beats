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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/reader"
	"github.com/elastic/beats/v7/libbeat/reader/readertest"
	"github.com/elastic/beats/v7/pkg/logp/logptest"
	"github.com/elastic/beats/v7/pkg/mapstr"
)

func TestParserBufferedFlusher(t *testing.T) {
	const line = `type=SYSCALL msg=audit(1485893834.891:18877199): arch=c000003e syscall=59 success=yes exit=0 a0=7f095d0a4b88 items=2 ppid=1234 pid=5678 auid=1000 uid=0 gid=0 comm="ls" exe="/bin/ls" key=(null)`
	deadline := time.Now().Add(time.Minute)
	newParser := func(mode Mode) reader.Reader {
		cfg := DefaultConfig()
		cfg.Mode = mode
		src := &readertest.PendingReader{
			Pending:  &reader.Message{Content: []byte(line), Bytes: len(line) + 1, Fields: mapstr.M{}},
			Deadline: deadline,
		}
		return NewParser(src, cfg, logptest.NewTestingLogger(t, ""))
	}

	for _, tc := range []struct {
		mode   Mode
		parsed bool
	}{{ModeParse, true}, {ModeNone, false}} {
		t.Run(string(tc.mode), func(t *testing.T) {
			p := newParser(tc.mode)
			pending, dl := reader.PendingDeadline(p)
			assert.True(t, pending, "pending message of wrapped reader")
			assert.Equal(t, deadline, dl, "deadline of wrapped reader")

			msg, ok, err := reader.FlushPending(p, true)
			require.NoError(t, err, "FlushPending")
			require.True(t, ok, "FlushPending")
			assert.Equal(t, line, string(msg.Content), "content")
			assert.Equal(t, len(line)+1, msg.Bytes, "bytes")

			v, parsed := auditdLogField(msg.Fields, "syscall")
			assert.Equal(t, tc.parsed, parsed, "flushed message parsed only in parse mode")
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
		p := newParser(ModeCoalesce)
		t.Cleanup(func() { _ = p.Close() })
		pending, _ := reader.PendingDeadline(p)
		assert.False(t, pending, "coalescing parser must not expose wrapped reader")
		_, ok, err := reader.FlushPending(p, true)
		require.NoError(t, err, "FlushPending")
		assert.False(t, ok, "coalescing parser must not flush wrapped reader")
	})
}
