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

package kafka

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFileTokenProvider(t *testing.T) {
	t.Run("returns error when credentials path not set", func(t *testing.T) {
		_, err := newFileTokenProvider("", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "credentials_path")
	})

	t.Run("succeeds with path and no extensions", func(t *testing.T) {
		dir := t.TempDir()
		credFile := filepath.Join(dir, "token")
		require.NoError(t, os.WriteFile(credFile, []byte("my-jwt"), 0600))

		p, err := newFileTokenProvider(credFile, nil)
		require.NoError(t, err)
		assert.Equal(t, credFile, p.credentialsPath)
		assert.Nil(t, p.extensions)
	})

	t.Run("succeeds with path and extensions", func(t *testing.T) {
		dir := t.TempDir()
		credFile := filepath.Join(dir, "token")
		require.NoError(t, os.WriteFile(credFile, []byte("my-jwt"), 0600))
		extensions := map[string]string{"logicalCluster": "lkc-abc", "identityPoolId": "pool-xyz"}

		p, err := newFileTokenProvider(credFile, extensions)
		require.NoError(t, err)
		assert.Equal(t, extensions, p.extensions)
	})
}

func TestFileTokenProviderToken(t *testing.T) {
	const tokenValue = "eyJhbGciOiJSUzI1NiJ9.test-payload"

	writeTokenFile := func(t *testing.T, content string) string {
		t.Helper()
		dir := t.TempDir()
		path := filepath.Join(dir, "token")
		require.NoError(t, os.WriteFile(path, []byte(content), 0600))
		return path
	}

	t.Run("returns token and nil extensions when none configured", func(t *testing.T) {
		path := writeTokenFile(t, tokenValue)
		p, err := newFileTokenProvider(path, nil)
		require.NoError(t, err)

		tok, err := p.Token()
		require.NoError(t, err)
		assert.Equal(t, tokenValue, tok.Token)
		assert.Nil(t, tok.Extensions)
	})

	t.Run("returns token with extensions", func(t *testing.T) {
		extensions := map[string]string{
			"logicalCluster": "lkc-abc123",
			"identityPoolId": "pool-xyz789",
		}
		path := writeTokenFile(t, tokenValue)
		p, err := newFileTokenProvider(path, extensions)
		require.NoError(t, err)

		tok, err := p.Token()
		require.NoError(t, err)
		assert.Equal(t, tokenValue, tok.Token)
		assert.Equal(t, extensions, tok.Extensions)
	})

	t.Run("trims whitespace from token", func(t *testing.T) {
		path := writeTokenFile(t, "  "+tokenValue+"\n")
		p, err := newFileTokenProvider(path, nil)
		require.NoError(t, err)

		tok, err := p.Token()
		require.NoError(t, err)
		assert.Equal(t, tokenValue, tok.Token)
	})

	t.Run("returns error when credentials file does not exist", func(t *testing.T) {
		p := &fileTokenProvider{credentialsPath: "/nonexistent/path/token"}
		_, err := p.Token()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "/nonexistent/path/token")
	})

	t.Run("re-reads file on each call to pick up rotated credentials", func(t *testing.T) {
		path := writeTokenFile(t, "first-token")
		p, err := newFileTokenProvider(path, nil)
		require.NoError(t, err)

		tok1, err := p.Token()
		require.NoError(t, err)
		assert.Equal(t, "first-token", tok1.Token)

		require.NoError(t, os.WriteFile(path, []byte("rotated-token"), 0600))

		tok2, err := p.Token()
		require.NoError(t, err)
		assert.Equal(t, "rotated-token", tok2.Token)
	})
}
