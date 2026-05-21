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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/reader"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var _ reader.Reader = &testReader{}

type testReader struct {
	messages    [][]byte
	currentLine int
}

func (*testReader) Close() error { return nil }

func (t *testReader) Next() (reader.Message, error) {
	if t.currentLine == len(t.messages) {
		return reader.Message{}, io.EOF
	}
	line := t.messages[t.currentLine]
	t.currentLine++
	return reader.Message{
		Content: line,
		Bytes:   len(line),
		Fields:  mapstr.M{},
	}, nil
}

// auditdLogField returns the value of a field within msg.Fields["auditd"]["log"].
func auditdLogField(fields mapstr.M, key string) (interface{}, bool) {
	auditd, ok := fields["auditd"].(mapstr.M)
	if !ok {
		return nil, false
	}
	log, ok := auditd["log"].(mapstr.M)
	if !ok {
		return nil, false
	}
	v, ok := log[key]
	return v, ok
}

func TestParser(t *testing.T) {
	tests := map[string]struct {
		cfg Config
		// line to parse
		line []byte
		// auditd.log sub-fields that must be present with exact values
		wantLogFields map[string]string
		// whether an "error" top-level field should be present
		wantErrorKey bool
		// expected timestamp epoch (zero means don't check)
		wantEpoch int64
	}{
		"syscall record": {
			cfg:  DefaultConfig(),
			line: []byte(`type=SYSCALL msg=audit(1485893834.891:18877199): arch=c000003e syscall=59 success=yes exit=0 a0=7f095d0a4b88 items=2 ppid=1234 pid=5678 auid=1000 uid=0 gid=0 comm="ls" exe="/bin/ls" key=(null)`),
			wantLogFields: map[string]string{
				"record_type": "SYSCALL",
				"sequence":    "18877199",
				"arch":        "x86_64",  // auparse resolves c000003e → x86_64
				"syscall":     "execve",  // auparse resolves syscall 59 → execve
				"ppid":        "1234",
				"pid":         "5678",
				"auid":        "1000",
				"uid":         "0",
				"comm":        "ls",
				"exe":         "/bin/ls",
			},
			wantEpoch: 1485893834,
		},
		"user login record": {
			cfg:  DefaultConfig(),
			line: []byte(`type=USER_LOGIN msg=audit(1489636960.072:19623791): pid=28281 uid=0 auid=700 ses=6793 msg='op=login acct="root" exe="/usr/sbin/sshd" hostname=1.2.3.4 addr=1.2.3.4 terminal=sshd res=success'`),
			wantLogFields: map[string]string{
				"record_type": "USER_LOGIN",
				"sequence":    "19623791",
				"pid":         "28281",
				"uid":         "0",
				"auid":        "700",
			},
			wantEpoch: 1489636960,
		},
		"invalid line adds error key": {
			cfg:          DefaultConfig(),
			line:         []byte(`not a valid audit line`),
			wantLogFields: map[string]string{},
			wantErrorKey: true,
		},
		"invalid line no error key when disabled": {
			cfg:          Config{LogErrors: false, AddErrorKey: false},
			line:         []byte(`not a valid audit line`),
			wantLogFields: map[string]string{},
			wantErrorKey: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			r := &testReader{messages: [][]byte{tc.line}}
			p := NewParser(r, tc.cfg, logptest.NewTestingLogger(t, ""))

			msg, err := p.Next()
			require.NoError(t, err)

			for k, want := range tc.wantLogFields {
				got, ok := auditdLogField(msg.Fields, k)
				assert.True(t, ok, "auditd.log.%s missing", k)
				assert.Equal(t, want, got, "auditd.log.%s mismatch", k)
			}

			_, hasErr := msg.Fields["error"]
			assert.Equal(t, tc.wantErrorKey, hasErr, "error key presence mismatch")

			if tc.wantEpoch != 0 {
				assert.Equal(t, tc.wantEpoch, msg.Ts.Unix(), "timestamp mismatch")
			}

			_, err = p.Next()
			assert.ErrorIs(t, err, io.EOF)
		})
	}
}
