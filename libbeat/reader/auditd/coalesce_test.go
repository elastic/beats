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
	"errors"
	"io"
	"slices"
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/reader"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func coalesceConfig() Config {
	return Config{
		Mode:        ModeCoalesce,
		LogErrors:   true,
		AddErrorKey: true,
	}
}

func TestCoalescingCompoundEvent(t *testing.T) {
	lines := [][]byte{
		[]byte(`type=SYSCALL msg=audit(1626700000.000:100): arch=c000003e syscall=59 success=yes exit=0 a0=55b7e3a1e8c0 a1=55b7e3a1ea40 a2=55b7e3a1e010 a3=7ffd3a263100 items=2 ppid=1000 pid=2000 auid=1000 uid=0 gid=0 euid=0 suid=0 fsuid=0 egid=0 sgid=0 fsgid=0 tty=pts0 ses=1 comm="ls" exe="/usr/bin/ls" key=(null)`),
		[]byte(`type=EXECVE msg=audit(1626700000.000:100): argc=2 a0="ls" a1="-la"`),
		[]byte(`type=CWD msg=audit(1626700000.000:100): cwd="/home/user"`),
		[]byte(`type=PATH msg=audit(1626700000.000:100): item=0 name="/usr/bin/ls" inode=123456 dev=08:01 mode=0100755 ouid=0 ogid=0 rdev=00:00 nametype=NORMAL`),
		[]byte(`type=PROCTITLE msg=audit(1626700000.000:100): proctitle=6C73002D6C61`),
		[]byte(`type=EOE msg=audit(1626700000.000:100):`),
	}
	r := &testReader{messages: lines}
	p := NewParser(r, coalesceConfig(), logptest.NewTestingLogger(t, t.Name()))

	msg, err := p.Next()
	if err != nil {
		t.Fatalf("Next() returned error: %v", err)
	}

	if got := msg.Ts.Unix(); got != 1626700000 {
		t.Errorf("Ts.Unix() = %d; want 1626700000", got)
	}

	checkField(t, msg.Fields, "auditd.message_type", "syscall")
	checkField(t, msg.Fields, "auditd.sequence", uint32(100))
	checkField(t, msg.Fields, "process.pid", 2000)
	checkField(t, msg.Fields, "process.executable", "/usr/bin/ls")
	checkField(t, msg.Fields, "process.args", []string{"ls", "-la"})
	checkField(t, msg.Fields, "process.working_directory", "/home/user")
	checkField(t, msg.Fields, "file.path", "/usr/bin/ls")

	action, _ := msg.Fields.GetValue("event.action")
	if action == nil || action == "" {
		t.Error("event.action is empty; want a value from aucoalesce normalisation")
	}

	_, err = p.Next()
	if !errors.Is(err, io.EOF) {
		t.Errorf("second Next() = %v; want io.EOF", err)
	}
}

func TestCoalescingInterleaved(t *testing.T) {
	lines := [][]byte{
		[]byte(`type=SYSCALL msg=audit(1626700000.000:200): arch=c000003e syscall=59 success=yes exit=0 a0=1 a1=2 a2=3 a3=4 items=0 ppid=100 pid=201 auid=1000 uid=0 gid=0 euid=0 suid=0 fsuid=0 egid=0 sgid=0 fsgid=0 tty=pts0 ses=1 comm="cat" exe="/usr/bin/cat" key=(null)`),
		[]byte(`type=SYSCALL msg=audit(1626700000.001:201): arch=c000003e syscall=59 success=yes exit=0 a0=1 a1=2 a2=3 a3=4 items=0 ppid=100 pid=202 auid=1000 uid=0 gid=0 euid=0 suid=0 fsuid=0 egid=0 sgid=0 fsgid=0 tty=pts0 ses=1 comm="grep" exe="/usr/bin/grep" key=(null)`),
		[]byte(`type=EXECVE msg=audit(1626700000.000:200): argc=2 a0="cat" a1="/etc/hosts"`),
		[]byte(`type=EXECVE msg=audit(1626700000.001:201): argc=3 a0="grep" a1="-i" a2="localhost"`),
		[]byte(`type=EOE msg=audit(1626700000.000:200):`),
		[]byte(`type=EOE msg=audit(1626700000.001:201):`),
	}
	r := &testReader{messages: lines}
	p := NewParser(r, coalesceConfig(), logptest.NewTestingLogger(t, t.Name()))

	msg1, err := p.Next()
	if err != nil {
		t.Fatalf("first Next() returned error: %v", err)
	}
	msg2, err := p.Next()
	if err != nil {
		t.Fatalf("second Next() returned error: %v", err)
	}

	exe1, _ := msg1.Fields.GetValue("process.executable")
	exe2, _ := msg2.Fields.GetValue("process.executable")
	exes := []any{exe1, exe2}

	if !slices.Contains(exes, "/usr/bin/cat") {
		t.Errorf("events missing /usr/bin/cat; got %v", exes)
	}
	if !slices.Contains(exes, "/usr/bin/grep") {
		t.Errorf("events missing /usr/bin/grep; got %v", exes)
	}

	_, err = p.Next()
	if !errors.Is(err, io.EOF) {
		t.Errorf("third Next() = %v; want io.EOF", err)
	}
}

