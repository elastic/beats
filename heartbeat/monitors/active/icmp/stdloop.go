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

package icmp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/rand"
	"net"
	"os"
	"runtime"
	"sync"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

type stdICMPLoop struct {
	conn4, conn6 *icmp.PacketConn
	recv         chan packet

	mutex    sync.Mutex
	requests map[requestID]*requestContext
}

type timeoutError struct {
}

const (
	// iana types
	protocolICMP     = 1
	protocolIPv6ICMP = 58
)

type packet struct {
	ts   time.Time
	addr net.Addr

	Type     icmp.Type // type, either ipv4.ICMPType or ipv6.ICMPType
	Code     int       // code
	Checksum int       // checksum
	Echo     icmp.Echo
}

type requestID struct {
	addr  string
	proto int
	id    int
	seq   int
}

type requestContext struct {
	l      *stdICMPLoop
	id     requestID
	ts     time.Time
	result chan requestResult
}

type requestResult struct {
	packet packet
	err    error
}

// stdLoop is a singleton for our main ICMP loop since it doesn't
// make sense to have multiples. While having a singleton is ugly
// is mandatory for the ICMP interface in go, where all monitors
// must share a single loop.
// These vars should not be used directly, but rather getStdLoop
// should be invoked to initialize and return stdLoop.
var (
	stdICMPLoopInit      sync.Mutex
	stdICMPLoopSingleton *stdICMPLoop
)

func getStdLoop() (*stdICMPLoop, error) {
	stdICMPLoopInit.Lock()
	defer stdICMPLoopInit.Unlock()

	if stdICMPLoopSingleton == nil {
		debugf("initializing ICMP loop")
		singleton, err := newICMPLoop()
		if err != nil {
			return nil, err
		}
		stdICMPLoopSingleton = singleton
		debugf("ICMP loop successfully initialized")
	}

	return stdICMPLoopSingleton, nil
}

func noPingCapabilityError(message string) error {
	return fmt.Errorf(fmt.Sprintf("Insufficient privileges to perform ICMP ping. %s", message))
}

func newICMPLoop() (*stdICMPLoop, error) {
	// Log errors at info level, as the loop is setup globally when ICMP module is loaded
	// first (not yet configured).
	// With multiple configurations using the icmp loop, we have to postpose
	// IPv4/IPv6 checking
	conn4 := createListener("IPv4", "ip4:icmp")
	conn6 := createListener("IPv6", "ip6:ipv6-icmp")
	unprivilegedPossible := false
	l := &stdICMPLoop{
		conn4:    conn4,
		conn6:    conn6,
		recv:     make(chan packet, 16),
		requests: map[requestID]*requestContext{},
	}

	if l.conn4 == nil && l.conn6 == nil {
		switch runtime.GOOS {
		case "linux", "darwin":
			unprivilegedPossible = true
			//This is non-privileged ICMP, not udp
			l.conn4 = createListener("Unprivileged IPv4", "udp4")
			l.conn6 = createListener("Unprivileged IPv6", "udp6")
		}
	}

	if l.conn4 != nil {
		go l.runICMPRecv(l.conn4, protocolICMP)
	}
	if l.conn6 != nil {
		go l.runICMPRecv(l.conn6, protocolIPv6ICMP)
	}

	if l.conn4 == nil && l.conn6 == nil {
		if unprivilegedPossible {
			var buffer bytes.Buffer
			path, _ := os.Executable()
			buffer.WriteString("You can run without root by setting cap_net_raw:\n sudo setcap cap_net_raw+eip ")
			buffer.WriteString(path + " \n")
			buffer.WriteString("Your system allows the use of unprivileged ping by setting net.ipv4.ping_group_range \n sysctl -w net.ipv4.ping_group_range='<min-uid> <max-uid>' ")
			return nil, noPingCapabilityError(buffer.String())
		}
		return nil, noPingCapabilityError("You must provide the appropriate permissions to this executable")
	}

	return l, nil
}

func (l *stdICMPLoop) runICMPRecv(conn *icmp.PacketConn, proto int) {
	for {
		bytes := make([]byte, 512)
		conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		_, addr, err := conn.ReadFrom(bytes)
		if err != nil {
			if neterr, ok := err.(*net.OpError); ok {
				if neterr.Timeout() {
					continue
				} else {
					// TODO: report error and quit loop?
					return
				}
			}
		}

		ts := time.Now()
		m, err := icmp.ParseMessage(proto, bytes)
		if err != nil {
			continue
		}

		// process echo reply only
		if m.Type != ipv4.ICMPTypeEchoReply && m.Type != ipv6.ICMPTypeEchoReply {
			continue
		}
		echo, ok := m.Body.(*icmp.Echo)
		if !ok {
			continue
		}

		id := requestID{
			addr:  addr.String(),
			proto: proto,
			id:    echo.ID,
			seq:   echo.Seq,
		}

		l.mutex.Lock()
		ctx := l.requests[id]
		if ctx != nil {
			delete(l.requests, id)
		}
		l.mutex.Unlock()

		// no return context available for echo reply -> handle next message
		if ctx == nil {
			continue
		}

		ctx.result <- requestResult{
			packet: packet{
				ts:   ts,
				addr: addr,

				Type:     m.Type,
				Code:     m.Code,
				Checksum: m.Checksum,
				Echo:     *echo,
			},
		}
	}
}

