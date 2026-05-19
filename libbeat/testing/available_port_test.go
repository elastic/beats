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

package testing

import (
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAvailableTCP4Ports(t *testing.T) {
	ports, err := AvailableTCP4Ports(3)
	require.NoError(t, err, "failed to allocate multiple available TCP4 ports")
	require.Len(t, ports, 3, "expected three allocated TCP4 ports")

	seen := map[uint16]struct{}{}
	for _, port := range ports {
		_, exists := seen[port]
		assert.False(t, exists, "allocated duplicate TCP4 port: %d", port)
		seen[port] = struct{}{}
	}

	for _, port := range ports {
		listener, err := net.Listen("tcp4", fmt.Sprintf("127.0.0.1:%d", port))
		require.NoError(t, err, "expected to bind allocated TCP4 port %d after helper returns", port)
		require.NoError(t, listener.Close(), "failed closing listener for TCP4 port %d", port)
	}
}

func TestAvailableTCP4PortsInvalidCount(t *testing.T) {
	ports, err := AvailableTCP4Ports(0)
	require.Error(t, err, "expected invalid port count to fail")
	assert.Nil(t, ports, "expected no ports for invalid count")
}