func TestCoalescingTimeoutFlush(t *testing.T) {
	lines := [][]byte{
		[]byte(`type=SYSCALL msg=audit(1626700000.000:300): arch=c000003e syscall=2 success=yes exit=3 a0=1 a1=2 a2=3 a3=4 items=1 ppid=100 pid=301 auid=1000 uid=0 gid=0 euid=0 suid=0 fsuid=0 egid=0 sgid=0 fsgid=0 tty=pts0 ses=1 comm="open" exe="/usr/bin/open" key=(null)`),
	}

	r := &deadlineTestReader{
		messages:      lines,
		deadlineCount: 5,
		deadlineDelay: 500 * time.Millisecond,
	}
	p := NewParser(r, coalesceConfig(), logptest.NewTestingLogger(t, t.Name()))

	msg, err := p.Next()
	if err != nil {
		t.Fatalf("Next() returned error: %v", err)
	}

	checkField(t, msg.Fields, "auditd.sequence", uint32(300))

	_, err = p.Next()
	if !errors.Is(err, io.EOF) {
		t.Errorf("second Next() = %v; want io.EOF", err)
	}
}

// TestCoalescingInterleavedBytesAccounting verifies that when multiple groups
// are flushed simultaneously (e.g. by Maintain or Close), every emitted event
// has non-zero Bytes and the total equals the input bytes consumed. This
// exercises the path where Bytes=0 would cause the filestream session to drop
// events as empty.
func TestCoalescingInterleavedBytesAccounting(t *testing.T) {
	// Two incomplete groups (no EOE) that will be flushed together by Close
	// when the reader hits EOF.
	lines := [][]byte{
		[]byte(`type=SYSCALL msg=audit(1626700000.000:900): arch=c000003e syscall=59 success=yes exit=0 a0=1 a1=2 a2=3 a3=4 items=0 ppid=100 pid=901 auid=1000 uid=0 gid=0 euid=0 suid=0 fsuid=0 egid=0 sgid=0 fsgid=0 tty=pts0 ses=1 comm="a" exe="/a" key=(null)`),
		[]byte(`type=SYSCALL msg=audit(1626700000.001:901): arch=c000003e syscall=59 success=yes exit=0 a0=1 a1=2 a2=3 a3=4 items=0 ppid=100 pid=902 auid=1000 uid=0 gid=0 euid=0 suid=0 fsuid=0 egid=0 sgid=0 fsgid=0 tty=pts0 ses=1 comm="b" exe="/b" key=(null)`),
		[]byte(`type=EXECVE msg=audit(1626700000.000:900): argc=1 a0="a"`),
		[]byte(`type=EXECVE msg=audit(1626700000.001:901): argc=1 a0="b"`),
	}

	var totalInput int
	for _, l := range lines {
		totalInput += len(l)
	}

	r := &testReader{messages: lines}
	p := NewParser(r, coalesceConfig(), logptest.NewTestingLogger(t, t.Name()))

	var (
		totalOutput int
		count       int
	)
	for {
		msg, err := p.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("Next() returned unexpected error: %v", err)
		}
		if !msg.IsEmpty() {
			count++
		}
		if msg.Bytes == 0 {
			t.Errorf("event %d has Bytes=0 (dropped as empty)", count)
		}
		totalOutput += msg.Bytes
	}

	if count != 2 {
		t.Errorf("got %d events; want 2", count)
	}
	if totalOutput != totalInput {
		t.Errorf("sum(output.Bytes) = %d; want %d (sum of input bytes)", totalOutput, totalInput)
	}
}

