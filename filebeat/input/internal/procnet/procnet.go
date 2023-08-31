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

// Package procnet provides support for obtaining and formatting /proc/net
// network addresses for linux systems.
package procnet

import (
	"fmt"
	"net"
	"strconv"

	"github.com/elastic/elastic-agent-libs/logp"
)

// Addrs returns the linux /proc/net/tcp or /proc/net/udp addresses for the
// provided host address, addr. addr is a host:port address and host may be
// an IPv4 or IPv6 address, or an FQDN for the host. The returned slices
// contain the string representations of the address as they would appear in
// the /proc/net tables.
func Addrs(addr string, log *logp.Logger) (addr4, addr6 []string, err error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get address for %s: could not split host and port: %w", addr, err)
	}
	ip, err := net.LookupIP(host)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get address for %s: %w", addr, err)
	}
	pn, err := strconv.ParseInt(port, 10, 16)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get port for %s: %w", addr, err)
	}
	addr4 = make([]string, 0, len(ip))
	addr6 = make([]string, 0, len(ip))
	for _, p := range ip {
		// Ensure the length of the net.IP is canonicalised to the standard
		// length for the format, as the net package may return IPv4 addresses
		// as the IPv6 form ::ffff:wwxxyyzz. So if we only compare len(p) to
		// the len constants all addresses may appear to be IPv6.
		switch {
		case len(p.To4()) == net.IPv4len:
			addr4 = append(addr4, IPv4(p, int(pn)))
		case len(p.To16()) == net.IPv6len:
			addr6 = append(addr6, IPv6(p, int(pn)))
		default:
			log.Warnf("unexpected addr length %d for %s", len(p), p)
		}
	}
	return addr4, addr6, nil
}

// IPv4 returns the string representation of an IPv4 address in a /proc/net table.
func IPv4(ip net.IP, port int) string {
	return fmt.Sprintf("%08X:%04X", reverse(ip.To4()), port)
}

// IPv6 returns the string representation of an IPv6 address in a /proc/net table.
func IPv6(ip net.IP, port int) string {
	return fmt.Sprintf("%032X:%04X", reverse(ip.To16()), port)
}

func reverse(b []byte) []byte {
	c := make([]byte, len(b))
	for i, e := range b {
		c[len(b)-1-i] = e
	}
	return c
}
