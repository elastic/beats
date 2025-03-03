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

package fingerprint

import (
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"strings"
)

type namedEncodingMethod struct {
	Name   string
	Encode encodingMethod
}
type encodingMethod func([]byte) string

var encodings = map[string]namedEncodingMethod{}

func init() {
	for _, e := range []namedEncodingMethod{
		{Name: "hex", Encode: hex.EncodeToString},
		{Name: "base32", Encode: base32.StdEncoding.EncodeToString},
		{Name: "base64", Encode: base64.StdEncoding.EncodeToString},
	} {
		encodings[e.Name] = e
	}
}

// Unpack creates the encodingMethod from the given string
func (e *namedEncodingMethod) Unpack(str string) error {
	str = strings.ToLower(str)

	m, found := encodings[str]
	if !found {
		return makeErrUnknownEncoding(str)
	}

	*e = m
	return nil
}