// TestCoalescingStaggeredBytesAccounting covers the case where one group
// completes (via EOE) while another group's lines are already in the byte
// accumulator. Without inflight-aware distribution, the first group drains all
// of bytesRead and the second group later flushes with Bytes=0, which the
// filestream session treats as empty and drops.
func TestCoalescingStaggeredBytesAccounting(t *testing.T) {
	// seq 1000 completes with EOE; seq 1001's SYSCALL line has already been
	// read and is in bytesRead. seq 1001 has no EOE so it flushes at EOF.
	lines := [][]byte{
		[]byte(`type=SYSCALL msg=audit(1626700000.000:1000): arch=c000003e syscall=59 success=yes exit=0 a0=1 a1=2 a2=3 a3=4 items=0 ppid=100 pid=1001 auid=1000 uid=0 gid=0 euid=0 suid=0 fsuid=0 egid=0 sgid=0 fsgid=0 tty=pts0 ses=1 comm="a" exe="/a" key=(null)`),
		[]byte(`type=SYSCALL msg=audit(1626700000.001:1001): arch=c000003e syscall=59 success=yes exit=0 a0=1 a1=2 a2=3 a3=4 items=0 ppid=100 pid=1002 auid=1000 uid=0 gid=0 euid=0 suid=0 fsuid=0 egid=0 sgid=0 fsgid=0 tty=pts0 ses=1 comm="b" exe="/b" key=(null)`),
		[]byte(`type=EOE msg=audit(1626700000.000:1000):`),
	}

	var totalInput int
	for _, l := range lines {
		totalInput += len(l)
	}

	r := &testReader{messages: lines}
	p := NewParser(r, coalesceConfig(), logptest.NewTestingLogger(t, t.Name()))

	var (
		totalOutput int
		count       int
	)
	for {
		msg, err := p.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("Next() returned unexpected error: %v", err)
		}
		if !msg.IsEmpty() {
			count++
		}
		if msg.Bytes == 0 {
			t.Errorf("event %d has Bytes=0 (would be dropped as empty)", count)
		}
		totalOutput += msg.Bytes
	}

	if count != 2 {
		t.Errorf("got %d events; want 2", count)
	}
	if totalOutput != totalInput {
		t.Errorf("sum(output.Bytes) = %d; want %d (sum of input bytes)", totalOutput, totalInput)
	}
}

func TestCoalescingEOFFlush(t *testing.T) {
	lines := [][]byte{
		[]byte(`type=SYSCALL msg=audit(1626700000.000:400): arch=c000003e syscall=1 success=yes exit=10 a0=1 a1=2 a2=3 a3=4 items=0 ppid=100 pid=401 auid=1000 uid=0 gid=0 euid=0 suid=0 fsuid=0 egid=0 sgid=0 fsgid=0 tty=pts0 ses=1 comm="echo" exe="/usr/bin/echo" key=(null)`),
		[]byte(`type=EXECVE msg=audit(1626700000.000:400): argc=2 a0="echo" a1="hello"`),
	}
	r := &testReader{messages: lines}
	p := NewParser(r, coalesceConfig(), logptest.NewTestingLogger(t, t.Name()))

	msg, err := p.Next()
	if err != nil {
		t.Fatalf("Next() returned error: %v", err)
	}

	checkField(t, msg.Fields, "process.executable", "/usr/bin/echo")

	_, err = p.Next()
	if !errors.Is(err, io.EOF) {
		t.Errorf("second Next() = %v; want io.EOF", err)
	}
}

func TestCoalescingParseErrors(t *testing.T) {
	lines := [][]byte{
		[]byte(`type=SYSCALL msg=audit(1626700000.000:500): arch=c000003e syscall=59 success=yes exit=0 a0=1 a1=2 a2=3 a3=4 items=0 ppid=100 pid=501 auid=1000 uid=0 gid=0 euid=0 suid=0 fsuid=0 egid=0 sgid=0 fsgid=0 tty=pts0 ses=1 comm="id" exe="/usr/bin/id" key=(null)`),
		[]byte(`this is not a valid audit line`),
		[]byte(`another garbage line`),
		[]byte(`type=EOE msg=audit(1626700000.000:500):`),
	}
	r := &testReader{messages: lines}
	p := NewParser(r, coalesceConfig(), logptest.NewTestingLogger(t, t.Name()))

	msg, err := p.Next()
	if err != nil {
		t.Fatalf("Next() returned error: %v", err)
	}

	checkField(t, msg.Fields, "process.executable", "/usr/bin/id")

	_, err = p.Next()
	if !errors.Is(err, io.EOF) {
		t.Errorf("second Next() = %v; want io.EOF", err)
	}
}

func TestCoalescingNodePreservation(t *testing.T) {
	lines := [][]byte{
		[]byte(`node=audit-host.local type=SYSCALL msg=audit(1626700000.000:600): arch=c000003e syscall=59 success=yes exit=0 a0=1 a1=2 a2=3 a3=4 items=0 ppid=100 pid=601 auid=1000 uid=0 gid=0 euid=0 suid=0 fsuid=0 egid=0 sgid=0 fsgid=0 tty=pts0 ses=1 comm="hostname" exe="/usr/bin/hostname" key=(null)`),
		[]byte(`node=audit-host.local type=EOE msg=audit(1626700000.000:600):`),
	}
	r := &testReader{messages: lines}
	p := NewParser(r, coalesceConfig(), logptest.NewTestingLogger(t, t.Name()))

	msg, err := p.Next()
	if err != nil {
		t.Fatalf("Next() returned error: %v", err)
	}

	checkField(t, msg.Fields, "auditd.data.node", "audit-host.local")
}

