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

//go:build go1.13
// +build go1.13

package tlscommon

import (
	"crypto/tls"
)

// Define all the possible TLS version.
const (
	TLSVersion10 TLSVersion = tls.VersionTLS10
	TLSVersion11 TLSVersion = tls.VersionTLS11
	TLSVersion12 TLSVersion = tls.VersionTLS12
	TLSVersion13 TLSVersion = tls.VersionTLS13

	// TLSVersionMin is the min TLS version supported.
	TLSVersionMin = TLSVersion10

	// TLSVersionMax is the max TLS version supported.
	TLSVersionMax = TLSVersion13

	// TLSVersionDefaultMin is the minimal default TLS version that is
	// enabled by default. TLSVersionDefaultMin is >= TLSVersionMin
	TLSVersionDefaultMin = TLSVersion11

	// TLSVersionDefaultMax is the max default TLS version that
	// is enabled by default.
	TLSVersionDefaultMax = TLSVersionMax
)

// TLSDefaultVersions list of versions of TLS we should support.
var TLSDefaultVersions = []TLSVersion{
	TLSVersion11,
	TLSVersion12,
	TLSVersion13,
}

var tlsProtocolVersions = map[string]TLSVersion{
	"TLSv1":   TLSVersion10,
	"TLSv1.0": TLSVersion10,
	"TLSv1.1": TLSVersion11,
	"TLSv1.2": TLSVersion12,
	"TLSv1.3": TLSVersion13,
}

// Intended for ECS's tls.version_protocol_field, which does not include
// numeric version and should be lower case
type TLSVersionDetails struct {
	Version  string
	Protocol string
	Combined string
}

func (pv TLSVersionDetails) String() string {
	return pv.Combined
}

var tlsInverseLookup = map[TLSVersion]TLSVersionDetails{
	TLSVersion10: TLSVersionDetails{Version: "1.0", Protocol: "tls", Combined: "TLSv1.0"},
	TLSVersion11: TLSVersionDetails{Version: "1.1", Protocol: "tls", Combined: "TLSv1.1"},
	TLSVersion12: TLSVersionDetails{Version: "1.2", Protocol: "tls", Combined: "TLSv1.2"},
	TLSVersion13: TLSVersionDetails{Version: "1.3", Protocol: "tls", Combined: "TLSv1.3"},
}
