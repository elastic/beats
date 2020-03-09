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

package tlscommon

import "fmt"

// TLSVersion type for TLS version.
type TLSVersion uint16

func (v TLSVersion) String() string {
	if s, ok := tlsProtocolVersionsInverse[v]; ok {
		return s
	}
	return "unknown"
}

//Unpack transforms the string into a constant.
func (v *TLSVersion) Unpack(s string) error {
	version, found := tlsProtocolVersions[s]
	if !found {
		return fmt.Errorf("invalid tls version '%v'", s)
	}

	*v = version
	return nil
}
