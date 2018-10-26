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

package keystore

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	ucfg "github.com/elastic/go-ucfg"
)

func TestResolverWhenTheKeyDoesntExist(t *testing.T) {
	path := GetTemporaryKeystoreFile()
	defer os.Remove(path)

	keystore := CreateAnExistingKeystore(path)

	resolver := ResolverWrap(keystore)
	_, err := resolver("donotexist")
	assert.Equal(t, err, ucfg.ErrMissing)
}

func TestResolverWhenTheKeyExist(t *testing.T) {
	path := GetTemporaryKeystoreFile()
	defer os.Remove(path)

	keystore := CreateAnExistingKeystore(path)

	resolver := ResolverWrap(keystore)
	v, err := resolver("output.elasticsearch.password")
	assert.NoError(t, err)
	assert.Equal(t, v, "secret")
}

func TestObfuscatedResolver(t *testing.T) {
	path := GetTemporaryKeystoreFile()
	defer os.Remove(path)

	keystore := CreateAnExistingKeystore(path)
	resolver := ResolverWrap(keystore)
	v, err := resolver("output.elasticsearch.password")
	assert.NoError(t, err)
	assert.Equal(t, v, "secret")

	t.Run("when the key exist", func(t *testing.T) {
		obfuscated := ObfuscatedResolverWrap(resolver)
		v, err := obfuscated("output.elasticsearch.password")
		if !assert.NoError(t, err) {
			return
		}
		assert.Equal(t, v, "${output.elasticsearch.password}")
	})

	t.Run("when the key doesn't exist", func(t *testing.T) {
		obfuscated := ObfuscatedResolverWrap(resolver)
		_, err := obfuscated("donotexist")
		assert.Error(t, err)
	})
}
