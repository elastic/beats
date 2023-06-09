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
	"bytes"
	"crypto/sha256"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	"github.com/elastic/beats/v7/libbeat/common/file"
	conf "github.com/elastic/elastic-agent-libs/config"
)

type testFileIdentifierConfig struct {
	Identifier *conf.Namespace `config:"identifier"`
}

func TestFileIdentifier(t *testing.T) {
	t.Run("default file identifier", func(t *testing.T) {
		identifier, err := newFileIdentifier(nil, "")
		require.NoError(t, err)
		assert.Equal(t, DefaultIdentifierName, identifier.Name())

		tmpFile, err := ioutil.TempFile("", "test_file_identifier_native")
		if err != nil {
			t.Fatalf("cannot create temporary file for test: %v", err)
		}
		defer os.Remove(tmpFile.Name())

		fi, err := tmpFile.Stat()
		if err != nil {
			t.Fatalf("cannot stat temporary file for test: %v", err)
		}

		src, err := identifier.GetSource(loginp.FSEvent{
			NewPath: tmpFile.Name(),
			Info:    fi,
		})
		require.NoError(t, err)

		assert.Equal(t, identifier.Name()+"::"+file.GetOSState(fi).String(), src.Name())
	})

	t.Run("default file identifier with suffix", func(t *testing.T) {
		identifier, err := newFileIdentifier(nil, "my-suffix")
		require.NoError(t, err)
		assert.Equal(t, DefaultIdentifierName, identifier.Name())

		tmpFile, err := ioutil.TempFile("", "test_file_identifier_native")
		if err != nil {
			t.Fatalf("cannot create temporary file for test: %v", err)
		}
		defer os.Remove(tmpFile.Name())

		fi, err := tmpFile.Stat()
		if err != nil {
			t.Fatalf("cannot stat temporary file for test: %v", err)
		}

		src, err := identifier.GetSource(loginp.FSEvent{
			NewPath: tmpFile.Name(),
			Info:    fi,
		})
		require.NoError(t, err)

		assert.Equal(t, identifier.Name()+"::"+file.GetOSState(fi).String()+"-my-suffix", src.Name())
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

		identifier, err := newFileIdentifier(cfg.Identifier, "")
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
			src, err := identifier.GetSource(loginp.FSEvent{
				NewPath: test.newPath,
				OldPath: test.oldPath,
				Op:      test.operation,
			})
			require.NoError(t, err)
			assert.Equal(t, test.expectedSrc, src.Name())
		}
	})

	t.Run("fingerprint identifier", func(t *testing.T) {
		t.Run("cannot set length under SHA block size", func(t *testing.T) {
			c := conf.MustNewConfigFrom(map[string]interface{}{
				"identifier": map[string]interface{}{
					fingerprintName: map[string]interface{}{
						"length": sha256.BlockSize - 1,
					},
				},
			})
			var cfg testFileIdentifierConfig
			err := c.Unpack(&cfg)
			require.NoError(t, err)

			identifier, err := newFingerprintIdentifier(cfg.Identifier.Config())
			require.Error(t, err)
			require.Nil(t, identifier)
		})

		dir := t.TempDir()
		filePath := filepath.Join(dir, "file")

		testCases := []struct {
			name         string
			e            loginp.FSEvent
			configLength interface{}
			configOffset interface{}
			fileSize     int64
			expectedSrc  string
			expErr       error
		}{
			{
				name: "Returns error for a created file with a size under the fingerprint length",
				e: loginp.FSEvent{
					NewPath: filePath,
					Op:      loginp.OpCreate,
				},
				configLength: 256,
				fileSize:     128,
				expErr:       ErrFileSizeTooSmall,
			},
			{
				name: "Returns error for a created file with a size under the fingerprint offset and length",
				e: loginp.FSEvent{
					NewPath: filePath,
					Op:      loginp.OpCreate,
				},
				configLength: 256,
				configOffset: 64,
				fileSize:     300,
				expErr:       ErrFileSizeTooSmall,
			},
			{
				name: "Computes fingerprint for a created file with a default fingerprint length",
				e: loginp.FSEvent{
					NewPath: filePath,
					Op:      loginp.OpCreate,
				},
				fileSize: 1100,
				// SHA256 from the 'a' character repeated 1024 times
				expectedSrc: fingerprintName + identitySep + "2edc986847e209b4016e141a6dc8716d3207350f416969382d431539bf292e4a",
			},
			{
				name: "Computes fingerprint for a created file with an offset",
				e: loginp.FSEvent{
					NewPath: filePath,
					Op:      loginp.OpCreate,
				},
				fileSize:     256,
				configLength: 64,
				configOffset: 128,
				// SHA256 from the 'a' character repeated 64 times
				expectedSrc: fingerprintName + identitySep + "ffe054fe7ae0cb6dc65c3af9b61d5209f439851db43d0ba5997337df154668eb",
			},
			{
				name: "Computes fingerprint for a created file with enough bytes when length set to 256",
				e: loginp.FSEvent{
					NewPath: filePath,
					Op:      loginp.OpCreate,
				},
				configLength: 256,
				fileSize:     400,
				// SHA256 from the 'a' character repeated 256 times
				expectedSrc: fingerprintName + identitySep + "02d7160d77e18c6447be80c2e355c7ed4388545271702c50253b0914c65ce5fe",
			},
			{
				name: "Computes fingerprint for a renamed file using the new path",
				e: loginp.FSEvent{
					NewPath: filePath,
					OldPath: filePath,
					Op:      loginp.OpRename,
				},
				configLength: 256,
				fileSize:     400,
				// SHA256 from the 'a' character repeated 256 times
				expectedSrc: fingerprintName + identitySep + "02d7160d77e18c6447be80c2e355c7ed4388545271702c50253b0914c65ce5fe",
			},
			{
				name: "Computes fingerprint for a deleted file",
				e: loginp.FSEvent{
					OldPath: filePath,
					Op:      loginp.OpDelete,
				},
				configLength: 256,
				fileSize:     400,
				// SHA256 from the 'a' character repeated 256 times
				expectedSrc: fingerprintName + identitySep + "02d7160d77e18c6447be80c2e355c7ed4388545271702c50253b0914c65ce5fe",
			},
		}

		for _, test := range testCases {
			t.Run(test.name, func(t *testing.T) {
				c := conf.MustNewConfigFrom(map[string]interface{}{
					"identifier": map[string]interface{}{
						fingerprintName: map[string]interface{}{
							"offset": test.configOffset,
							"length": test.configLength,
						},
					},
				})
				var cfg testFileIdentifierConfig
				err := c.Unpack(&cfg)
				require.NoError(t, err)

				identifier, err := newFingerprintIdentifier(cfg.Identifier.Config())
				require.NoError(t, err)
				assert.Equal(t, fingerprintName, identifier.Name())

				content := bytes.Repeat([]byte{'a'}, int(test.fileSize))

				err = os.WriteFile(filePath, content, 0777)
				require.NoError(t, err)

				test.e.Info, err = os.Stat(filePath)
				require.NoError(t, err)

				src, err := identifier.GetSource(test.e)
				if test.expErr != nil {
					require.ErrorIs(t, err, test.expErr)
					return
				}
				require.NoError(t, err)
				assert.Equal(t, test.expectedSrc, src.Name())
			})
		}

		t.Run("Supports offset", func(t *testing.T) {
			c := conf.MustNewConfigFrom(map[string]interface{}{
				"identifier": map[string]interface{}{
					fingerprintName: map[string]interface{}{
						"offset": 64,
						"length": 128,
					},
				},
			})
			var cfg testFileIdentifierConfig
			err := c.Unpack(&cfg)
			require.NoError(t, err)

			identifier, err := newFingerprintIdentifier(cfg.Identifier.Config())
			require.NoError(t, err)
			assert.Equal(t, fingerprintName, identifier.Name())

			// different characters to mark the offset
			content := append(bytes.Repeat([]byte{'a'}, 64), bytes.Repeat([]byte{'b'}, 128)...)

			err = os.WriteFile(filePath, content, 0777)
			require.NoError(t, err)

			e := loginp.FSEvent{
				NewPath: filePath,
				Op:      loginp.OpCreate,
			}
			e.Info, err = os.Stat(filePath)
			require.NoError(t, err)

			src, err := identifier.GetSource(e)
			require.NoError(t, err)
			expectedSrc := fingerprintName + identitySep + "70ae1c5307f5250d5cb9e40742ba9613fcdf9b8d9eb6dd393330443b2d5effbd"
			assert.Equal(t, expectedSrc, src.Name())
		})
	})
}
