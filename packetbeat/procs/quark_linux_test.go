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
	"net/netip"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/packetbeat/protos/applayer"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	quark "github.com/elastic/go-quark"
)

// fakeQuarkQueue is a quarkQueue backed by static socket and process
// tables. All methods except Block are only called while holding the
// quarkWatcher mutex, so no synchronization is needed.
type fakeQuarkQueue struct {
	sockets map[socketKey]quark.Socket
	procs   map[int]quark.Process
	closed  chan struct{}
}

type socketKey struct {
	local, remote netip.AddrPort
}

func newFakeQuarkQueue() *fakeQuarkQueue {
	return &fakeQuarkQueue{
		sockets: make(map[socketKey]quark.Socket),
		procs:   make(map[int]quark.Process),
		closed:  make(chan struct{}),
	}
}

func (q *fakeQuarkQueue) addSocket(local, remote netip.AddrPort, s quark.Socket) {
	s.Local, s.Remote = local, remote
	q.sockets[socketKey{local, remote}] = s
}

func (q *fakeQuarkQueue) GetEvent() (quark.Event, bool) { return quark.Event{}, false }

func (q *fakeQuarkQueue) Block() error {
	time.Sleep(time.Millisecond)
	return nil
}

func (q *fakeQuarkQueue) Lookup(pid int) (quark.Process, bool) {
	p, ok := q.procs[pid]
	return p, ok
}

func (q *fakeQuarkQueue) SocketLookup(local, remote netip.AddrPort) (quark.Socket, bool) {
	key := socketKey{
		netip.AddrPortFrom(local.Addr().Unmap(), local.Port()),
		netip.AddrPortFrom(remote.Addr().Unmap(), remote.Port()),
	}
	s, ok := q.sockets[key]
	return s, ok
}

func (q *fakeQuarkQueue) Close() { close(q.closed) }

