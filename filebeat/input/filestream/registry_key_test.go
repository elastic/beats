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

package filestream

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRegistryKey(t *testing.T) {
	testCases := map[string]struct {
		key      string
		want     registryKey
		wantOK   bool
		identity string
	}{
		"fingerprint identity": {
			key:      "filestream::my-input::fingerprint::aabb",
			want:     registryKey{"filestream", "my-input", "fingerprint", "aabb"},
			wantOK:   true,
			identity: "fingerprint::aabb",
		},
		"native identity": {
			key:      "filestream::my-input::native::13643776-64768",
			want:     registryKey{"filestream", "my-input", "native", "13643776-64768"},
			wantOK:   true,
			identity: "native::13643776-64768",
		},
		"path identity": {
			key:      "filestream::my-input::path::/var/log/app.log",
			want:     registryKey{"filestream", "my-input", "path", "/var/log/app.log"},
			wantOK:   true,
			identity: "path::/var/log/app.log",
		},
		"empty fingerprint value (final SHA-256 placeholder)": {
			key:      "filestream::my-input::fingerprint::",
			want:     registryKey{"filestream", "my-input", "fingerprint", ""},
			wantOK:   true,
			identity: "fingerprint::",
		},
		"single colon in input ID is not a separator": {
			key:      "filestream::my:input::fingerprint::aabb",
			want:     registryKey{"filestream", "my:input", "fingerprint", "aabb"},
			wantOK:   true,
			identity: "fingerprint::aabb",
		},
		"input ID containing fingerprint substring": {
			key:      "filestream::my-fingerprint-input::fingerprint::aabb",
			want:     registryKey{"filestream", "my-fingerprint-input", "fingerprint", "aabb"},
			wantOK:   true,
			identity: "fingerprint::aabb",
		},
		"too few separators is rejected": {
			key:    "filestream::malformed",
			wantOK: false,
		},
		"too many separators is rejected": {
			key:    "filestream::my-input::fingerprint::aabb::extra",
			wantOK: false,
		},
		"empty key is rejected": {
			key:    "",
			wantOK: false,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			rk, ok := parseRegistryKey(tc.key)
			assert.Equal(t, tc.wantOK, ok, "ok mismatch for key %q", tc.key)
			if !tc.wantOK {
				assert.Equal(t, registryKey{}, rk, "expected zero value on parse failure")
				return
			}

			assert.Equal(t, tc.want, rk, "parsed components mismatch")
			assert.Equal(t, tc.identity, rk.identity(), "identity() mismatch")
			assert.Equal(t, tc.key, rk.keyForIdentity(rk.identity()),
				"keyForIdentity(identity()) must round-trip the original key")
		})
	}
}

func TestRegistryKeyIsFingerprint(t *testing.T) {
	fp, ok := parseRegistryKey("filestream::my-input::fingerprint::aabb")
	require.True(t, ok)
	assert.True(t, fp.isFingerprint(), "fingerprint identity should report true")

	native, ok := parseRegistryKey("filestream::my-input::native::13643776-64768")
	require.True(t, ok)
	assert.False(t, native.isFingerprint(), "native identity should report false")
}

// TestRegistryKeyForIdentity checks that the plugin and input prefix is
// preserved while the identity is swapped — the operation used to move a
// registry entry to a new key when its fingerprint grows.
func TestRegistryKeyForIdentity(t *testing.T) {
	rk, ok := parseRegistryKey("filestream::my-fingerprint-input::fingerprint::aabb")
	require.True(t, ok)

	newKey := rk.keyForIdentity("fingerprint::aabbccdd")
	assert.Equal(t, "filestream::my-fingerprint-input::fingerprint::aabbccdd", newKey,
		"keyForIdentity must keep the plugin/input prefix and swap the identity")

	// Fed the key's own identity, keyForIdentity reconstructs the original key.
	assert.Equal(t, "filestream::my-fingerprint-input::fingerprint::aabb", rk.keyForIdentity(rk.identity()),
		"keyForIdentity(identity()) must reconstruct the original key")
}
