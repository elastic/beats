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

package netmetrics

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

func TestAddrs(t *testing.T) {
	t.Run("ipv4", func(t *testing.T) {
		addr4, addr6, err := addrs("127.0.0.1:65535", "", logptest.NewTestingLogger(t, ""))
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(addr4) == 0 {
			t.Errorf("expected addr in addr4 for IPv4 address: addr6 is %v", addr6)
		}
		if len(addr6) != 0 {
			t.Errorf("unexpected addrs in addr6 for IPv4 address: %v", addr6)
		}
	})

	// A listener on 0.0.0.0 is a dual-stack socket that is only listed in
	// /proc/net/{tcp,udp}6 when IPv6 is enabled, so both tables are needed.
	t.Run("ipv4_unspecified", func(t *testing.T) {
		for _, network := range []string{"", "tcp", "udp"} {
			addr4, addr6, err := addrs("0.0.0.0:65535", network, logptest.NewTestingLogger(t, ""))
			assert.NoError(t, err)
			assert.Equal(t, []string{"00000000:FFFF"}, addr4, "network %q", network)
			assert.Equal(t, []string{"00000000000000000000000000000000:FFFF"}, addr6, "network %q", network)
		}
	})

	// With an IPv4-only network the socket is AF_INET, so matching IPv6
	// sockets by port would count unrelated listeners.
	t.Run("ipv4_unspecified_ipv4_only_network", func(t *testing.T) {
		for _, network := range []string{"tcp4", "udp4"} {
			addr4, addr6, err := addrs("0.0.0.0:65535", network, logptest.NewTestingLogger(t, ""))
			assert.NoError(t, err)
			assert.Equal(t, []string{"00000000:FFFF"}, addr4, "network %q", network)
			assert.Empty(t, addr6, "network %q", network)
		}
	})

	t.Run("ipv6", func(t *testing.T) {
		addr4, addr6, err := addrs("[::]:9001", "", logptest.NewTestingLogger(t, ""))
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(addr4) != 0 {
			t.Errorf("unexpected addr in addr4 for IPv6 address: %v", addr4)
		}
		if len(addr6) == 0 {
			t.Errorf("expected addrs in addr6 for IPv6 address: addr4 is %v", addr4)
		}
	})
}
