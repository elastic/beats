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

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
)

func TestMarshallText(t *testing.T) {
	var verification TLSVerificationMode
	bytes, err := verification.MarshalText()
	require.NoError(t, err)
	require.NotNil(t, bytes)
	require.Equal(t, "full", string(bytes))

	verification = VerifyNone
	bytes, err = verification.MarshalText()
	require.NoError(t, err)
	require.NotNil(t, bytes)
	require.Equal(t, "none", string(bytes))
}

func TestLoadWithEmptyStringVerificationMode(t *testing.T) {
	cfg, err := load(`
    enabled: true
    certificate: mycert.pem
    key: mycert.key
    verification_mode: ""
    supported_protocols: [TLSv1.1, TLSv1.2]
    renegotiation: freely
  `)

	assert.NoError(t, err)
	assert.Equal(t, cfg.VerificationMode, VerifyFull)
}

func TestLoadWithEmptyVerificationMode(t *testing.T) {
	cfg, err := load(`
    enabled: true
    verification_mode:
    supported_protocols: [TLSv1.1, TLSv1.2]
    curve_types:
      - P-521
    renegotiation: freely
  `)

	assert.NoError(t, err)
	assert.Equal(t, cfg.VerificationMode, VerifyFull)
}
