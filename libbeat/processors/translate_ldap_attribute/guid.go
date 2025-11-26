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

//go:build !requirefips

package translate_ldap_attribute

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrInvalidGUIDFormat is returned when the GUID format is invalid
	ErrInvalidGUIDFormat = errors.New("invalid GUID format")
)

// guidToBytes converts a GUID string in various formats to the binary format
// expected by Microsoft Active Directory.
//
// IMPORTANT: This conversion is ONLY for Microsoft Active Directory's objectGUID.
// Do NOT use for other LDAP implementations:
//   - 389 Directory Server: Uses nsUniqueId (different format)
//   - OpenLDAP and Other LDAP: Typically use RFC 4122 standard UUIDs
//
// Supported input formats:
//   - {xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx}
//   - xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
//   - xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
//
// The function handles the byte-order conversion required for Microsoft GUIDs:
// The first three components (Data1, Data2, Data3) are little-endian,
// while the remaining bytes are in network byte order.
//
// Example:
//
//	Input:  "{7fb125ee-ceaf-48ff-8385-32c516ab10ed}"
//	Output: []byte{0xee, 0x25, 0xb1, 0x7f, 0xaf, 0xce, 0xff, 0x48, 0x83, 0x85, 0x32, 0xc5, 0x16, 0xab, 0x10, 0xed}
func guidToBytes(guid string) ([]byte, error) {
	// Remove curly braces if present
	guid = strings.Trim(guid, "{}")

	// Remove hyphens
	guid = strings.ReplaceAll(guid, "-", "")

	// Validate length
	if len(guid) != 32 {
		return nil, fmt.Errorf("%w: expected 32 hex characters, got %d", ErrInvalidGUIDFormat, len(guid))
	}

	// Decode hex string
	bytes, err := hex.DecodeString(guid)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidGUIDFormat, err)
	}

	// Microsoft GUID format requires byte swapping for the first three components
	// GUID structure: {Data1-Data2-Data3-Data4[8]}
	// Data1: 4 bytes (little-endian)
	// Data2: 2 bytes (little-endian)
	// Data3: 2 bytes (little-endian)
	// Data4: 8 bytes (big-endian/network order)

	// Swap Data1 (first 4 bytes)
	bytes[0], bytes[1], bytes[2], bytes[3] = bytes[3], bytes[2], bytes[1], bytes[0]

	// Swap Data2 (next 2 bytes)
	bytes[4], bytes[5] = bytes[5], bytes[4]

	// Swap Data3 (next 2 bytes)
	bytes[6], bytes[7] = bytes[7], bytes[6]

	// Data4 remains in network byte order (no swap needed)

	return bytes, nil
}

// escapeBinaryForLDAP escapes binary data for use in LDAP filters.
// Each byte is represented as \XX where XX is the hexadecimal value.
func escapeBinaryForLDAP(data []byte) string {
	var sb strings.Builder
	for _, b := range data {
		fmt.Fprintf(&sb, "\\%02x", b)
	}
	return sb.String()
}
