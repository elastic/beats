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

//go:build !windows

package filestream

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	"github.com/elastic/beats/v7/libbeat/common/file"
<<<<<<< HEAD:filebeat/input/filestream/identifier_test_non_windows.go
=======
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
>>>>>>> 2a205834d (filebeat: fix misnamed OS-suffixed test files so they actually run (#51853)):filebeat/input/filestream/identifier_nonwindows_test.go
)

func TestFileIdentifierInodeMarker(t *testing.T) {
	t.Run("inode_marker file identifier", func(t *testing.T) {
		const markerContents = "unique_marker"
		dir := t.TempDir()
		markerFile, err := os.Create(filepath.Join(dir, "marker"))
		if err != nil {
			t.Fatalf("cannot create marker file for test: %v", err)
		}

		_, err = markerFile.Write([]byte(markerContents))
		if err != nil {
			t.Fatalf("cannot write marker contents to file: %v", err)
		}
		require.NoError(t, markerFile.Sync())

		c := conf.MustNewConfigFrom(map[string]interface{}{
			"identifier": map[string]interface{}{
				"inode_marker": map[string]interface{}{
					"path": markerFile.Name(),
				},
			},
		})
		var cfg testFileIdentifierConfig
		err = c.Unpack(&cfg)
		require.NoError(t, err)

		identifier, err := newFileIdentifier(cfg.Identifier, "", logptest.NewTestingLogger(t, ""))
		require.NoError(t, err)
		assert.Equal(t, inodeMarkerName, identifier.Name())

		tmpFile, err := os.Create(filepath.Join(dir, "target"))
		if err != nil {
			t.Fatalf("cannot create temporary file for test: %v", err)
		}

		fi, err := tmpFile.Stat()
		if err != nil {
			t.Fatalf("cannot stat temporary file for test: %v", err)
		}

		fsEvent := loginp.FSEvent{
			NewPath:    tmpFile.Name(),
			Descriptor: loginp.FileDescriptor{Info: file.ExtendFileInfo(fi)},
		}
		src := identifier.GetSource(fsEvent)

		osState := file.GetOSState(fi)
		assert.Equal(t, identifier.Name()+"::"+osState.InodeString()+"-"+markerContents, src.Name())

		const changedMarkerContents = "different_unique_marker"
		_, err = markerFile.WriteAt([]byte(changedMarkerContents), 0)
		if err != nil {
			t.Fatalf("cannot write marker contents to file: %v", err)
		}
		require.NoError(t, markerFile.Sync())

		src = identifier.GetSource(fsEvent)

		assert.Equal(t, identifier.Name()+"::"+osState.InodeString()+"-"+changedMarkerContents, src.Name())
	})
}
