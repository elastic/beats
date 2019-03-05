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

package dns

// This file contains the name mapping data used to convert various DNS IDs to
// their string values.

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	mkdns "github.com/miekg/dns"
)

// dnsOpCodeToString converts a Opcode value to a string. If the type's
// string representation is unknown then the numeric value will be returned as
// a string.
func dnsOpCodeToString(opCode int) string {
	s, exists := mkdns.OpcodeToString[opCode]
	if !exists {
		return strconv.Itoa(int(opCode))
	}
	return s
}

// dnsResponseCodeToString converts a Rcode value to a string. If
// the type's string representation is unknown then "Unknown <rcode value>"
// will be returned.
func dnsResponseCodeToString(rcode int) string {
	s, exists := mkdns.RcodeToString[rcode]
	if !exists {
		return fmt.Sprintf("Unknown %d", int(rcode))
	}
	return s
}

// dnsTypeToString converts a RR type value to a string. If the type's
// string representation is unknown then the numeric value will be returned
// as a string.
func dnsTypeToString(t uint16) string {
	s, exists := mkdns.TypeToString[uint16(t)]
	if !exists {
		return strconv.Itoa(int(t))
	}
	return s
}

// dnsClassToString converts a RR class value to a string. If the class'es
// string representation is unknown then the numeric value will be returned
// as a string.
func dnsClassToString(c uint16) string {
	s, exists := mkdns.ClassToString[uint16(c)]
	if !exists {
		return strconv.Itoa(int(c))
	}
	return s
}

// dnsAlgorithmToString converts an algorithm value to a string. If the algorithm's
// string representation is unknown then the numeric value will be returned
// as a string.
func dnsAlgorithmToString(a uint8) string {
	s, exists := mkdns.AlgorithmToString[uint8(a)]
	if !exists {
		return strconv.Itoa(int(a))
	}
	return s
}

// dnsHashToString converts a hash value to a string. If the hash's
// string representation is unknown then the numeric value will be returned
// as a string.
func dnsHashToString(h uint8) string {
	s, exists := mkdns.HashToString[uint8(h)]
	if !exists {
		return strconv.Itoa(int(h))
	}
	return s
}

// dnsTypeBitsMapToString converts a map of type bits to a string. If the type's
// string representation is unknown then the numeric value will be returned
// as a string.
func dnsTypeBitsMapToString(t []uint16) string {
	var s string
	for i := 0; i < len(t); i++ {
		s += dnsTypeToString(t[i]) + " "
	}
	return strings.TrimSuffix(s, " ")
}

// saltToString converts a NSECX salt to uppercase and
// returns "-" when it is empty
// func copied from miekg/dns because unexported
func dnsSaltToString(s string) string {
	if len(s) == 0 {
		return "-"
	}
	return strings.ToUpper(s)
}

// hexStringToString converts an hexadecimal string to string. Bytes
// below 32 or above 126 are represented as an escaped base10 integer (\DDD).
// Back slashes and quotes are escaped. Tabs, carriage returns, and line feeds
// will be converted to \t, \r and \n respectively.
// Example:
func hexStringToString(hexString string) (string, error) {
	bytes, err := hex.DecodeString(hexString)
	if err != nil {
		return hexString, err
	}

	var s []byte
	for _, value := range bytes {
		switch value {
		default:
			if value < 32 || value >= 127 {
				// Unprintable characters are written as \\DDD (e.g. \\012).
				s = append(s, []byte(fmt.Sprintf("\\%03d", int(value)))...)
			} else {
				s = append(s, value)
			}
		case '"', '\\':
			s = append(s, '\\', value)
		case '\t':
			s = append(s, '\\', 't')
		case '\r':
			s = append(s, '\\', 'r')
		case '\n':
			s = append(s, '\\', 'n')
		}
	}
	return string(s), nil
}