func TestQuarkFindProcessTuple(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "procs")

	const (
		curlPid    = 4242
		curlPpid   = 4000
		serverPid  = 5252
		legacyPid  = 6262
		udpPid     = 7272
		unnamedPid = 8282
	)

	fake := newFakeQuarkQueue()
	fake.procs[curlPid] = quark.Process{
		Pid:     curlPid,
		Comm:    "curl",
		Exe:     "/usr/bin/curl",
		Cmdline: []string{"curl", "http://elastic.co/"},
		Cwd:     "/home/user",
		Proc: quark.Proc{
			Ppid:     curlPpid,
			TimeBoot: 123456,
			Valid:    true,
		},
	}
	fake.procs[serverPid] = quark.Process{
		Pid:     serverPid,
		Comm:    "myv6_service",
		Exe:     "/usr/bin/myv6_service",
		Cmdline: []string{"myv6_service"},
	}
	fake.procs[unnamedPid] = quark.Process{
		Pid: unnamedPid,
		// A comm of 15 bytes may have been truncated by the kernel,
		// so the name must come from argv[0].
		Comm:    "verylongprocess",
		Cmdline: []string{"/opt/bin/verylongprocessname", "-d"},
	}

	// A closed short-lived IPv4 connection, still within quark's cache
	// grace time. This is the elastic/beats#51125 regression case: the
	// socket is gone from the OS socket table (the legacy mock below
	// does not know it), but quark still resolves it.
	fake.addSocket(
		netip.MustParseAddrPort("192.168.1.1:34000"),
		netip.MustParseAddrPort("93.184.216.34:80"),
		quark.Socket{
			PidOrigin:       curlPid,
			PidLastUse:      curlPid,
			EstablishedTime: 1000,
			CloseTime:       2000,
		},
	)
	// An open IPv6 connection on the accept side, with PidLastUse
	// unset so that PidOrigin is used.
	fake.addSocket(
		netip.MustParseAddrPort("[7777::33]:443"),
		netip.MustParseAddrPort("[2001:db8::2]:56000"),
		quark.Socket{
			PidOrigin:       serverPid,
			EstablishedTime: 1000,
		},
	)
	// A TCP connection owned by a process with a possibly truncated comm.
	fake.addSocket(
		netip.MustParseAddrPort("192.168.1.1:35000"),
		netip.MustParseAddrPort("93.184.216.34:80"),
		quark.Socket{
			PidOrigin:  unnamedPid,
			PidLastUse: unnamedPid,
		},
	)
	// A connection quark knows, but whose process it does not: the
	// lookup must fail over to the legacy path (which does not know
	// the port either, so enrichment yields nothing).
	fake.addSocket(
		netip.MustParseAddrPort("192.168.1.1:36000"),
		netip.MustParseAddrPort("93.184.216.34:80"),
		quark.Socket{PidOrigin: 999999},
	)

	legacy := newMockWatcher(
		[]net.IP{
			net.ParseIP("127.0.0.1"),
			net.ParseIP("192.168.1.1"),
			net.ParseIP("7777::33"),
		},
		[]runningProcess{
			{
				process: process{name: "legacy_service", pid: legacyPid},
				ports:   []endpoint{{address: "192.168.1.1", port: 38000}},
				proto:   applayer.TransportTCP,
			},
			{
				process: process{name: "udp_service", pid: udpPid},
				ports:   []endpoint{{address: "192.168.1.1", port: 34000}},
				proto:   applayer.TransportUDP,
			},
		},
	)

	config := ProcsConfig{
		Enabled: true,
		Monitored: []ProcConfig{
			{Process: "Curl", CmdlineGrep: "curl"},
		},
	}
	proc := &ProcessesWatcher{}
	require.NoError(t, proc.init(config, legacy, logger))
	proc.kernelTracing = newQuarkWatcher(fake, logger)
	defer proc.Close()

	wallClockStart := time.Unix(0, int64(quark.Boottime()+123456))

	for _, tc := range []struct {
		name      string
		tuple     common.IPPortTuple
		transport string
		src, dst  common.Process
	}{
		{
			name: "closed short-lived connection resolved by quark, name aliased by cmdline_grep",
			tuple: common.IPPortTuple{
				BaseTuple: common.BaseTuple{
					SrcIP: net.ParseIP("192.168.1.1"), SrcPort: 34000,
					DstIP: net.ParseIP("93.184.216.34"), DstPort: 80,
				},
			},
			transport: "tcp",
			src: common.Process{
				PID:       curlPid,
				PPID:      curlPpid,
				Name:      "Curl",
				Args:      []string{"curl", "http://elastic.co/"},
				Exe:       "/usr/bin/curl",
				CWD:       "/home/user",
				StartTime: wallClockStart,
			},
		},
		{
			name: "IPv6 connection resolved via PidOrigin",
			tuple: common.IPPortTuple{
				BaseTuple: common.BaseTuple{
					SrcIP: net.ParseIP("2001:db8::2"), SrcPort: 56000,
					DstIP: net.ParseIP("7777::33"), DstPort: 443,
				},
			},
			transport: "tcp",
			dst: common.Process{
				PID:  serverPid,
				Name: "myv6_service",
				Args: []string{"myv6_service"},
				Exe:  "/usr/bin/myv6_service",
			},
		},
		{
			name: "truncated comm falls back to argv[0] basename",
			tuple: common.IPPortTuple{
				BaseTuple: common.BaseTuple{
					SrcIP: net.ParseIP("192.168.1.1"), SrcPort: 35000,
					DstIP: net.ParseIP("93.184.216.34"), DstPort: 80,
				},
			},
			transport: "tcp",
			src: common.Process{
				PID:  unnamedPid,
				Name: "verylongprocessname",
				Args: []string{"/opt/bin/verylongprocessname", "-d"},
			},
		},
		{
			name: "connection unknown to quark falls back to the legacy path",
			tuple: common.IPPortTuple{
				BaseTuple: common.BaseTuple{
					SrcIP: net.ParseIP("192.168.1.1"), SrcPort: 38000,
					DstIP: net.ParseIP("93.184.216.34"), DstPort: 80,
				},
			},
			transport: "tcp",
			src:       common.Process{PID: legacyPid, Name: "legacy_service"},
		},
		{
			name: "socket known to quark without process falls back to the legacy path",
			tuple: common.IPPortTuple{
				BaseTuple: common.BaseTuple{
					SrcIP: net.ParseIP("192.168.1.1"), SrcPort: 36000,
					DstIP: net.ParseIP("93.184.216.34"), DstPort: 80,
				},
			},
			transport: "tcp",
			src:       common.Process{},
		},
		{
			name: "UDP is never resolved by quark",
			tuple: common.IPPortTuple{
				BaseTuple: common.BaseTuple{
					SrcIP: net.ParseIP("192.168.1.1"), SrcPort: 34000,
					DstIP: net.ParseIP("93.184.216.34"), DstPort: 80,
				},
			},
			transport: "udp",
			src:       common.Process{PID: udpPid, Name: "udp_service"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got *common.ProcessTuple
			if tc.transport == "tcp" {
				got = proc.FindProcessesTupleTCP(&tc.tuple)
			} else {
				got = proc.FindProcessesTupleUDP(&tc.tuple)
			}
			assert.Equal(t, tc.src, got.Src, "src process")
			assert.Equal(t, tc.dst, got.Dst, "dst process")
		})
	}
}

func TestQuarkWatcherClose(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "procs")

	fake := newFakeQuarkQueue()
	fake.procs[1234] = quark.Process{Pid: 1234, Comm: "closer"}
	local := netip.MustParseAddrPort("127.0.0.1:34000")
	remote := netip.MustParseAddrPort("127.0.0.1:80")
	fake.addSocket(local, remote, quark.Socket{PidOrigin: 1234})

	w := newQuarkWatcher(fake, logger)
	p := w.findProcTuple(net.ParseIP("127.0.0.1"), 34000, net.ParseIP("127.0.0.1"), 80)
	require.NotNil(t, p)
	assert.Equal(t, 1234, p.pid)

	w.close()
	// The pump closes the queue and marks the watcher closed.
	select {
	case <-fake.closed:
	case <-time.After(10 * time.Second):
		t.Fatal("quark queue was not closed")
	}
	require.Eventually(t, func() bool {
		return w.findProcTuple(net.ParseIP("127.0.0.1"), 34000, net.ParseIP("127.0.0.1"), 80) == nil
	}, 10*time.Second, 10*time.Millisecond, "lookups must fail after close")
}
