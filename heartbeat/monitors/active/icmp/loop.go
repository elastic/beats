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
	"errors"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"

	"github.com/elastic/beats/libbeat/logp"
)

type icmpLoop struct {
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
	l      *icmpLoop
	id     requestID
	ts     time.Time
	result chan requestResult
}

type requestResult struct {
	packet packet
	err    error
}

var (
	loopInit sync.Once
	loop     *icmpLoop
)

func newICMPLoop() (*icmpLoop, error) {
	// Log errors at info level, as the loop is setup globally when ICMP module is loaded
	// first (not yet configured).
	// With multiple configurations using the icmp loop, we have to postpose
	// IPv4/IPv6 checking
	conn4 := createListener("IPv4", "ip4:icmp")
	conn6 := createListener("IPv6", "ip6:ipv6-icmp")

	l := &icmpLoop{
		conn4:    conn4,
		conn6:    conn6,
		recv:     make(chan packet, 16),
		requests: map[requestID]*requestContext{},
	}

	if conn4 != nil {
		go l.runICMPRecv(conn4, protocolICMP)
	}
	if conn6 != nil {
		go l.runICMPRecv(conn6, protocolIPv6ICMP)
	}

	return l, nil
}

func (l *icmpLoop) checkNetworkMode(mode string) error {
	ip4, ip6 := false, false
	switch mode {
	case "ip4":
		ip4 = true
	case "ip6":
		ip6 = true
	case "ip":
		ip4, ip6 = true, true
	default:
		return fmt.Errorf("'%v' is not supported", mode)
	}

	if ip4 && l.conn4 == nil {
		return errors.New("failed to initiate IPv4 support")
	}
	if ip6 && l.conn6 == nil {
		return errors.New("failed to initiate IPv6 support")
	}

	return nil
}

func (l *icmpLoop) runICMPRecv(conn *icmp.PacketConn, proto int) {
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

func (l *icmpLoop) ping(
	addr *net.IPAddr,
	timeout time.Duration,
	interval time.Duration,
) (time.Duration, int, error) {

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
			// and remove all requests from request table.
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

func (l *icmpLoop) sendEchoRequest(addr *net.IPAddr) (*requestContext, error) {
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
		return nil, err
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
		logp.Info("%v ICMP not supported: %v", name, err)
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
