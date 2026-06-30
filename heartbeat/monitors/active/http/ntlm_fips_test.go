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

package http

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conf "github.com/elastic/elastic-agent-libs/config"
)

// TestNTLMRejectedInFIPS verifies NTLM monitors fail to build in FIPS mode,
// since NTLM relies on non-FIPS-approved primitives (MD4/RC4).
func TestNTLMRejectedInFIPS(t *testing.T) {
	cfg, err := conf.NewConfigFrom(map[string]interface{}{
		"hosts": "http://localhost:9200",
		"ntlm": map[string]interface{}{
			"enabled":  true,
			"username": "user",
			"password": "pass",
		},
	})
	require.NoError(t, err)

	_, err = create("ntlm", cfg)
	require.Error(t, err, "ntlm monitor must not be creatable in fips mode")
	assert.Contains(t, err.Error(), "fips", "error should explain that ntlm is unavailable in fips mode")
}
