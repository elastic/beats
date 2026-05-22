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
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/reader"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var update = flag.Bool("update", false, "update expected parser output json file")

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
				"arch":        "x86_64", // auparse resolves c000003e → x86_64
				"syscall":     "execve", // auparse resolves syscall 59 → execve
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
				// auparse normalises res → result; parser restores res so the
				// ingest pipeline rename auditd.log.res → event.outcome fires.
				"res":    "success",
				"result": "success",
			},
			wantEpoch: 1489636960,
		},
		"execve with decoded args": {
			cfg:  DefaultConfig(),
			line: []byte(`type=EXECVE msg=audit(1485893834.891:18877200): argc=3 a0="ls" a1="-la" a2="/tmp"`),
			wantLogFields: map[string]string{
				"record_type": "EXECVE",
				"sequence":    "18877200",
				"argc":        "3",
				"a0":          "ls",
				"a1":          "-la",
				"a2":          "/tmp",
			},
		},
		"avc selinux denial": {
			cfg:  DefaultConfig(),
			line: []byte(`type=AVC msg=audit(1226874073.147:96): avc:  denied  { getattr } for  pid=2465 comm="httpd" path="/var/www/html/file1" dev=dm-0 ino=284133 scontext=unconfined_u:system_r:httpd_t:s0 tcontext=unconfined_u:object_r:samba_share_t:s0 tclass=file`),
			wantLogFields: map[string]string{
				"record_type": "AVC",
				"sequence":    "96",
				"seresult":    "denied",
				"seperms":     "getattr",
				"comm":        "httpd",
				"tclass":      "file",
			},
			wantEpoch: 1226874073,
		},
		"syscall record with node prefix": {
			// When name_format=hostname is set in /etc/audit/auditd.conf, userspace
			// auditd prepends "node=<hostname> " to every log line. auparse does not
			// handle this prefix (it reads from the kernel directly), so the parser
			// must strip it and expose the value as auditd.log.node.
			cfg:  DefaultConfig(),
			line: []byte(`node=myhost.example.com type=SYSCALL msg=audit(1485893834.891:18877199): arch=c000003e syscall=59 success=yes exit=0 a0=7f095d0a4b88 items=2 ppid=1234 pid=5678 auid=1000 uid=0 gid=0 comm="ls" exe="/bin/ls" key=(null)`),
			wantLogFields: map[string]string{
				"record_type": "SYSCALL",
				"sequence":    "18877199",
				"node":        "myhost.example.com",
				"arch":        "x86_64",
				"syscall":     "execve",
				"pid":         "5678",
				"comm":        "ls",
			},
			wantEpoch: 1485893834,
		},
		"data error still sets record_type and sequence": {
			// EXECVE with argc=3 but a1 missing causes Data() to error;
			// record_type and sequence from the parsed header must survive.
			cfg:           DefaultConfig(),
			line:          []byte(`type=EXECVE msg=audit(1485893834.891:18877201): argc=3 a0="ls" a2="/tmp"`),
			wantLogFields: map[string]string{"record_type": "EXECVE", "sequence": "18877201"},
			wantErrorKey:  true,
		},
		"invalid line adds error key": {
			cfg:           DefaultConfig(),
			line:          []byte(`not a valid audit line`),
			wantLogFields: map[string]string{},
			wantErrorKey:  true,
		},
		"invalid line no error key when disabled": {
			cfg:           Config{LogErrors: false, AddErrorKey: false},
			line:          []byte(`not a valid audit line`),
			wantLogFields: map[string]string{},
			wantErrorKey:  false,
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

// TestParserMultiRecord verifies that related records sharing the same
// sequence number are not consolidated into a single message. This is
// to preserve the prior behaviour of the integrations.
func TestParserMultiRecord(t *testing.T) {
	lines := [][]byte{
		[]byte(`type=SYSCALL msg=audit(1485893834.891:42): arch=c000003e syscall=59 success=yes exit=0 a0=7f095d0a4b88 items=1 ppid=1234 pid=5678 auid=1000 uid=0 gid=0 comm="ls" exe="/bin/ls" key=(null)`),
		[]byte(`type=EXECVE msg=audit(1485893834.891:42): argc=1 a0="ls"`),
	}
	r := &testReader{messages: lines}
	p := NewParser(r, DefaultConfig(), logptest.NewTestingLogger(t, ""))

	msg1, err := p.Next()
	require.NoError(t, err)
	seq1, ok := auditdLogField(msg1.Fields, "sequence")
	assert.True(t, ok, "sequence missing from first record")
	assert.Equal(t, "42", seq1)

	msg2, err := p.Next()
	require.NoError(t, err)
	seq2, ok := auditdLogField(msg2.Fields, "sequence")
	assert.True(t, ok, "sequence missing from second record")
	assert.Equal(t, "42", seq2)

	_, err = p.Next()
	assert.ErrorIs(t, err, io.EOF)
}

func TestLogFiles(t *testing.T) {
	const (
		inputPath  = "testdata/sample.log"
		goldenPath = "testdata/sample.log-expected.json"
	)

	f, err := os.Open(inputPath)
	require.NoError(t, err)
	defer f.Close()

	var lines [][]byte
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, []byte(scanner.Text()))
	}
	require.NoError(t, scanner.Err())

	p := NewParser(&testReader{messages: lines}, DefaultConfig(), logptest.NewTestingLogger(t, ""))
	var got []mapstr.M
	for {
		msg, err := p.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		var norm mapstr.M
		b, _ := json.Marshal(msg.Fields)
		_ = json.Unmarshal(b, &norm)
		got = append(got, norm)
	}

	if *update {
		b, err := json.MarshalIndent(got, "", "  ")
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(goldenPath, b, 0o644))
		return
	}

	b, err := os.ReadFile(goldenPath)
	require.NoError(t, err)
	var want []mapstr.M
	require.NoError(t, json.Unmarshal(b, &want))
	assert.Equal(t, want, got)
}
