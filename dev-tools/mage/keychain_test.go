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

// +build darwin

package mage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAppleKeychainListIdentities(t *testing.T) {
	idents, err := _appleKeychain.ListIdentities()
	if err != nil {
		t.Fatal(err)
	}

	assert.NotZero(t, idents)

	for i, ident := range idents {
		t.Log(i, ident)
	}
}

func TestGetAppleSigningInfo(t *testing.T) {
	signingInfo, err := GetAppleSigningInfo()
	if err != nil {
		t.Fatal(err)
	}

	if assert.NotNil(t, signingInfo) {
		assert.False(t, signingInfo.Sign)
	}
}
