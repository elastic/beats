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
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	"github.com/elastic/beats/v7/libbeat/common/file"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

type testFileIdentifierConfig struct {
	Identifier *conf.Namespace `config:"identifier"`
}

func TestFileIdentifier(t *testing.T) {
	t.Run("native file identifier", func(t *testing.T) {
		cfg := conf.MustNewConfigFrom(`native: ~`)
		ns := conf.Namespace{}
		if err := cfg.Unpack(&ns); err != nil {
			t.Fatalf("cannot unpack config into conf.Namespace: %s", err)
		}
		identifier, err := newFileIdentifier(&ns, "", logptest.NewTestingLogger(t, ""))
		require.NoError(t, err)
		assert.Equal(t, DefaultIdentifierName, identifier.Name())

		tmpFile, err := os.CreateTemp("", "test_file_identifier_native")
		if err != nil {
			t.Fatalf("cannot create temporary file for test: %v", err)
		}
		defer os.Remove(tmpFile.Name())

		fi, err := tmpFile.Stat()
		if err != nil {
			t.Fatalf("cannot stat temporary file for test: %v", err)
		}

		src := identifier.GetSource(loginp.FSEvent{
			NewPath:    tmpFile.Name(),
			Descriptor: loginp.FileDescriptor{Info: file.ExtendFileInfo(fi)},
		})

		assert.Equal(t, identifier.Name()+"::"+file.GetOSState(fi).Identifier(), src.Name())
	})

	t.Run("native file identifier with suffix", func(t *testing.T) {
		cfg := conf.MustNewConfigFrom(`native: ~`)
		ns := conf.Namespace{}
		if err := cfg.Unpack(&ns); err != nil {
			t.Fatalf("cannot unpack config into conf.Namespace: %s", err)
		}
		identifier, err := newFileIdentifier(&ns, "my-suffix", logptest.NewTestingLogger(t, ""))
		require.NoError(t, err)
		assert.Equal(t, DefaultIdentifierName, identifier.Name())

		tmpFile, err := os.CreateTemp("", "test_file_identifier_native")
		if err != nil {
			t.Fatalf("cannot create temporary file for test: %v", err)
		}
		defer os.Remove(tmpFile.Name())

		fi, err := tmpFile.Stat()
		if err != nil {
			t.Fatalf("cannot stat temporary file for test: %v", err)
		}

		src := identifier.GetSource(loginp.FSEvent{
			NewPath:    tmpFile.Name(),
			Descriptor: loginp.FileDescriptor{Info: file.ExtendFileInfo(fi)},
		})

		assert.Equal(t, identifier.Name()+"::"+file.GetOSState(fi).Identifier()+"-my-suffix", src.Name())
	})

	t.Run("path identifier", func(t *testing.T) {
		c := conf.MustNewConfigFrom(map[string]interface{}{
			"identifier": map[string]interface{}{
				"path": nil,
			},
		})
		var cfg testFileIdentifierConfig
		err := c.Unpack(&cfg)
		require.NoError(t, err)

		identifier, err := newFileIdentifier(cfg.Identifier, "", logptest.NewTestingLogger(t, ""))
		require.NoError(t, err)
		assert.Equal(t, pathName, identifier.Name())

		testCases := []struct {
			newPath     string
			oldPath     string
			operation   loginp.Operation
			expectedSrc string
		}{
			{
				newPath:     "/path/to/file",
				expectedSrc: "path::/path/to/file",
			},
			{
				newPath:     "/new/path/to/file",
				oldPath:     "/old/path/to/file",
				operation:   loginp.OpRename,
				expectedSrc: "path::/new/path/to/file",
			},
			{
				oldPath:     "/old/path/to/file",
				operation:   loginp.OpDelete,
				expectedSrc: "path::/old/path/to/file",
			},
		}

		for _, test := range testCases {
			src := identifier.GetSource(loginp.FSEvent{
				NewPath: test.newPath,
				OldPath: test.oldPath,
				Op:      test.operation,
			})
			assert.Equal(t, test.expectedSrc, src.Name())
		}
	})

	t.Run("fingerprint identifier", func(t *testing.T) {
		c := conf.MustNewConfigFrom(map[string]interface{}{
			"identifier": map[string]interface{}{
				"fingerprint": nil,
			},
		})
		var cfg testFileIdentifierConfig
		err := c.Unpack(&cfg)
		require.NoError(t, err)

		identifier, err := newFileIdentifier(cfg.Identifier, "", logptest.NewTestingLogger(t, ""))
		require.NoError(t, err)
		assert.Equal(t, fingerprintName, identifier.Name())

		testCases := []struct {
			newPath     string
			oldPath     string
			operation   loginp.Operation
			desc        loginp.FileDescriptor
			expectedSrc string
		}{
			{
				newPath:     "/path/to/file",
				desc:        loginp.FileDescriptor{Fingerprint: "fingerprintvalue"},
				expectedSrc: fingerprintName + "::fingerprintvalue",
			},
			{
				newPath:     "/new/path/to/file",
				oldPath:     "/old/path/to/file",
				operation:   loginp.OpRename,
				desc:        loginp.FileDescriptor{Fingerprint: "fingerprintvalue"},
				expectedSrc: fingerprintName + "::fingerprintvalue",
			},
			{
				oldPath:     "/old/path/to/file",
				operation:   loginp.OpDelete,
				desc:        loginp.FileDescriptor{Fingerprint: "fingerprintvalue"},
				expectedSrc: fingerprintName + "::fingerprintvalue",
			},
		}

		for _, test := range testCases {
			src := identifier.GetSource(loginp.FSEvent{
				NewPath:    test.newPath,
				OldPath:    test.oldPath,
				Op:         test.operation,
				Descriptor: test.desc,
			})
			assert.Equal(t, test.expectedSrc, src.Name())
		}
	})
}

