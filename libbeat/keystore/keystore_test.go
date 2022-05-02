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
	"github.com/stretchr/testify/require"

	ucfg "github.com/elastic/go-ucfg"
	"github.com/elastic/go-ucfg/parse"
)

func TestResolverWhenTheKeyDoesntExist(t *testing.T) {
	path := GetTemporaryKeystoreFile()
	defer os.Remove(path)

	keystore := CreateAnExistingKeystore(path)

	resolver := ResolverWrap(keystore)
	_, _, err := resolver("donotexist")
	assert.Equal(t, err, ucfg.ErrMissing)
}

func TestResolverWhenTheKeyExist(t *testing.T) {
	path := GetTemporaryKeystoreFile()
	defer os.Remove(path)

	keystore := CreateAnExistingKeystore(path)

	resolver := ResolverWrap(keystore)
	v, pCfg, err := resolver("output.elasticsearch.password")
	require.NoError(t, err)
	require.Equal(t, pCfg, parseConfig)

	// Cheat a bit by reproducing part of the go-ucfg dynamic variable resolution process here. The
	// config returned by the resolver will be used with a call to the go-ucfg parser. The
	// public entrypoint is the ValueWithConfig function below. Make sure the parsed value is
	// correct. See https://github.com/elastic/go-ucfg/blob/fc880abbe1f30b653d113da96a4a7e82743c0cc1/types.go#L539
	iface, err := parse.ValueWithConfig(v, pCfg)
	require.NoError(t, err)
	t.Logf("%v", iface)

	secret, ok := iface.(string)
	require.True(t, ok, "parsed secret is not a string")
	require.Equal(t, string(secretValue), secret)
}
