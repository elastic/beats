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

//go:build requirefips

package tlscommon

import "crypto/tls"

func init() {
	// try to stick to NIST SP 800-52 Rev.2
	// avoid CBC mode
	// avoid go insecure cipher suites
	// pick ciphers with NIST-approved algorithms
	for cipherName, i := range tlsCipherSuites {
		switch uint16(i) {
		case tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384:
			supportedCipherSuites[i] = cipherName
		}
	}
	// Elliptic curves approved for use in ECDSA are specified in SP 800-186,
	// as implemented in FIPS 186-5.
	// Based on NIST SP 800-186 section 3 and SP 800-56A Rev.3
	// only allows P-256, P-384, P-521
	for name, curveType := range tlsCurveTypes {
		switch tls.CurveID(curveType) {
		case tls.CurveP256, tls.CurveP384, tls.CurveP521:
			supportedCurveTypes[curveType] = name
		}
	}
}
