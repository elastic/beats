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

//go:build linux && (amd64 || arm64) && cgo

package procs

import (
	"fmt"
	"net"
	"net/netip"
	"path/filepath"
	"sync"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
	quark "github.com/elastic/go-quark"
)

const (
	// quarkHoldTime is how long quark buffers kernel events for
	// ordering and aggregation before they reach its caches.
	// Packetbeat only consumes the caches, so it keeps this short to
	// make new connections visible quickly. Lookups that race this
	// window fall back to the procfs path, which works while the
	// socket is still open.
	quarkHoldTime = 250 * time.Millisecond
)

// quarkQueue is the subset of *quark.Queue used by quarkWatcher. It
// allows a fake queue to be injected for testing.
type quarkQueue interface {
	GetEvent() (quark.Event, bool)
	Block() error
	Lookup(pid int) (quark.Process, bool)
	SocketLookup(local, remote netip.AddrPort) (quark.Socket, bool)
	Close()
}

// quarkWatcher resolves TCP tuples to processes using quark's socket
// and process caches, which are populated from kernel events. Unlike
// the procfs path, it can resolve a connection for the configured
// grace time after its socket closed, covering short-lived connections.
type quarkWatcher struct {
	// mu serializes access to queue: quark queues are not safe for
	// concurrent use, and all state, including the caches consulted
	// by findProcTuple, is mutated inside GetEvent.
	mu     sync.Mutex
	queue  quarkQueue
	closed bool
	done   chan struct{}
	logger *logp.Logger
}

// newTupleWatcher opens a quark queue with socket tracking and starts
// the goroutine that drives it. Closed sockets and exited processes
// stay resolvable for grace. Socket tracking requires quark's eBPF
// backend; there is no kprobe fallback.
func newTupleWatcher(grace time.Duration, logger *logp.Logger) (tupleWatcher, error) {
	attr := quark.DefaultQueueAttr() // The default flags select the eBPF backend.
	attr.Flags |= quark.QQ_SOCK_CONN
	attr.CacheGraceTime = int(grace / time.Millisecond)
	attr.HoldTime = int(quarkHoldTime / time.Millisecond)
	queue, err := quark.OpenQueue(attr)
	if err != nil {
		return nil, fmt.Errorf("cannot open quark queue (requires eBPF): %w", err)
	}
	logger.Debugf("procs: quark queue opened with %v grace time", grace)
	return newQuarkWatcher(queue, logger), nil
}

// newQuarkWatcher starts a watcher driving queue. It is split from
// newTupleWatcher so tests can inject a fake queue.
func newQuarkWatcher(queue quarkQueue, logger *logp.Logger) *quarkWatcher {
	w := &quarkWatcher{
		queue:  queue,
		done:   make(chan struct{}),
		logger: logger,
	}
	go w.pump()
	return w
}

// pump drives the quark queue until the watcher is closed. The events
// themselves are discarded; GetEvent is called for its side effect of
// updating quark's socket and process caches.
func (w *quarkWatcher) pump() {
	defer func() {
		w.mu.Lock()
		w.queue.Close()
		w.closed = true
		w.mu.Unlock()
	}()
	for {
		select {
		case <-w.done:
			return
		default:
		}
		w.mu.Lock()
		_, ok := w.queue.GetEvent()
		w.mu.Unlock()
		if !ok {
			if err := w.queue.Block(); err != nil {
				w.logger.Errorf("procs: quark block failed, kernel_tracing process enrichment stops: %v", err)
				return
			}
		}
	}
}

// close stops the watcher. Lookups made after close return nil.
func (w *quarkWatcher) close() {
	close(w.done)
}

// findProcTuple returns process information for the TCP connection
// with the given local and remote endpoints, or nil if quark does not
// know the connection. Closed connections remain resolvable for the
// grace time given to newTupleWatcher.
func (w *quarkWatcher) findProcTuple(localIP net.IP, localPort uint16, remoteIP net.IP, remotePort uint16) *process {
	local, ok := addrPort(localIP, localPort)
	if !ok {
		return nil
	}
	remote, ok := addrPort(remoteIP, remotePort)
	if !ok {
		return nil
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return nil
	}
	socket, ok := w.queue.SocketLookup(local, remote)
	if !ok {
		return nil
	}
	pid := socket.PidLastUse
	if pid == 0 {
		pid = socket.PidOrigin
	}
	if pid == 0 {
		return nil
	}
	qp, ok := w.queue.Lookup(int(pid))
	if !ok {
		return nil
	}
	return processFromQuark(qp)
}

func addrPort(ip net.IP, port uint16) (netip.AddrPort, bool) {
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return netip.AddrPort{}, false
	}
	// net.IP usually holds IPv4 addresses in 16-byte form, which
	// AddrFromSlice turns into an IPv4-mapped IPv6 address. Unmap so
	// quark gets a plain IPv4 endpoint; it tries the AF_INET key
	// first and the IPv4-mapped AF_INET6 key (dual-stack sockets)
	// second.
	return netip.AddrPortFrom(addr.Unmap(), port), true
}

func processFromQuark(qp quark.Process) *process {
	p := &process{
		pid:  int(qp.Pid),
		name: quarkProcName(qp),
		exe:  qp.Exe,
		cwd:  qp.Cwd,
		args: qp.Cmdline,
	}
	if qp.Proc.Valid {
		p.ppid = int(qp.Proc.Ppid)
		// TimeBoot is nanoseconds since boot; Boottime is the boot
		// instant in nanoseconds since the Unix epoch.
		p.startTime = time.Unix(0, int64(quark.Boottime()+qp.Proc.TimeBoot)) //nolint:gosec // The sum cannot overflow int64 for plausible times.
	}
	return p
}

// quarkProcName mirrors procName in procs_linux.go: the kernel
// truncates comm to 15 bytes (TASK_COMM_LEN minus the NUL), so prefer
// the basename of argv[0] when comm may be truncated.
func quarkProcName(qp quark.Process) string {
	if len(qp.Comm) >= 15 && len(qp.Cmdline) > 0 && qp.Cmdline[0] != "" {
		return filepath.Base(qp.Cmdline[0])
	}
	return qp.Comm
}
