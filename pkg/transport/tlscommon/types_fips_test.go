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

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadUnsupporteTLSVersion(t *testing.T) {
	cfg, err := load(`
    enabled: true
    certificate: mycert.pem
    key: mycert.key
    verification_mode: ""
    supported_protocols: [TLSv1.1, TLSv1.2]
    renegotiation: freely
  `)

	assert.ErrorContains(t, err, "unsupported tls version")
	assert.Nil(t, cfg)
}

func TestLoadUnsupportedCiphers(t *testing.T) {
	cfg, err := load(`
    enabled: true
    certificate: mycert.pem
    key: mycert.key
    verification_mode: ""
    supported_protocols: [TLSv1.2, TLSv1.3]
    cipher_suites: ["RSA-AES-256-CBC-SHA"]
    renegotiation: freely
  `)

	assert.ErrorContains(t, err, "unsupported tls cipher suite: TLS_RSA_WITH_AES_256_CBC_SHA")
	assert.Nil(t, cfg)
}

func TestLoadUnsupportedCurveTypes(t *testing.T) {
	cfg, err := load(`
    enabled: true
    certificate: mycert.pem
    key: mycert.key
    verification_mode: ""
    supported_protocols: [TLSv1.2, TLSv1.3]
    curve_types:
      - X25519
    renegotiation: freely
  `)

	assert.ErrorContains(t, err, "unsupported curve type: X25519")
	assert.Nil(t, cfg)
}
