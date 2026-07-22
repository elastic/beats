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

//go:build !integration

package kubernetes

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemoveGODEBUGKey(t *testing.T) {
	tests := []struct {
		name    string
		godebug string
		key     string
		want    string
	}{
		{
			name:    "empty",
			godebug: "",
			key:     "fips140",
			want:    "",
		},
		{
			name:    "only fips140",
			godebug: "fips140=only",
			key:     "fips140",
			want:    "",
		},
		{
			name:    "fips140 with other settings",
			godebug: "fips140=only,tlsmlkem=0",
			key:     "fips140",
			want:    "tlsmlkem=0",
		},
		{
			name:    "fips140 in middle",
			godebug: "tlsmlkem=0,fips140=on,http2client=0",
			key:     "fips140",
			want:    "tlsmlkem=0,http2client=0",
		},
		{
			name:    "no matching key",
			godebug: "tlsmlkem=0",
			key:     "fips140",
			want:    "tlsmlkem=0",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := removeGODEBUGKey(tc.godebug, tc.key)
			assert.Equal(t, tc.want, got, "unexpected GODEBUG after removing %q", tc.key)
		})
	}
}

func TestWithGODEBUGWithoutFIPS140Restores(t *testing.T) {
	const orig = "fips140=only,tlsmlkem=0"
	t.Setenv("GODEBUG", orig)

	var seenDuring string
	err := withGODEBUGWithoutFIPS140(func() error {
		seenDuring = os.Getenv("GODEBUG")
		return nil
	})
	require.NoError(t, err, "callback should succeed")
	assert.Equal(t, "tlsmlkem=0", seenDuring, "fips140 should be cleared while callback runs")
	assert.Equal(t, orig, os.Getenv("GODEBUG"), "GODEBUG should be restored after callback")
}
