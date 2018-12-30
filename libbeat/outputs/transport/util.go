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
	"math/rand"
	"net"
	"strings"
)

func fullAddress(host string, defaultPort int) string {
	if _, _, err := net.SplitHostPort(host); err == nil {
		return host
	}

	idx := strings.Index(host, ":")
	if idx >= 0 {
		// IPv6 address detected
		return fmt.Sprintf("[%v]:%v", host, defaultPort)
	}
	return fmt.Sprintf("%v:%v", host, defaultPort)
}

// DialWith randomly dials one of a number of addresses with a given dialer.
//
// Use this to select and dial one IP being known for one host name.
func DialWith(
	dialer Dialer,
	network, host string,
	addresses []string,
	port string,
) (c net.Conn, err error) {
	switch len(addresses) {
	case 0:
		return nil, fmt.Errorf("no route to host %v", host)
	case 1:
		return dialer.Dial(network, net.JoinHostPort(addresses[0], port))
	}

	// Use randomization on DNS reported addresses combined with timeout and ACKs
	// to spread potential load when starting up large number of beats using
	// lumberjack.
	//
	// RFCs discussing reasons for ignoring order of DNS records:
	// http://www.ietf.org/rfc/rfc3484.txt
	// > is specific to locality-based address selection for multiple dns
	// > records, but exists as prior art in "Choose some different ordering for
	// > the dns records" done by a client
	//
	// https://tools.ietf.org/html/rfc1794
	// > "Clients, of course, may reorder this information" - with respect to
	// > handling order of dns records in a response.forwarded. Really required?
	for _, i := range rand.Perm(len(addresses)) {
		c, err = dialer.Dial(network, net.JoinHostPort(addresses[i], port))
		if err == nil && c != nil {
			return c, err
		}
	}

	if err == nil {
		err = fmt.Errorf("unable to connect to '%v'", host)
	}
	return nil, err
}
