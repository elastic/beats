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

package monitors

import (
	"fmt"
	"net"
)

// Resolver lets us define custom DNS resolvers similar to what the go stdlib provides, but
// potentially with custom functionality
type Resolver interface {
	// ResolveIPAddr is an analog of net.ResolveIPAddr
	ResolveIPAddr(network string, host string) (*net.IPAddr, error)
	// LookupIP is an analog of net.LookupIP
	LookupIP(host string) ([]net.IP, error)
}

// stdResolver uses the go std library to perform DNS resolution.
type stdResolver struct{}

// We only ever need one instance of this
var StdResolver = stdResolver{}

func (s stdResolver) ResolveIPAddr(network string, host string) (*net.IPAddr, error) {
	return net.ResolveIPAddr(network, host)
}

func (s stdResolver) LookupIP(host string) ([]net.IP, error) {
	return net.LookupIP(host)
}

type InvalidIPStringError struct {
	ipString string
}

// staticResolver allows for a custom in-memory mapping of hosts to IPs, it ignores network names
// and zones.
type staticResolver struct {
	mapping map[string][]net.IP
}

func CreateStaticResolver() staticResolver {
	return staticResolver{mapping: map[string][]net.IP{}}
}

func (i InvalidIPStringError) Error() string {
	return fmt.Sprintf("Could not parse IP from string %s", i.ipString)
}

func (s staticResolver) Add(hostname string, ip string) error {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return InvalidIPStringError{ip}
	}

	if found, ok := s.mapping[hostname]; ok {
		s.mapping[hostname] = append(found, parsed)
	} else {
		s.mapping[hostname] = []net.IP{parsed}
	}

	return nil
}

func (s staticResolver) ResolveIPAddr(network string, host string) (*net.IPAddr, error) {
	found, err := s.LookupIP(host)
	if err != nil {
		return nil, err
	}
	return &net.IPAddr{IP: found[0]}, nil
}

func (s staticResolver) LookupIP(host string) ([]net.IP, error) {
	if found, ok := s.mapping[host]; ok {
		return found, nil
	} else {
		return nil, makeStaticNXDomainErr(host)
	}
}

func makeStaticNXDomainErr(host string) *net.DNSError {
	return &net.DNSError{
		IsNotFound: true,
		Err:        fmt.Sprintf("Hostname '%s' not found in static resolver", host),
	}
}
