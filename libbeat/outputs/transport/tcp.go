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

package transport

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/testing"
)

func NetDialer(timeout time.Duration) Dialer {
	return TestNetDialer(testing.NullDriver, timeout)
}

func TestNetDialer(d testing.Driver, timeout time.Duration) Dialer {
	return DialerFunc(func(network, address string) (net.Conn, error) {
		switch network {
		case "tcp", "tcp4", "tcp6", "udp", "udp4", "udp6":
		default:
			d.Fatal("network type", fmt.Errorf("unsupported network type %v", network))
			return nil, fmt.Errorf("unsupported network type %v", network)
		}

		host, port, err := net.SplitHostPort(address)
		d.Fatal("parse host", err)
		if err != nil {
			return nil, err
		}
		addresses, err := net.LookupHost(host)
		d.Fatal("dns lookup", err)
		d.Info("addresses", strings.Join(addresses, ", "))
		if err != nil {
			logp.Warn(`DNS lookup failure "%s": %v`, host, err)
			return nil, err
		}

		// dial via host IP by randomized iteration of known IPs
		dialer := &net.Dialer{Timeout: timeout}
		return DialWith(dialer, network, host, addresses, port)
	})
}

// UnixDialer creates a Unix Dialer when using unix domain socket.
func UnixDialer(timeout time.Duration, sockFile string) Dialer {
	return TestUnixDialer(testing.NullDriver, timeout, sockFile)
}

// TestUnixDialer creates a Test Unix Dialer when using domain socket.
func TestUnixDialer(d testing.Driver, timeout time.Duration, sockFile string) Dialer {
	return DialerFunc(func(network, address string) (net.Conn, error) {
		d.Info("connecting using unix domain socket", sockFile)
		return net.DialTimeout("unix", sockFile, timeout)
	})
}
