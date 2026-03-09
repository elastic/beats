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

//go:build integration && !windows

package integration

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
)

func TestFilestreamHasOwnerAndGroup(t *testing.T) {
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)
	tempDir := filebeat.TempDir()
	logFilePath := filepath.Join(tempDir, "input.log")

	integration.WriteLogFile(t, logFilePath, 25, false)

	cfg := fmt.Sprintf(`
filebeat.inputs:
  - type: filestream
    id: TestFilestreamHasOwnerAndGroup
    paths:
      - %s
    include_file_owner_name: true
    include_file_owner_group_name: true

logging:
  level: debug
  metrics:
    enabled: false

output:
  file:
    path: ${path.home}
    filename: "output"
    rotate_on_startup: false
`, logFilePath)

	filebeat.WriteConfigFile(cfg)
	filebeat.Start()

	// Get logFilePath owner and group
	logFileInfo, err := os.Stat(logFilePath)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	stat, ok := logFileInfo.Sys().(*syscall.Stat_t)
	if !ok {
		t.Fatalf("Failed to stat file")
	}

	logFileOwner, err := user.LookupId(strconv.FormatUint(uint64(stat.Uid), 10))
	if err != nil {
		t.Fatalf("Failed to lookup uid %v", err)
	}
	logFileGroup, err := user.LookupGroupId(strconv.FormatUint(uint64(stat.Gid), 10))
	if err != nil {
		t.Fatalf("Failed to lookup gid %v", err)
	}

	filebeat.WaitPublishedEvents(30*time.Second, 25)

	type evt struct {
		Log struct {
			File struct {
				Owner string `json:"owner"`
				Group string `json:"group"`
			} `json:"file"`
		} `json:"log"`
	}
	evts := integration.GetEventsFromFileOutput[evt](filebeat, 5, false)
	for _, e := range evts {
		assert.Equal(t, logFileOwner.Username, e.Log.File.Owner)
		assert.Equal(t, logFileGroup.Name, e.Log.File.Group)
	}
}
