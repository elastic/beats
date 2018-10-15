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

	"github.com/insomniacslk/dhcp/dhcpv4"
)

// TextOption represents an option whose value is a string.
type TextOption struct {
	OptionCode dhcpv4.OptionCode
	Text       string
}

// ParseTextOption returns a new TextOption from a byte
// stream, or error if any.
func ParseTextOption(data []byte) (*TextOption, error) {
	if len(data) < 2 {
		return nil, dhcpv4.ErrShortByteStream
	}
	code := dhcpv4.OptionCode(data[0])
	length := int(data[1])
	if len(data) < 2+length {
		return nil, dhcpv4.ErrShortByteStream
	}
	return &TextOption{OptionCode: code, Text: string(data[2 : 2+length])}, nil
}

// Code returns the option code.
func (o *TextOption) Code() dhcpv4.OptionCode {
	return o.OptionCode
}

// ToBytes returns a serialized stream of bytes for this option.
func (o *TextOption) ToBytes() []byte {
	return append([]byte{byte(o.Code()), byte(o.Length())}, []byte(o.Text)...)
}

// String returns a human-readable string.
func (o *TextOption) String() string {
	return fmt.Sprintf("Text -> %v", o.Text)
}

// Length returns the length of the data portion (excluding option code an byte
// length).
func (o *TextOption) Length() int {
	return len(o.Text)
}
