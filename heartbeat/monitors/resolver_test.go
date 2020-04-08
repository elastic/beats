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
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEmptyStaticResolver(t *testing.T) {
	r := CreateStaticResolver()

	host := "foo"

	ip, err := r.ResolveIPAddr("ip", host)
	require.Nil(t, ip)
	require.Equal(t, makeStaticNXDomainErr(host), err)

	ips, err := r.LookupIP(host)
	require.Nil(t, ips)
	require.Equal(t, makeStaticNXDomainErr(host), err)
}

func TestStaticResolver(t *testing.T) {
	r := CreateStaticResolver()

	host := "foo"
	expectedIp := net.ParseIP("123.123.123.123")
	r.Add(host, expectedIp.String())

	ipAddr, err := r.ResolveIPAddr("ip", host)
	require.Equal(t, &net.IPAddr{IP: expectedIp}, ipAddr)
	require.Nil(t, err)

	ips, err := r.LookupIP(host)
	require.Equal(t, []net.IP{expectedIp}, ips)
	require.Nil(t, err)

	// Test that adding 'foo' doesn't cause other lookups to succeed
	missingHost := "missing"
	_, err = r.ResolveIPAddr("ip", missingHost)
	require.Equal(t, makeStaticNXDomainErr(missingHost), err)
	_, err = r.LookupIP(missingHost)
	require.Equal(t, makeStaticNXDomainErr(missingHost), err)
}

func TestStaticResolverMulti(t *testing.T) {
	r := CreateStaticResolver()

	host := "foo"
	expectedIps := []net.IP{
		net.ParseIP("123.123.123.123"),
		net.ParseIP("1.2.3.4"),
		net.ParseIP("5.6.7.8"),
	}

	for _, ip := range expectedIps {
		r.Add(host, ip.String())
	}

	ipAddr, err := r.ResolveIPAddr("ip", host)
	require.Equal(t, &net.IPAddr{IP: expectedIps[0]}, ipAddr)
	require.Nil(t, err)

	ips, err := r.LookupIP(host)
	require.Equal(t, expectedIps, ips)
	require.Nil(t, err)
}
