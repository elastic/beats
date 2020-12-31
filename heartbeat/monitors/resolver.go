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

// StdResolver uses the go std library to perform DNS resolution.
type StdResolver struct{}

func NewStdResolver() StdResolver {
	return StdResolver{}
}

func (s StdResolver) ResolveIPAddr(network string, host string) (*net.IPAddr, error) {
	return net.ResolveIPAddr(network, host)
}

func (s StdResolver) LookupIP(host string) ([]net.IP, error) {
	return net.LookupIP(host)
}
