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

package dialchain

import (
	cryptoTLS "crypto/tls"
	"fmt"
	"net"
	"time"

	"github.com/menderesk/beats/v7/heartbeat/monitors/active/dialchain/tlsmeta"
	"github.com/menderesk/beats/v7/libbeat/beat"
	"github.com/menderesk/beats/v7/libbeat/common/transport"
	"github.com/menderesk/beats/v7/libbeat/common/transport/tlscommon"
)

// TLSLayer configures the TLS layer in a DialerChain.
// The layer will update the active event with the TLS RTT and
// crypto/cert details.
func TLSLayer(cfg *tlscommon.TLSConfig, to time.Duration) Layer {
	return func(event *beat.Event, next transport.Dialer) (transport.Dialer, error) {
		var timer timer

		// Wrap next dialer so to start the timer when 'next' returns.
		// This gets us the timestamp for when the TLS layer will start the handshake.
		next = startTimerAfterDial(&timer, next)

		dialer := transport.TLSDialer(next, cfg, to)
		return afterDial(dialer, func(conn net.Conn) (net.Conn, error) {
			tlsConn, ok := conn.(*cryptoTLS.Conn)
			if !ok {
				panic(fmt.Sprintf("TLS afterDial received a non-tls connection %t. This should never happen", conn))
			}
			connState := tlsConn.ConnectionState()
			timer.stop()

			tlsmeta.AddTLSMetadata(event.Fields, connState, timer.duration())

			return conn, nil
		}), nil
	}
}
