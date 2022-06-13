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

//nolint:unused,structcheck,varcheck // How many ways to check for unused? (╯°□°）╯︵ ┻━┻
package route

import (
	"errors"
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	libiphlpapi          = windows.NewLazySystemDLL("Iphlpapi.dll")
	getBestInterfaceEx   = libiphlpapi.NewProc("GetBestInterfaceEx")
	getInterfaceInfo     = libiphlpapi.NewProc("GetInterfaceInfo")
	getAdaptersAddresses = libiphlpapi.NewProc("GetAdaptersAddresses")
)

// Default returns the interface and netstat device index of the network interface
// used for the first identified default route for the specified address family.
// Valid values for af are syscall.AF_INET and syscall.AF_INET6.
func Default(af int) (name string, index int, err error) {
	switch af {
	case windows.AF_INET, windows.AF_INET6:
	default:
		return "", -1, errors.New("invalid family")
	}

	// https://docs.microsoft.com/en-us/windows/win32/api/iphlpapi/nf-iphlpapi-getbestinterfaceex
	// https://docs.microsoft.com/en-us/windows/win32/api/iphlpapi/nf-iphlpapi-getinterfaceinfo
	//
	// FIXME: This may not correctly work with IPv6 (when the interface is only available for IPv4),
	// for the API to obtain this see:
	// https://docs.microsoft.com/en-us/windows/win32/api/iphlpapi/nf-iphlpapi-getadaptersaddresses
	// Types are declared below, but calls need to be put in place.

	type sockaddr struct {
		Family uint16
		_      [26]byte
	}

	var idx uint32
	family := &sockaddr{Family: uint16(af)}
	ret, _, err := getBestInterfaceEx.Call(uintptr(unsafe.Pointer(family)), uintptr(unsafe.Pointer(&idx)))
	runtime.KeepAlive(family)
	if ret != windows.NO_ERROR {
		return "", -1, err
	}
	var dwOutBufLen int32
	ret, _, err = getInterfaceInfo.Call(0, uintptr(unsafe.Pointer(&dwOutBufLen)))
	switch syscall.Errno(ret) {
	case windows.ERROR_INSUFFICIENT_BUFFER, windows.NO_ERROR:
	default:
		return "", -1, err
	}
	if dwOutBufLen == 0 {
		return "", -1, ErrNotFound
	}
	buf := make([]byte, dwOutBufLen)
	ret, _, err = getInterfaceInfo.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&dwOutBufLen)))
	runtime.KeepAlive(dwOutBufLen)
	if ret != windows.NO_ERROR {
		return "", -1, err
	}
	pIfTable := (*ipInterfaceInfo)(unsafe.Pointer(&buf[0]))
	adapters := unsafe.Slice(&pIfTable.adapter, pIfTable.numAdapters)
	for _, a := range adapters {
		if a.index == idx {
			return windows.UTF16ToString(a.name[:]), int(idx), nil
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
	alignment              uint64
	next                   *ipAdapterAddressesLH
	adapterName            *byte
	FirstUnicastAddress    *windows.IpAdapterUnicastAddress
	FirstAnycastAddress    *windows.IpAdapterAnycastAddress
	FirstMulticastAddress  *windows.IpAdapterMulticastAddress
	FirstDnsServerAddress  *windows.IpAdapterDnsServerAdapter
	DnsSuffix              *uint16
	Description            *uint16
	FriendlyName           *uint16
	PhysicalAddress        [syscall.MAX_ADAPTER_ADDRESS_LENGTH]byte
	PhysicalAddressLength  uint32
	Flags                  uint32
	Mtu                    uint32
	IfType                 uint32
	OperStatus             uint32
	Ipv6IfIndex            uint32
	ZoneIndices            [16]uint32
	FirstPrefix            *windows.IpAdapterPrefix
	TransmitLinkSpeed      uint64
	ReceiveLinkSpeed       uint64
	FirstWinsServerAddress *ipAdapterWinsServerAddressLH
	FirstGatewayAddress    *ipAdapterGatewayAddressLH
	Ipv4Metric             uint32
	Ipv6Metric             uint32
	Luid                   uint64
	Dhcpv4Server           socketAddress
	CompartmentId          uint32
	NetworkGuid            guid
	ConnectionType         uint32
	TunnelType             uint32
	Dhcpv6Server           socketAddress
	Dhcpv6ClientDuid       [maxDHCPv6DUIDLength]byte
	Dhcpv6ClientDuidLength uint32
	Dhcpv6Iaid             uint32
	FirstDnsSuffix         *ipAdapterDNSSuffix
}

// https://doxygen.reactos.org/d2/d14/iptypes_8h_source.html#l00176
type ipAdapterWinsServerAddressLH struct {
	Alignment uint64
	next      *ipAdapterWinsServerAddressLH
	Address   socketAddress
}

// https://doxygen.reactos.org/d2/d14/iptypes_8h_source.html#l00190
type ipAdapterGatewayAddressLH struct {
	Alignment uint64
	next      *ipAdapterGatewayAddressLH
	Address   socketAddress
}

// https://doxygen.reactos.org/d8/d15/scsiwmi_8h_source.html#l00050
type guid struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

// https://doxygen.reactos.org/d1/db0/ws2def_8h_source.html#l00374
type socketAddress struct {
	lpSockaddr      int32
	iSockaddrLength int32
}

// https://doxygen.reactos.org/d2/d14/iptypes_8h_source.html#l00204
type ipAdapterDNSSuffix struct {
	next   *ipAdapterDNSSuffix
	String [maxDNSSuffixStringLength]uint16
}

const (
	maxDHCPv6DUIDLength      = 130 // https://doxygen.reactos.org/d2/d14/iptypes_8h_source.html#l00033
	maxDNSSuffixStringLength = 256 // https://doxygen.reactos.org/d2/d14/iptypes_8h_source.html#l00034
)