func (l *stdICMPLoop) ping(
	addr *net.IPAddr,
	timeout time.Duration,
	interval time.Duration,
) (time.Duration, int, error) {
	isIPv6 := addr.IP.To4() == nil
	if isIPv6 && l.conn6 == nil {
		return -1, -1, fmt.Errorf("cannot ping IPv6 address '%s', no IPv6 connection available", addr)
	}
	if !isIPv6 && l.conn4 == nil {
		return -1, -1, fmt.Errorf("cannot ping IPv4 address '%s', no IPv4 connection available", addr)
	}

	var err error
	toTimer := time.NewTimer(timeout)
	defer toTimer.Stop()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	done := false
	doneSignal := make(chan struct{})

	success := false
	var rtt time.Duration

	// results accepts first response received only
	results := make(chan time.Duration, 1)
	requests := 0

	awaitResponse := func(ctx *requestContext) {
		select {
		case <-doneSignal:
			ctx.Stop()

		case r := <-ctx.result:
			// ctx is removed from request tables automatically a response is
			// received. No need to stop it.

			// try to push RTT. The first result available will be reported
			select {
			case results <- r.packet.ts.Sub(ctx.ts):
			default:
			}
		}
	}

	for !done {
		var ctx *requestContext
		ctx, err = l.sendEchoRequest(addr)
		if err != nil {
			close(doneSignal)
			break
		}
		go awaitResponse(ctx)
		requests++

		select {
		case <-toTimer.C:
			// no response for any active request received. Finish loop
			// and remove all pingRequests from request table.
			done = true
			close(doneSignal)

		case <-ticker.C:
			// No response yet. Send another request with every tick

		case rtt = <-results:
			success = true

			done = true
			close(doneSignal)
		}
	}

	if err != nil {
		return 0, 0, err
	}

	if !success {
		return 0, requests, timeoutError{}
	}

	return rtt, requests, nil
}

func (l *stdICMPLoop) sendEchoRequest(addr *net.IPAddr) (*requestContext, error) {
	var conn *icmp.PacketConn
	var proto int
	var typ icmp.Type

	if l == nil {
		panic("icmp loop not initialized")
	}

	if isIPv4(addr.IP) {
		conn = l.conn4
		proto = protocolICMP
		typ = ipv4.ICMPTypeEcho
	} else if isIPv6(addr.IP) {
		conn = l.conn6
		proto = protocolIPv6ICMP
		typ = ipv6.ICMPTypeEchoRequest
	} else {
		return nil, fmt.Errorf("%v is unknown ip address", addr)
	}

	id := requestID{
		addr:  addr.String(),
		proto: proto,
		id:    rand.Intn(0xffff),
		seq:   rand.Intn(0xffff),
	}

	ctx := &requestContext{
		l:      l,
		id:     id,
		result: make(chan requestResult, 1),
	}

	l.mutex.Lock()
	l.requests[id] = ctx
	l.mutex.Unlock()

	payloadBuf := make([]byte, 0, 8)
	payload := bytes.NewBuffer(payloadBuf)
	ts := time.Now()
	binary.Write(payload, binary.BigEndian, ts.UnixNano())

	msg := &icmp.Message{
		Type: typ,
		Body: &icmp.Echo{
			ID:   id.id,
			Seq:  id.seq,
			Data: payload.Bytes(),
		},
	}
	encoded, _ := msg.Marshal(nil)

	_, err := conn.WriteTo(encoded, addr)
	if err != nil {
		return nil, fmt.Errorf("could not write to conn: %w", err)
	}

	ctx.ts = ts
	return ctx, nil
}

func createListener(name, network string) *icmp.PacketConn {
	conn, err := icmp.ListenPacket(network, "")

	// XXX: need to check for conn == nil, as 'err != nil' seems always to be
	//      true, even if error value itself is `nil`. Checking for conn suppresses
	//      misleading log message.
	if conn == nil && err != nil {
		return nil
	}
	return conn
}

// timeoutError implements net.Error interface
func (timeoutError) Error() string   { return "ping timeout" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }

func (r *requestContext) Stop() {
	r.l.mutex.Lock()
	delete(r.l.requests, r.id)
	r.l.mutex.Unlock()
}
