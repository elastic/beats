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
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/transport"

	"github.com/elastic/beats/v7/heartbeat/ecserr"
	"github.com/elastic/beats/v7/heartbeat/eventext"
	"github.com/elastic/beats/v7/heartbeat/look"
	"github.com/elastic/beats/v7/libbeat/beat"
)

// TCPDialer creates a new NetDialer with constant event fields and default
// connection timeout.
// The fields parameter holds additional constants to be added to the final
// event structure.
//
// The dialer will update the active events with:
//
//	{
//	  "tcp": {
//	    "port": ...,
//	    "rtt": { "connect": { "us": ... }}
//	  }
//	}
func TCPDialer(to time.Duration) NetDialer {
	return CreateNetDialer(to)
}

// UDPDialer creates a new NetDialer with constant event fields and default
// connection timeout.
// The fields parameter holds additional constants to be added to the final
// event structure.
//
// The dialer will update the active events with:
//
//	{
//	  "udp": {
//	    "port": ...,
//	    "rtt": { "connect": { "us": ... }}
//	  }
//	}
func UDPDialer(to time.Duration) NetDialer {
	return CreateNetDialer(to)
}

// CreateNetDialer returns a NetDialer with the given timeout.
func CreateNetDialer(timeout time.Duration) NetDialer {
	return func(event *beat.Event) (transport.Dialer, error) {
		return makeDialer(func(network, address string) (net.Conn, error) {
			var namespace string

			switch network {
			case "tcp", "tcp4", "tcp6":
				namespace = "tcp"
			case "udp", "udp4", "udp6":
				namespace = "udp"
			default:
				return nil, fmt.Errorf("unsupported network type %v", network)
			}

			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}

			portNum, err := strconv.Atoi(port)
			if err != nil || portNum < 0 || portNum > (1<<16) {
				return nil, fmt.Errorf("invalid port number '%v' used", port)
			}

			addresses, err := net.LookupHost(host)
			if err != nil {
				return nil, ecserr.NewDNSLookupFailedErr(host, err)
			}

			// dial via host IP by randomized iteration of known IPs
			dialer := &net.Dialer{Timeout: timeout}

			start := time.Now()
			conn, err := transport.DialWith(dialer, network, host, addresses, port)
			if err != nil {
				return nil, ecserr.NewCouldNotConnectErr(host, port, err)
			}

			end := time.Now()
			ef := mapstr.M{}
			ef[namespace] = mapstr.M{
				"rtt": mapstr.M{
					"connect": look.RTT(end.Sub(start)),
				},
			}
			eventext.MergeEventFields(event, ef)

			return conn, nil
		}), nil
	}
}
