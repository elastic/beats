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

package tls

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	EmptyMD5    = "d41d8cd98f00b204e9800998ecf8427e"
	EmptySHA1   = "da39a3ee5e6b4b0d3255bfef95601890afd80709"
	EmptySHA256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
)

func TestGetFingerprintAlgorithm(t *testing.T) {
	for _, testCase := range []struct {
		requested, name string
		sum             string
	}{
		{"md5", "md5", EmptyMD5},
		{"sha1", "sha1", EmptySHA1},
		{"SHA-1", "sha1", EmptySHA1},
		{"SHA256", "sha256", EmptySHA256},
		{"sha-256", "sha256", EmptySHA256},
		{"md4", "", ""},
	} {
		result, err := GetFingerprintAlgorithm(testCase.requested)
		if len(testCase.name) == 0 {
			assert.Error(t, err, testCase.requested)
			continue
		}
		assert.Equal(t, nil, err, testCase.requested)
		assert.NotNil(t, result)
		assert.Equal(t, testCase.name, result.name)
		assert.Equal(t, testCase.sum, result.algo.Hash(nil))
	}
}
