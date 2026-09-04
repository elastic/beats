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

package http

import (
	"net/http"

	ntlmssp "github.com/Azure/go-ntlmssp"
)

// wrapNTLMRoundTripper wraps the transport with the NTLM negotiator, which
// converts the request's Basic auth credentials into an NTLM/Negotiate
// handshake. NTLM relies on MD4/RC4, so it is unavailable in FIPS builds (see
// ntlm_fips.go).
func wrapNTLMRoundTripper(rt http.RoundTripper) (http.RoundTripper, error) {
	return ntlmssp.Negotiator{RoundTripper: rt}, nil
}
