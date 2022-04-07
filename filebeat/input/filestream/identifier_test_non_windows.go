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
// +build !windows

package filestream

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	loginp "github.com/elastic/beats/v8/filebeat/input/filestream/internal/input-logfile"
	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/common/file"
)

func TestFileIdentifierInodeMarker(t *testing.T) {
	t.Run("inode_marker file identifier", func(t *testing.T) {
		const markerContents = "unique_marker"
		markerFile, err := ioutil.TempFile("", "test_file_identifier_inode_marker_identifier")
		if err != nil {
			t.Fatalf("cannot create marker file for test: %v", err)
		}
		defer os.Remove(markerFile.Name())

		_, err = markerFile.Write([]byte(markerContents))
		if err != nil {
			t.Fatalf("cannot write marker contents to file: %v", err)
		}
		markerFile.Sync()

		c := common.MustNewConfigFrom(map[string]interface{}{
			"identifier": map[string]interface{}{
				"inode_marker": map[string]interface{}{
					"path": markerFile.Name(),
				},
			},
		})
		var cfg testFileIdentifierConfig
		err = c.Unpack(&cfg)
		require.NoError(t, err)

		identifier, err := newFileIdentifier(cfg.Identifier)
		require.NoError(t, err)
		assert.Equal(t, inodeMarkerName, identifier.Name())

		tmpFile, err := ioutil.TempFile("", "test_file_identifier_inode_marker")
		if err != nil {
			t.Fatalf("cannot create temporary file for test: %v", err)
		}
		defer os.Remove(tmpFile.Name())

		fi, err := tmpFile.Stat()
		if err != nil {
			t.Fatalf("cannot stat temporary file for test: %v", err)
		}

		fsEvent := loginp.FSEvent{
			NewPath: tmpFile.Name(),
			Info:    fi,
		}
		src := identifier.GetSource(fsEvent)

		osState := file.GetOSState(fi)
		assert.Equal(t, identifier.Name()+"::"+osState.InodeString()+"-"+markerContents, src.Name())

		const changedMarkerContents = "different_unique_marker"
		_, err = markerFile.WriteAt([]byte(changedMarkerContents), 0)
		if err != nil {
			t.Fatalf("cannot write marker contents to file: %v", err)
		}
		markerFile.Sync()

		src = identifier.GetSource(fsEvent)

		assert.Equal(t, identifier.Name()+"::"+osState.InodeString()+"-"+changedMarkerContents, src.Name())
	})
}
