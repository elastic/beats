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

package route

import (
	"errors"
	"net"
	"strings"

	"github.com/google/gopacket/routing"
	"golang.org/x/sys/unix"
)

// Default returns the interface and netstat device index of the network interface
// used for the first identified default route for the specified address family.
// Valid values for af are syscall.AF_INET and syscall.AF_INET6.
func Default(af int) (name string, index int, err error) {
	var addr net.IP
	switch af {
	case unix.AF_INET:
		addr = make(net.IP, 4)
	case unix.AF_INET6:
		addr = make(net.IP, 16)
	default:
		return "", -1, errors.New("invalid family")
	}

	r, err := routing.New()
	if err != nil {
		return "", -1, err
	}
	iface, _, _, err := r.Route(addr)
	if err != nil {
		// This is nasty, but the only way we can get this information.
		// https://github.com/elastic/gopacket/blob/d412fca7f83ac6653ceec11f5276dae3e392a527/routing/routing.go#L153
		//
		// Note also that we should never receive any other error here
		// since the address we are passing in is guaranteed to be a
		// valid IP address.
		if strings.HasPrefix(err.Error(), "no route found") {
			err = ErrNotFound
		}
		return "", -1, err
	}
	return iface.Name, iface.Index, nil
}
