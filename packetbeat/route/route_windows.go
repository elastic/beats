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

//nolint:unused,structcheck // How many ways to check for unused? (╯°□°）╯︵ ┻━┻ Fields kept for documentation.
package route

import (
	"errors"
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	// For details of the APIs used, see:
	// https://docs.microsoft.com/en-us/windows/win32/api/iphlpapi/nf-iphlpapi-getbestinterfaceex
	// https://docs.microsoft.com/en-us/windows/win32/api/iphlpapi/nf-iphlpapi-getadaptersaddresses
	libiphlpapi          = windows.NewLazySystemDLL("Iphlpapi.dll")
	getBestInterfaceEx   = libiphlpapi.NewProc("GetBestInterfaceEx")
	getAdaptersAddresses = libiphlpapi.NewProc("GetAdaptersAddresses")
)

// Default returns the interface and netstat device index of the network interface
// used for the first identified default route for the specified address family.
// Valid values for af are syscall.AF_INET and syscall.AF_INET6. The iface name
// returned will include only the GUID of the device.
func Default(af int) (name string, index int, err error) {
	switch af {
	case windows.AF_INET, windows.AF_INET6:
	default:
		return "", -1, errors.New("invalid family")
	}

	type sockaddr struct {
		Family uint16
		_      [26]byte
	}

	var idx uint32
	family := &sockaddr{Family: uint16(af)}
	ret, _, err := getBestInterfaceEx.Call(uintptr(unsafe.Pointer(family)), uintptr(unsafe.Pointer(&idx)))
	runtime.KeepAlive(family)
	if ret != windows.NO_ERROR {
		if syscall.Errno(ret) == windows.ERROR_NOT_FOUND {
			err = ErrNotFound
		}
		return "", -1, err
	}

	const (
		workingBufferSize = 15000
		maxTries          = 3
	)
	var buf []byte
	outBufLen := workingBufferSize
loop:
	for i := 0; i < maxTries; i++ {
		buf = make([]byte, outBufLen)
		ret, _, err = getAdaptersAddresses.Call(uintptr(af), 0, 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&outBufLen)))
		runtime.KeepAlive(outBufLen)
		switch syscall.Errno(ret) {
		case windows.ERROR_BUFFER_OVERFLOW:
			continue
		case windows.NO_ERROR:
			break loop
		case windows.ERROR_NO_DATA:
			return "", -1, ErrNotFound
		default:
			return "", -1, err
		}
	}

	addresses := (*ipAdapterAddressesLH)(unsafe.Pointer(&buf[0]))
	for ; addresses != nil; addresses = addresses.next {
		switch af {
		case windows.AF_INET:
			if addresses.ifIndex != 0 && addresses.ifIndex == idx {
				return windows.BytePtrToString(addresses.adapterName), int(idx), nil
			}
		case windows.AF_INET6:
			if addresses.ipv6IfIndex != 0 && addresses.ipv6IfIndex == idx {
				return windows.BytePtrToString(addresses.adapterName), int(idx), nil
			}
		}
	}
	return "", -1, ErrNotFound
}

// https://docs.microsoft.com/en-us/windows/win32/api/ipexport/ns-ipexport-ip_interface_info
type ipInterfaceInfo struct {
	numAdapters int32
	adapter     ipAdapterIndexMap
}

// https://docs.microsoft.com/en-us/windows/win32/api/ipexport/ns-ipexport-ip_adapter_index_map
type ipAdapterIndexMap struct {
	index uint32
	name  [maxAdapterName]uint16
}

// https://doxygen.reactos.org/d3/d8d/ipexport_8h_source.html#l00143
const maxAdapterName = 128

// https://docs.microsoft.com/en-us/windows/win32/api/iptypes/ns-iptypes-ip_adapter_addresses_lh
type ipAdapterAddressesLH struct {
	length                 uint32
	ifIndex                uint32
	next                   *ipAdapterAddressesLH
	adapterName            *byte
	firstUnicastAddress    *windows.IpAdapterUnicastAddress
	firstAnycastAddress    *windows.IpAdapterAnycastAddress
	firstMulticastAddress  *windows.IpAdapterMulticastAddress
	firstDnsServerAddress  *windows.IpAdapterDnsServerAdapter
	dnsSuffix              *uint16
	description            *uint16
	friendlyName           *uint16
	physicalAddress        [syscall.MAX_ADAPTER_ADDRESS_LENGTH]byte
	physicalAddressLength  uint32
	flags                  uint32
	mtu                    uint32
	ifType                 uint32
	operStatus             uint32
	ipv6IfIndex            uint32
	zoneIndices            [16]uint32
	firstPrefix            *windows.IpAdapterPrefix
	transmitLinkSpeed      uint64
	receiveLinkSpeed       uint64
	firstWinsServerAddress *ipAdapterWinsServerAddressLH
	firstGatewayAddress    *ipAdapterGatewayAddressLH
	ipv4Metric             uint32
	ipv6Metric             uint32
	luid                   uint64
	dhcpv4Server           socketAddress
	compartmentId          uint32
	networkGuid            guid
	connectionType         uint32
	tunnelType             uint32
	dhcpv6Server           socketAddress
	dhcpv6ClientDuid       [maxDHCPv6DUIDLength]byte
	dhcpv6ClientDuidLength uint32
	dhcpv6Iaid             uint32
	firstDnsSuffix         *ipAdapterDNSSuffix
}

// https://doxygen.reactos.org/d2/d14/iptypes_8h_source.html#l00176
type ipAdapterWinsServerAddressLH struct {
	alignment uint64
	next      *ipAdapterWinsServerAddressLH
	address   socketAddress
}

// https://doxygen.reactos.org/d2/d14/iptypes_8h_source.html#l00190
type ipAdapterGatewayAddressLH struct {
	alignment uint64
	next      *ipAdapterGatewayAddressLH
	address   socketAddress
}

// https://doxygen.reactos.org/d8/d15/scsiwmi_8h_source.html#l00050
type guid struct {
	data1 uint32
	data2 uint16
	data3 uint16
	data4 [8]byte
}

// https://doxygen.reactos.org/d1/db0/ws2def_8h_source.html#l00374
type socketAddress struct {
	lpSockaddr      int32
	iSockaddrLength int32
}

// https://doxygen.reactos.org/d2/d14/iptypes_8h_source.html#l00204
type ipAdapterDNSSuffix struct {
	next   *ipAdapterDNSSuffix
	string [maxDNSSuffixStringLength]uint16
}

const (
	maxDHCPv6DUIDLength      = 130 // https://doxygen.reactos.org/d2/d14/iptypes_8h_source.html#l00033
	maxDNSSuffixStringLength = 256 // https://doxygen.reactos.org/d2/d14/iptypes_8h_source.html#l00034
)
