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
//
// This file was contributed to by generative AI

//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.etcd.io/bbolt"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
)

type bboltRegistryState struct {
	TTL     time.Duration      `json:"TTL"`
	Updated time.Time          `json:"Updated"`
	Cursor  filestreamCursor   `json:"Cursor"`
	Meta    filestreamFileMeta `json:"Meta"`
}

type filestreamCursor struct {
	Offset int64 `json:"offset"`
	EOF    bool  `json:"eof"`
}

type filestreamFileMeta struct {
	Source         string `json:"source"`
	IdentifierName string `json:"identifier_name"`
}

var bboltCfg = `
filebeat.inputs:
  - type: filestream
    id: test-bbolt-shutdown
    enabled: true
    file_identity.native: ~
    prospector.scanner.fingerprint.enabled: false
    paths:
      - %q

filebeat.registry:
  type: bbolt
  path: registry
  flush: 24h

queue.mem.flush.timeout: 0s

output.file:
  path: %q
  filename: "output"
  rotate_on_startup: true

path.home: %q

logging.level: debug
`

func TestBBoltRegistrySyncedOnShutdown(t *testing.T) {
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)

	home := filebeat.TempDir()
	logfile := filepath.Join(home, "log.log")
	integration.WriteLogFile(t, logfile, 50, false)
	expectedOffset := int64(50 * 50) // 50 lines of 50 bytes

	cfg := fmt.Sprintf(bboltCfg, logfile, home, home)

	filebeat.WriteConfigFile(cfg)
	filebeat.Start()

	filebeat.WaitLogsContains(
		fmt.Sprintf("End of file reached: %s; Backoff now.", logfile),
		30*time.Second,
		"Filebeat did not reach EOF",
	)

	// Ensure the registrar processed all events before shutdown.
	filebeat.WaitPublishedEvents(10*time.Second, 50)

	// Ensure shutdown triggers persistence even with a large flush interval.
	filebeat.Stop()

	dbPath := filepath.Join(home, "data", "registry", "filebeat.db")
	_, err := os.Stat(dbPath)
	require.NoErrorf(t, err, "expected bbolt registry db to exist at %q", dbPath)

	states := readBBoltRegistryStates(t, dbPath)
	for _, st := range states {
		if st.Meta.Source != logfile {
			continue
		}
		require.EqualValuesf(t, expectedOffset, st.Cursor.Offset, "expected bbolt registry cursor offset to match file size for %q", logfile)
		return
	}
	require.Failf(t, "missing registry state", "expected bbolt registry to contain state for %q", logfile)
}

func readBBoltRegistryStates(t *testing.T, dbPath string) []bboltRegistryState {
	t.Helper()

	db, err := bbolt.Open(dbPath, 0o600, &bbolt.Options{
		ReadOnly: true,
		Timeout:  1 * time.Second,
	})
	require.NoError(t, err)
	defer db.Close()

	var states []bboltRegistryState
	err = db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("data"))
		require.NotNil(t, b, "bbolt data bucket must exist")

		return b.ForEach(func(_ []byte, v []byte) error {
			var st bboltRegistryState
			if err := json.Unmarshal(v, &st); err != nil {
				return err
			}
			states = append(states, st)
			return nil
		})
	})
	require.NoError(t, err)

	return states
}
