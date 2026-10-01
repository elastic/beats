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

package dgram

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/filebeat/inputsource"
	"github.com/elastic/beats/v7/pkg/logp"
)

var errClosed = &net.OpError{Op: "read", Err: errors.New("use of closed network connection")}

type fakePacketConn struct {
	payloads  [][]byte
	repeat    []byte
	remaining int
	addr      net.Addr
}

func (c *fakePacketConn) ReadFrom(p []byte) (int, net.Addr, error) {
	if len(c.payloads) > 0 {
		n := copy(p, c.payloads[0])
		c.payloads = c.payloads[1:]
		return n, c.addr, nil
	}
	if c.remaining > 0 {
		c.remaining--
		return copy(p, c.repeat), c.addr, nil
	}
	return 0, nil, errClosed
}

func (c *fakePacketConn) WriteTo(p []byte, _ net.Addr) (int, error) { return len(p), nil }
func (c *fakePacketConn) Close() error                              { return nil }
func (c *fakePacketConn) LocalAddr() net.Addr                       { return c.addr }
func (c *fakePacketConn) SetDeadline(time.Time) error               { return nil }
func (c *fakePacketConn) SetReadDeadline(time.Time) error           { return nil }
func (c *fakePacketConn) SetWriteDeadline(time.Time) error          { return nil }

func TestDatagramReaderCallbackDataIsIndependent(t *testing.T) {
	first := []byte("first datagram payload")
	second := []byte("second")
	conn := &fakePacketConn{
		payloads: [][]byte{first, second},
		addr:     &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 514},
	}

	var received [][]byte
	handler := DatagramReaderFactory(inputsource.FamilyUDP, logp.NewNopLogger(),
		func(data []byte, _ inputsource.NetworkMetadata) {
			received = append(received, data)
		},
	)(ListenerConfig{MaxMessageSize: 1024})

	require.NoError(t, handler(t.Context(), conn), "handler must exit cleanly when the connection closes")
	require.Len(t, received, 2, "callback must be invoked once per datagram")
	assert.Equal(t, first, received[0], "first datagram must not be overwritten by a later read")
	assert.Equal(t, second, received[1], "second datagram must match what was sent")
}

func BenchmarkDatagramReader(b *testing.B) {
	const maxMessageSize = 10 * 1024
	for _, size := range []int{128, 1024, maxMessageSize} {
		b.Run(fmt.Sprintf("payload=%d", size), func(b *testing.B) {
			conn := &fakePacketConn{
				repeat:    bytes.Repeat([]byte("a"), size),
				remaining: b.N,
				addr:      &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 514},
			}

			var got int
			handler := DatagramReaderFactory(inputsource.FamilyUDP, logp.NewNopLogger(),
				func(data []byte, _ inputsource.NetworkMetadata) {
					got += len(data)
				},
			)(ListenerConfig{MaxMessageSize: maxMessageSize})

			b.ReportAllocs()
			b.SetBytes(int64(size))
			b.ResetTimer()
			err := handler(b.Context(), conn)
			b.StopTimer()
			require.NoError(b, err, "handler must exit cleanly when the connection closes")
			require.Equal(b, size*b.N, got, "callback must receive every byte of every datagram")
		})
	}
}
