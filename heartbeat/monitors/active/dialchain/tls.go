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
	"net"
	"time"

	"github.com/elastic/beats/heartbeat/look"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/outputs/transport"
)

// TLSLayer configures the TLS layer in a DialerChain.
//
// The layer will update the active event with:
//
//  {
//    "tls": {
//        "rtt": { "handshake": { "us": ... }}
//    }
//  }
func TLSLayer(cfg *transport.TLSConfig, to time.Duration) Layer {
	return func(event common.MapStr, next transport.Dialer) (transport.Dialer, error) {
		var timer timer

		// Wrap next dialer so to start the timer when 'next' returns.
		// This gets us the timestamp for when the TLS layer will start the handshake.
		next = startTimerAfterDial(&timer, next)

		dialer, err := transport.TLSDialer(next, cfg, to)
		if err != nil {
			return nil, err
		}

		return afterDial(dialer, func(conn net.Conn) (net.Conn, error) {
			// TODO: extract TLS connection parameters from connection object.

			timer.stop()
			event.Put("tls.rtt.handshake", look.RTT(timer.duration()))
			return conn, nil
		}), nil
	}
}
