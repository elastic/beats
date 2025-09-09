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

package common

import (
	"net"
)

// LocalIPAddrs finds the IP addresses of the hosts on which
// the shipper currently runs on.
func LocalIPAddrs() ([]net.IP, error) {
	var localIPAddrs []net.IP
	ipaddrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}
	for _, addr := range ipaddrs {
		var ip net.IP
		ok := true

		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		default:
			ok = false
		}

		if !ok {
			continue
		}

		localIPAddrs = append(localIPAddrs, ip)
	}
	return localIPAddrs, nil
}