func TestCoalescingSimpleRecord(t *testing.T) {
	lines := [][]byte{
		[]byte(`type=USER_LOGIN msg=audit(1626700000.000:700): pid=1000 uid=0 auid=1000 ses=1 msg='op=login acct="root" exe="/usr/sbin/sshd" hostname=10.0.0.1 addr=10.0.0.1 terminal=sshd res=success'`),
		[]byte(`type=EOE msg=audit(1626700000.000:700):`),
	}
	r := &testReader{messages: lines}
	p := NewParser(r, coalesceConfig(), logptest.NewTestingLogger(t, t.Name()))

	msg, err := p.Next()
	if err != nil {
		t.Fatalf("Next() returned error: %v", err)
	}

	checkField(t, msg.Fields, "auditd.message_type", "user_login")

	_, err = p.Next()
	if !errors.Is(err, io.EOF) {
		t.Errorf("second Next() = %v; want io.EOF", err)
	}
}

func TestCoalescingBytesAccounting(t *testing.T) {
	lines := [][]byte{
		[]byte(`type=SYSCALL msg=audit(1626700000.000:800): arch=c000003e syscall=59 success=yes exit=0 a0=1 a1=2 a2=3 a3=4 items=0 ppid=100 pid=801 auid=1000 uid=0 gid=0 euid=0 suid=0 fsuid=0 egid=0 sgid=0 fsgid=0 tty=pts0 ses=1 comm="a" exe="/a" key=(null)`),
		[]byte(`type=SYSCALL msg=audit(1626700000.001:801): arch=c000003e syscall=59 success=yes exit=0 a0=1 a1=2 a2=3 a3=4 items=0 ppid=100 pid=802 auid=1000 uid=0 gid=0 euid=0 suid=0 fsuid=0 egid=0 sgid=0 fsgid=0 tty=pts0 ses=1 comm="b" exe="/b" key=(null)`),
		[]byte(`not a valid audit line`),
		[]byte(`type=EOE msg=audit(1626700000.000:800):`),
		[]byte(`type=EOE msg=audit(1626700000.001:801):`),
	}

	var totalInput int
	for _, l := range lines {
		totalInput += len(l)
	}

	r := &testReader{messages: lines}
	p := NewParser(r, coalesceConfig(), logptest.NewTestingLogger(t, t.Name()))

	var totalOutput int
	for {
		msg, err := p.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("Next() returned unexpected error: %v", err)
		}
		totalOutput += msg.Bytes
	}

	if totalOutput != totalInput {
		t.Errorf("sum(output.Bytes) = %d; want %d (sum of input bytes)", totalOutput, totalInput)
	}
}

// checkField verifies that fields.GetValue(key) equals want. It handles
// []string specially since reflect.DeepEqual is needed for slice comparison.
func checkField(t *testing.T, fields mapstr.M, key string, want any) {
	t.Helper()

	got, _ := fields.GetValue(key)
	switch w := want.(type) {
	case []string:
		gs, ok := got.([]string)
		if !ok || !slices.Equal(gs, w) {
			t.Errorf("%s = %v (%T); want %v", key, got, got, want)
		}
	default:
		if got != want {
			t.Errorf("%s = %v (%T); want %v (%T)", key, got, got, want, want)
		}
	}
}

// deadlineTestReader returns real messages first, then simulates periodic
// ErrReadDeadline returns (as the TimeoutReader does during live tailing),
// and finally returns EOF.
type deadlineTestReader struct {
	messages      [][]byte
	currentLine   int
	deadlineCount int
	deadlineDelay time.Duration
	deadlinesSent int
}

func (*deadlineTestReader) Close() error { return nil }

func (r *deadlineTestReader) Next() (reader.Message, error) {
	if r.currentLine < len(r.messages) {
		line := r.messages[r.currentLine]
		r.currentLine++
		return reader.Message{
			Content: line,
			Bytes:   len(line),
			Fields:  mapstr.M{},
		}, nil
	}
	if r.deadlinesSent < r.deadlineCount {
		r.deadlinesSent++
		if r.deadlineDelay > 0 {
			time.Sleep(r.deadlineDelay)
		}
		return reader.Message{}, reader.ErrReadDeadline
	}
	return reader.Message{}, io.EOF
}
