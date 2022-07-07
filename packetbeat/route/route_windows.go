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
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Default returns the interface and netstat device index of the network interface
// used for the first identified default route for the specified address family.
// Valid values for af are syscall.AF_INET and syscall.AF_INET6. The iface name
// returned will include only the GUID of the device.
func Default(af int) (name string, index int, err error) {
	var family windows.Sockaddr
	switch af {
	case windows.AF_INET:
		family = &windows.SockaddrInet4{}
	case windows.AF_INET6:
		family = &windows.SockaddrInet6{}
	default:
		return "", -1, errors.New("invalid family")
	}

	var idx uint32
	err = windows.GetBestInterfaceEx(family, &idx)
	runtime.KeepAlive(family)
	switch err { //nolint:errorlint // These are errno errors.
	case nil, windows.ERROR_SUCCESS:
	case windows.ERROR_NOT_FOUND:
		return "", -1, ErrNotFound
	default:
		return "", -1, err
	}

	var addresses *windows.IpAdapterAddresses
	const (
		workingBufferSize = 15000
		maxTries          = 3
	)
	outBufLen := uint32(workingBufferSize)
loop:
	for i := 0; i < maxTries; i++ {
		buf := make([]byte, outBufLen)
		addresses = (*windows.IpAdapterAddresses)(unsafe.Pointer(&buf[0]))
		err = windows.GetAdaptersAddresses(uint32(af), 0, 0, addresses, &outBufLen)
		runtime.KeepAlive(outBufLen)
		switch err { //nolint:errorlint // These are errno errors.
		case nil, windows.ERROR_SUCCESS:
			break loop
		case windows.ERROR_BUFFER_OVERFLOW:
			continue
		case windows.ERROR_NO_DATA:
			return "", -1, ErrNotFound
		default:
			return "", -1, err
		}
	}

	for ; addresses != nil; addresses = addresses.Next {
		switch af {
		case windows.AF_INET:
			if addresses.IfIndex != 0 && addresses.IfIndex == idx {
				return windows.BytePtrToString(addresses.AdapterName), int(idx), nil
			}
		case windows.AF_INET6:
			if addresses.Ipv6IfIndex != 0 && addresses.Ipv6IfIndex == idx {
				return windows.BytePtrToString(addresses.AdapterName), int(idx), nil
			}
		}
	}
	return "", -1, ErrNotFound
}
