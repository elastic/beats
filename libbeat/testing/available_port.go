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

import "net"

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

	tcpAddr := uint16(listener.Addr().(*net.TCPAddr).Port)

	return tcpAddr, nil
}
