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

package generator

import (
	"crypto/rand"
	"net"
)

// Golang port of https://github.com/menderesk/elasticsearch/blob/a666fb2266/server/src/main/java/org/elasticsearch/common/MacAddressProvider.java

type id []byte

const addrLen = 6

func getSecureMungedMACAddress() ([]byte, error) {
	addr, err := getMacAddress()
	if err != nil {
		return nil, err
	}

	if !isValidAddress(addr) {
		addr, err = constructDummyMulticastAddress()
		if err != nil {
			return nil, err
		}
	}

	munged := make([]byte, addrLen)
	_, err = rand.Read(munged)
	if err != nil {
		return nil, err
	}

	for i := 0; i < addrLen; i++ {
		munged[i] ^= addr[i]
	}

	return munged, nil
}

func getMacAddress() ([]byte, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err

	}
	for _, i := range interfaces {
		if i.Flags != net.FlagLoopback {
			// Pick the first valid non loopback address we find
			addr := i.HardwareAddr
			if isValidAddress(addr) {
				return addr, nil
			}
		}
	}

	// Could not find a mac address
	return nil, nil
}

func isValidAddress(addr []byte) bool {
	if addr == nil || len(addr) != 6 {
		return false
	}

	for _, b := range addr {
		if b != 0x00 {
			return true // If any of the bytes are non zero assume a good address
		}
	}

	return false
}

func constructDummyMulticastAddress() ([]byte, error) {
	dummy := make([]byte, addrLen)
	_, err := rand.Read(dummy)
	if err != nil {
		return nil, err
	}

	// Set the broadcast bit to indicate this is not a _real_ mac address
	dummy[0] |= byte(0x01)
	return dummy, nil
}
