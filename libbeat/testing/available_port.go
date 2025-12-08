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

	"github.com/stretchr/testify/require"
)

// AvailableTCP4Port returns an unused TCP port for 127.0.0.1.
func AvailableTCP4Port() (uint16, error) {
	resolved, err := net.ResolveTCPAddr("tcp4", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}

	listener, err := net.ListenTCP("tcp4", resolved)
	if err != nil {
		return 0, err
	}
	defer listener.Close()

	tcpAddr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return 0, fmt.Errorf("expected TCP address, got %T", listener.Addr())
	}

	return uint16(tcpAddr.Port), nil //nolint:gosec // Safe conversion for port number
}

func MustAvailableTCP4Port(t *testing.T) uint16 {
	port, err := AvailableTCP4Port()
	require.NoError(t, err, "failed to get available TCP4 port")
	return port
}
