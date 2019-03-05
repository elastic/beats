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

package flowhash

import (
	"encoding/base64"
	"encoding/hex"
)

// Encoding is used to encode the flow hash.
type Encoding interface {
	EncodeToString([]byte) string
}

var (
	// HexEncoding encodes the checksum in hexadecimal.
	HexEncoding = hexEncoding{}

	// Base64Encoding uses Base64 to encode the checksum, including
	// padding characters. This is the default for a Community ID.
	// This is an alias for the StdEncoding in the encoding/base64 package.
	Base64Encoding = base64.StdEncoding
)

type hexEncoding struct{}

func (hexEncoding) EncodeToString(data []byte) string {
	return hex.EncodeToString(data)
}
