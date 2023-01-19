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

package backup

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/stretchr/testify/require"
)

func TestRegistryBackup(t *testing.T) {
	log := logp.NewLogger("backup-test")
	t.Run("creates backups for registry files including the checkpoint", func(t *testing.T) {
		regHome, cleanUp := createRegistryFiles(t, true)
		t.Cleanup(func() {
			cleanUp(t)
		})
		backuper := NewRegistryBackuper(log, regHome)

		err := backuper.Backup()
		require.NoError(t, err)
		requireRegistryBackups(t, regHome, 2)

		t.Run("creates the second round of backups for registry files", func(t *testing.T) {
			err := backuper.Backup()
			require.NoError(t, err)
			requireRegistryBackups(t, regHome, 4)
		})

		t.Run("removes all the created backups", func(t *testing.T) {
			err := backuper.Remove()
			require.NoError(t, err)
			requireRegistryBackups(t, regHome, 0)
		})
	})

	t.Run("creates backups for registry files without a checkpoint", func(t *testing.T) {
		regHome, cleanUp := createRegistryFiles(t, false)
		t.Cleanup(func() {
			cleanUp(t)
		})
		backuper := NewRegistryBackuper(log, regHome)

		err := backuper.Backup()
		require.NoError(t, err)
		requireRegistryBackups(t, regHome, 1)

		t.Run("creates the second round of backups for registry files", func(t *testing.T) {
			err := backuper.Backup()
			require.NoError(t, err)
			requireRegistryBackups(t, regHome, 2)
		})

		t.Run("removes all the created backups", func(t *testing.T) {
			err := backuper.Remove()
			require.NoError(t, err)
			requireRegistryBackups(t, regHome, 0)
		})
	})
}

func createRegistryFiles(t *testing.T, createCheckpoint bool) (regHome string, cleanUp func(t *testing.T)) {
	tmp, err := os.MkdirTemp(os.TempDir(), "backup-registry-test-*")
	require.NoError(t, err)

	registry, err := os.Create(filepath.Join(tmp, regLogFilename))
	require.NoError(t, err)
	defer registry.Close()

	_, err = registry.WriteString(registry.Name())
	require.NoError(t, err)

	if createCheckpoint {
		checkpointFilename := filepath.Join(tmp, fmt.Sprintf("%d.json", time.Now().Unix()))
		checkpoint, err := os.Create(checkpointFilename)
		require.NoError(t, err)
		defer checkpoint.Close()

		_, err = checkpoint.WriteString(checkpoint.Name())
		require.NoError(t, err)
	}

	return tmp, func(t *testing.T) {
		require.NoError(t, os.RemoveAll(tmp))
	}
}

func requireRegistryBackups(t *testing.T, regHome string, expectedCount int) {
	matches, err := filepath.Glob(regHome + "/*" + backupSuffix)
	require.NoError(t, err)
	require.Len(t, matches, expectedCount, "expected a different amount of created backups")
	for _, match := range matches {
		content, err := os.ReadFile(match)
		require.NoError(t, err)
		require.Contains(t, string(content), regHome)
	}
}
