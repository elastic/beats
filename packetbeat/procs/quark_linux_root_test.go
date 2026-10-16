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

//go:build linux && (amd64 || arm64) && cgo && !integration

package procs

import (
	"net"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

// quarkTestDialEnv makes the re-executed test binary act as a
// short-lived client: dial the address, then exit immediately.
const quarkTestDialEnv = "QUARK_TEST_DIAL"

func TestMain(m *testing.M) {
	if addr := os.Getenv(quarkTestDialEnv); addr != "" {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			os.Exit(1)
		}
		conn.Close()
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// TestQuarkShortLivedConnection is the elastic/beats#51125 regression
// test against a real quark queue: a short-lived child process makes a
// TCP connection and exits; the connection must still resolve to the
// child after both the socket and the process are gone. This fails on
// the procfs backend, which needs the socket present in /proc.
//
// It requires root and a kernel where quark's eBPF backend can load,
// and is skipped otherwise.
func TestQuarkShortLivedConnection(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("skipping: this test must be run as root")
	}

	logger := logptest.NewTestingLogger(t, "procs")

	proc := &ProcessesWatcher{}
	err := proc.Init(ProcsConfig{Enabled: true, Backend: BackendKernelTracing}, logger)
	if err != nil {
		t.Skipf("skipping: cannot start the kernel_tracing backend: %v", err)
	}
	defer proc.Close()

	// A local listener gives the connection a stable server side and
	// tells us the child's ephemeral port when it connects.
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	accepted := make(chan net.Addr, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		accepted <- conn.RemoteAddr()
		conn.Close()
	}()

	// Re-execute the test binary as a short-lived client and wait for
	// it to exit, so that its socket and /proc entry are gone.
	exe, err := os.Executable()
	require.NoError(t, err)
	cmd := exec.Command(exe)
	cmd.Env = append(os.Environ(), quarkTestDialEnv+"="+listener.Addr().String())
	require.NoError(t, cmd.Start())
	childPid := cmd.Process.Pid
	require.NoError(t, cmd.Wait())

	var childAddr *net.TCPAddr
	select {
	case addr := <-accepted:
		var ok bool
		childAddr, ok = addr.(*net.TCPAddr)
		require.True(t, ok)
	case <-time.After(10 * time.Second):
		t.Fatal("child did not connect")
	}

	serverAddr, ok := listener.Addr().(*net.TCPAddr)
	require.True(t, ok)

	tuple := &common.IPPortTuple{
		BaseTuple: common.BaseTuple{
			SrcIP:   childAddr.IP,
			SrcPort: uint16(childAddr.Port),
			DstIP:   serverAddr.IP,
			DstPort: uint16(serverAddr.Port),
		},
	}

	// Quark delivers the connection after its hold time; the process
	// is dead, so only the kernel_tracing path can resolve it.
	var src common.Process
	require.Eventually(t, func() bool {
		src = proc.FindProcessesTupleTCP(tuple).Src
		return src.PID != 0
	}, 30*time.Second, 100*time.Millisecond,
		"short-lived connection was not resolved after the child exited")

	assert.Equal(t, childPid, src.PID, "resolved PID must be the exited child")
	assert.NotEmpty(t, src.Name)
	assert.False(t, src.StartTime.IsZero(), "start time must be set")

	// The server side of the connection must resolve to this test
	// process, which is still alive.
	dst := proc.FindProcessesTupleTCP(tuple).Dst
	assert.Equal(t, os.Getpid(), dst.PID, "server side must resolve to the test process")
}
