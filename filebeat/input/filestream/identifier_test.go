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
	"fmt"
	"io/ioutil"
	"os"
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

		src := identifier.GetSource(loginp.FSEvent{
			NewPath:    tmpFile.Name(),
			Descriptor: loginp.FileDescriptor{Info: file.ExtendFileInfo(fi)},
		})

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

		src := identifier.GetSource(loginp.FSEvent{
			NewPath:    tmpFile.Name(),
			Descriptor: loginp.FileDescriptor{Info: file.ExtendFileInfo(fi)},
		})

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

		identifier, err := newFileIdentifier(cfg.Identifier, "")
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

func FuzzFileIdentifier(f *testing.F) {
	identifierFmtWithSuffix := "%s::%s-%s"
	// The last %s is to not break the fmt.Sprintf call
	identifierFmtWithoutSuffix := "%s::%s%s"

	f.Add("foo")
	f.Add("bar")
	f.Add("世界")
	f.Fuzz(func(t *testing.T, suffix string) {
		identifierFmtStr := identifierFmtWithSuffix
		if suffix == "" {
			identifierFmtStr = identifierFmtWithoutSuffix
		}

		identifier, err := newFileIdentifier(nil, suffix)
		require.NoError(t, err)
		assert.Equal(t, DefaultIdentifierName, identifier.Name(), "identifier has got invalid name")

		tmpFile, err := os.CreateTemp("", "fuzz_file_identifier_native")
		if err != nil {
			t.Fatalf("cannot create temporary file for test: %v", err)
		}
		t.Cleanup(func() { os.Remove(tmpFile.Name()) })

		fi, err := tmpFile.Stat()
		if err != nil {
			t.Fatalf("cannot stat temporary file for test: %v", err)
		}

		src := identifier.GetSource(loginp.FSEvent{
			NewPath: tmpFile.Name(),
			Info:    fi,
		})

		got := src.Name()
		want := fmt.Sprintf(identifierFmtStr, identifier.Name(), file.GetOSState(fi).String(), suffix)
		if want != got {
			t.Fatalf("expecting file identifier to be %q, got %q instead", want, got)
		}
	})
}
