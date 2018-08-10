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

// IPAddressOption represents an option whose value is a single IP address.
type IPAddressOption struct {
	OptionCode dhcpv4.OptionCode
	IPAddress  net.IP
}

// ParseIPAddressOption returns a new IPAddressOption from a byte
// stream, or error if any.
func ParseIPAddressOption(data []byte) (*IPAddressOption, error) {
	if len(data) < 2 {
		return nil, dhcpv4.ErrShortByteStream
	}
	code := dhcpv4.OptionCode(data[0])
	length := int(data[1])
	if length != 4 {
		return nil, fmt.Errorf("unexepcted length: expected 4, got %v", length)
	}
	if len(data) < 6 {
		return nil, dhcpv4.ErrShortByteStream
	}
	return &IPAddressOption{OptionCode: code, IPAddress: net.IP(data[2 : 2+length])}, nil
}

// Code returns the option code.
func (o *IPAddressOption) Code() dhcpv4.OptionCode {
	return o.OptionCode
}

// ToBytes returns a serialized stream of bytes for this option.
func (o *IPAddressOption) ToBytes() []byte {
	ret := []byte{byte(o.Code()), byte(o.Length())}
	return append(ret, o.IPAddress.To4()...)
}

// String returns a human-readable string.
func (o *IPAddressOption) String() string {
	return fmt.Sprintf("IP Address -> %v", o.IPAddress.String())
}

// Length returns the length of the data portion (excluding option code an byte
// length).
func (o *IPAddressOption) Length() int {
	return len(o.IPAddress.To4())
}
