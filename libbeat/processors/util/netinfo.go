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

package util

import (
	"net"
	"sort"

	"github.com/joeshaw/multierror"
)

// GetNetInfo returns lists of IPs and MACs for the machine it is executed on.
func GetNetInfo() (ipList []string, hwList []string, err error) {
	// Get all interfaces and loop through them
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, nil, err
	}

	// Keep track of all errors
	var errs multierror.Errors

	for _, i := range ifaces {
		// Skip loopback interfaces
		if i.Flags&net.FlagLoopback == net.FlagLoopback {
			continue
		}

		hw := i.HardwareAddr.String()
		// Skip empty hardware addresses
		if hw != "" {
			hwList = append(hwList, hw)
		}

		addrs, err := i.Addrs()
		if err != nil {
			// If we get an error, keep track of it and continue with the next interface
			errs = append(errs, err)
			continue
		}

		for _, addr := range addrs {
			switch v := addr.(type) {
			case *net.IPNet:
				ipList = append(ipList, v.IP.String())
			case *net.IPAddr:
				ipList = append(ipList, v.IP.String())
			}
		}
	}

	return ipList, unique(hwList), errs.Err()
}

// unique returns addrs lexically sorted and with repeated elements
// omitted.
func unique(addrs []string) []string {
	if len(addrs) < 2 {
		return addrs
	}
	sort.Strings(addrs)
	curr := 0
	for i, addr := range addrs {
		if addr == addrs[curr] {
			continue
		}
		curr++
		if curr < i {
			addrs[curr], addrs[i] = addrs[i], ""
		}
	}
	return addrs[:curr+1]
}