// TestBoundFingerprintKey documents the bounded-key optimization: a final
// SHA-256 fingerprint is used verbatim in the registry key (preserving
// byte-identical state with static fingerprint), while a growing raw-hex
// fingerprint is hashed to a fixed 64-char key so it cannot bloat the memlog
// WAL or leak file content via the key.
func TestBoundFingerprintKey(t *testing.T) {
	const sha = "2edc986847e209b4016e141a6dc8716d3207350f416969382d431539bf292e4a"

	t.Run("final SHA-256 is used as-is", func(t *testing.T) {
		got := boundFingerprintKey(loginp.FileDescriptor{Fingerprint: sha, FingerprintGrowing: false})
		assert.Equal(t, sha, got, "a final fingerprint must be byte-identical to the static-fingerprint key")
	})

	t.Run("growing raw-hex is hashed to a bounded 64-char key", func(t *testing.T) {
		raw := hex.EncodeToString([]byte("small file header that is below the threshold"))
		got := boundFingerprintKey(loginp.FileDescriptor{Fingerprint: raw, FingerprintGrowing: true})
		sum := sha256.Sum256([]byte(raw))
		assert.Equal(t, hex.EncodeToString(sum[:]), got, "growing key must be sha256(rawHex)")
		assert.Len(t, got, 64, "bounded key must be a fixed 64-char hex string")
		assert.NotContains(t, got, raw, "the raw file bytes must not appear in the key")
	})

	t.Run("a long growing fingerprint stays bounded", func(t *testing.T) {
		raw := strings.Repeat("ab", 1024) // 2048 chars, as a near-threshold raw-hex fp
		got := boundFingerprintKey(loginp.FileDescriptor{Fingerprint: raw, FingerprintGrowing: true})
		assert.Len(t, got, 64, "the key length must not grow with the fingerprint length")
	})

	t.Run("empty fingerprint passes through", func(t *testing.T) {
		assert.Empty(t, boundFingerprintKey(loginp.FileDescriptor{Fingerprint: "", FingerprintGrowing: true}))
	})
}

// TestGrowingRawFingerprint verifies the value-side helper: the raw fingerprint
// is persisted (in fileMeta.Fingerprint) only while the file is still growing.
func TestGrowingRawFingerprint(t *testing.T) {
	raw := "41414141"
	assert.Equal(t, raw, growingRawFingerprint(loginp.FileDescriptor{Fingerprint: raw, FingerprintGrowing: true}),
		"a growing descriptor must persist its raw fingerprint for prefix matching")
	assert.Empty(t, growingRawFingerprint(loginp.FileDescriptor{Fingerprint: raw, FingerprintGrowing: false}),
		"a final descriptor must not persist a raw fingerprint (keeps the entry byte-identical to static)")
}
