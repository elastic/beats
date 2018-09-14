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

package dhcpv4

import (
	"fmt"
	"net"

	"github.com/insomniacslk/dhcp/dhcpv4"
)

// IPAddressesOption represents an option whose value is a list of IP addresses.
type IPAddressesOption struct {
	dhcpv4.OptionCode
	IPAddresses []net.IP
}

// ParseIPAddressesOption returns a new IPAddressesOption from a byte
// stream, or error if any.
func ParseIPAddressesOption(data []byte) (*IPAddressesOption, error) {
	if len(data) < 2 {
		return nil, dhcpv4.ErrShortByteStream
	}
	code := dhcpv4.OptionCode(data[0])
	length := int(data[1])
	if length == 0 || length%4 != 0 {
		return nil, fmt.Errorf("Invalid length: expected multiple of 4 larger than 4, got %v", length)
	}
	if len(data) < 2+length {
		return nil, dhcpv4.ErrShortByteStream
	}
	servers := make([]net.IP, 0, length/4)
	for idx := 0; idx < length; idx += 4 {
		b := data[2+idx : 2+idx+4]
		servers = append(servers, net.IPv4(b[0], b[1], b[2], b[3]))
	}
	return &IPAddressesOption{OptionCode: code, IPAddresses: servers}, nil
}

// Code returns the option code.
func (o *IPAddressesOption) Code() dhcpv4.OptionCode {
	return o.OptionCode
}

// ToBytes returns a serialized stream of bytes for this option.
func (o *IPAddressesOption) ToBytes() []byte {
	ret := []byte{byte(o.Code()), byte(o.Length())}
	for _, ns := range o.IPAddresses {
		ret = append(ret, ns...)
	}
	return ret
}

// String returns a human-readable string.
func (o *IPAddressesOption) String() string {
	var servers string
	for idx, ns := range o.IPAddresses {
		servers += ns.String()
		if idx < len(o.IPAddresses)-1 {
			servers += ", "
		}
	}
	return fmt.Sprintf("IP Addresses -> %v", servers)
}

// Length returns the length of the data portion (excluding option code an byte
// length).
func (o *IPAddressesOption) Length() int {
	return len(o.IPAddresses) * 4
}
