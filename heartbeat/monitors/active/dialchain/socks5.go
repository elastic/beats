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

	"github.com/elastic/beats/v8/heartbeat/look"
	"github.com/elastic/beats/v8/libbeat/beat"
	"github.com/elastic/beats/v8/libbeat/common/transport"
	"github.com/elastic/beats/v8/libbeat/logp"
)

// SOCKS5Layer configures a SOCKS5 proxy layer in a DialerChain.
//
// The layer will update the active event with:
//
//  {
//    "socks5": {
//        "rtt": { "connect": { "us": ... }}
//    }
//  }
func SOCKS5Layer(config *transport.ProxyConfig) Layer {
	return func(event *beat.Event, next transport.Dialer) (transport.Dialer, error) {
		var timer timer

		dialer, err := transport.ProxyDialer(logp.NewLogger("socks5Layer"), config, startTimerAfterDial(&timer, next))
		if err != nil {
			return nil, err
		}

		return afterDial(dialer, func(conn net.Conn) (net.Conn, error) {
			// TODO: extract connection parameter from connection object?
			// TODO: add proxy url to event?

			timer.stop()
			event.Fields.Put("socks5.rtt.connect", look.RTT(timer.duration()))
			return conn, nil
		}), nil
	}
}
