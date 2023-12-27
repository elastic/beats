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

//go:build darwin || dragonfly || freebsd || netbsd || openbsd

package route

import (
	"errors"

	"golang.org/x/net/route"
	"golang.org/x/sys/unix"
)

// Default returns the interface and netstat device index of the network interface
// used for the first identified default route for the specified address family.
// Valid values for af are syscall.AF_INET and syscall.AF_INET6.
func Default(af int) (name string, index int, err error) {
	const gateway = unix.RTF_UP | unix.RTF_GATEWAY

	switch af {
	case unix.AF_INET, unix.AF_INET6:
	default:
		return "", -1, errors.New("invalid family")
	}

	rib, err := route.FetchRIB(af, route.RIBTypeRoute, 0)
	if err != nil {
		return "", -1, err
	}
	msgs, err := route.ParseRIB(route.RIBTypeRoute, rib)
	if err != nil {
		return "", -1, err
	}
	ok := false
	for _, m := range msgs {
		m := m.(*route.RouteMessage)
		if m.Flags&gateway == gateway {
			index = m.Index
			ok = true
			break
		}
	}
	if !ok {
		return "", -1, ErrNotFound
	}

	rib, err = route.FetchRIB(af, route.RIBTypeInterface, 0)
	if err != nil {
		return "", -1, err
	}
	msgs, err = route.ParseRIB(route.RIBTypeInterface, rib)
	if err != nil {
		return "", -1, err
	}

	if index < len(msgs) {
		// Trust but verify.
		m, ok := msgs[index].(*route.InterfaceMessage)
		if ok {
			if m.Index == index {
				return m.Name, index, nil
			}
		}
	}
	for _, m := range msgs {
		switch m := m.(type) {
		case *route.InterfaceMessage:
			if m.Index == index {
				return m.Name, index, nil
			}
		}
	}
	return "", -1, ErrNotFound
}
