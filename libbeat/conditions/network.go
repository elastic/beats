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

package conditions

import (
	"fmt"
	"net"
	"strings"

	"github.com/pkg/errors"

	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/logp"
)

var (
	// RFC 1918
	privateIPv4 = []net.IPNet{
		{IP: net.IPv4(10, 0, 0, 0), Mask: net.IPv4Mask(255, 0, 0, 0)},
		{IP: net.IPv4(172, 16, 0, 0), Mask: net.IPv4Mask(255, 240, 0, 0)},
		{IP: net.IPv4(192, 168, 0, 0), Mask: net.IPv4Mask(255, 255, 0, 0)},
	}

	// RFC 4193
	privateIPv6 = net.IPNet{
		IP:   net.IP{0xfd, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		Mask: net.IPMask{0xff, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	}

	namedNetworks = map[string]netContainsFunc{
		"loopback":                  func(ip net.IP) bool { return ip.IsLoopback() },
		"global_unicast":            func(ip net.IP) bool { return ip.IsGlobalUnicast() },
		"unicast":                   func(ip net.IP) bool { return ip.IsGlobalUnicast() },
		"link_local_unicast":        func(ip net.IP) bool { return ip.IsLinkLocalUnicast() },
		"interface_local_multicast": func(ip net.IP) bool { return ip.IsInterfaceLocalMulticast() },
		"link_local_multicast":      func(ip net.IP) bool { return ip.IsLinkLocalMulticast() },
		"multicast":                 func(ip net.IP) bool { return ip.IsMulticast() },
		"unspecified":               func(ip net.IP) bool { return ip.IsUnspecified() },
		"private":                   isPrivateNetwork,
		"public":                    func(ip net.IP) bool { return !isLocalOrPrivate(ip) },
	}
)

// Network is a condition that tests if an IP address is in a network range.
type Network struct {
	fields map[string]networkMatcher
	log    *logp.Logger
}

type networkMatcher interface {
	fmt.Stringer
	Contains(net.IP) bool
}

type netContainsFunc func(net.IP) bool

type singleNetworkMatcher struct {
	name string
	netContainsFunc
}

func (m singleNetworkMatcher) Contains(ip net.IP) bool { return m.netContainsFunc(ip) }
func (m singleNetworkMatcher) String() string          { return m.name }

type multiNetworkMatcher []networkMatcher

func (m multiNetworkMatcher) Contains(ip net.IP) bool {
	for _, network := range m {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func (m multiNetworkMatcher) String() string {
	var names []string
	for _, network := range m {
		names = append(names, network.String())
	}
	return strings.Join(names, " OR ")
}

// NewNetworkCondition builds a new Network using the given configuration.
func NewNetworkCondition(fields map[string]interface{}) (*Network, error) {
	cond := &Network{
		fields: map[string]networkMatcher{},
		log:    logp.NewLogger(logName),
	}

	makeMatcher := func(network string) (networkMatcher, error) {
		m := singleNetworkMatcher{name: network, netContainsFunc: namedNetworks[network]}
		if m.netContainsFunc == nil {
			subnet, err := parseCIDR(network)
			if err != nil {
				return nil, err
			}
			m.netContainsFunc = subnet.Contains
		}
		return m, nil
	}

	invalidTypeError := func(field string, value interface{}) error {
		return fmt.Errorf("network condition attempted to set "+
			"'%v' -> '%v' and encountered unexpected type '%T', only "+
			"strings or []strings are allowed", field, value, value)
	}

	for field, value := range common.MapStr(fields).Flatten() {
		switch v := value.(type) {
		case string:
			m, err := makeMatcher(v)
			if err != nil {
				return nil, err
			}
			cond.fields[field] = m
		case []interface{}:
			var matchers multiNetworkMatcher
			for _, networkIfc := range v {
				network, ok := networkIfc.(string)
				if !ok {
					return nil, invalidTypeError(field, networkIfc)
				}
				m, err := makeMatcher(network)
				if err != nil {
					return nil, err
				}
				matchers = append(matchers, m)
			}
			cond.fields[field] = matchers
		default:
			return nil, invalidTypeError(field, value)
		}
	}

	return cond, nil
}

// Check determines whether the given event matches this condition.
func (c *Network) Check(event ValuesMap) bool {
	for field, network := range c.fields {
		value, err := event.GetValue(field)
		if err != nil {
			return false
		}

		ip := extractIP(value)
		if ip == nil {
			c.log.Debugf("Invalid IP address in field=%v for network condition", field)
			return false
		}

		if !network.Contains(ip) {
			return false
		}
	}

	return true
}

// String returns a string representation of the Network condition.
func (c *Network) String() string {
	var sb strings.Builder
	sb.WriteString("network:(")
	var i int
	for field, network := range c.fields {
		sb.WriteString(field)
		sb.WriteString(":")
		sb.WriteString(network.String())
		if i < len(c.fields)-1 {
			sb.WriteString(" AND ")
		}
		i++
	}
	sb.WriteString(")")
	return sb.String()
}

// parseCIDR parses a network CIDR.
func parseCIDR(value string) (*net.IPNet, error) {
	_, mask, err := net.ParseCIDR(value)
	return mask, errors.Wrap(err, "failed to parse CIDR, values must be "+
		"an IP address and prefix length, like '192.0.2.0/24' or "+
		"'2001:db8::/32', as defined in RFC 4632 and RFC 4291.")
}

// extractIP return an IP address if unk is an IP address string or a net.IP.
// Otherwise it returns nil.
func extractIP(unk interface{}) net.IP {
	switch v := unk.(type) {
	case string:
		return net.ParseIP(v)
	case net.IP:
		return v
	default:
		return nil
	}
}

func isPrivateNetwork(ip net.IP) bool {
	for _, net := range privateIPv4 {
		if net.Contains(ip) {
			return true
		}
	}

	return privateIPv6.Contains(ip)
}

func isLocalOrPrivate(ip net.IP) bool {
	return isPrivateNetwork(ip) ||
		ip.IsLoopback() ||
		ip.IsUnspecified() ||
		ip.Equal(net.IPv4bcast) ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast()
}

// NetworkContains returns true if the given IP is contained by any of the
// networks. networks can be a CIDR or any of these named networks:
//   - loopback
//   - global_unicast
//   - unicast
//   - link_local_unicast
//   - interface_local_multicast
//   - link_local_multicast
//   - multicast
//   - unspecified
//   - private
//   - public
func NetworkContains(ip net.IP, networks ...string) (bool, error) {
	for _, net := range networks {
		contains, found := namedNetworks[net]
		if !found {
			subnet, err := parseCIDR(net)
			if err != nil {
				return false, err
			}
			contains = subnet.Contains
		}

		if contains(ip) {
			return true, nil
		}
	}
	return false, nil
}
