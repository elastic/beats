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
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/stretchr/testify/require"
)

func TestFileBackup(t *testing.T) {
	log := logp.NewLogger("backup-test")
	files := createFiles(t, 3)

	backuper := NewFileBackuper(log, files)

	t.Run("creates exactly one backup per given file", func(t *testing.T) {
		err := backuper.Backup()
		require.NoError(t, err)
		requireBackups(t, files, 1)
	})

	t.Run("creates second round of backups", func(t *testing.T) {
		// there is a unix time with nanosecond precision in the filename
		// we can create only one backup per nanosecond
		// if there is already a file created in the same nanosecond, the backup fails
		time.Sleep(time.Microsecond)

		err := backuper.Backup()
		require.NoError(t, err)

		requireBackups(t, files, 2)
	})

	t.Run("removes all created backups", func(t *testing.T) {
		err := backuper.Remove()
		require.NoError(t, err)

		requireBackups(t, files, 0)
	})
}

func createFiles(t *testing.T, count int) (created []string) {
	t.Helper()

	tmp := t.TempDir()

	for i := 0; i < count; i++ {
		file, err := os.CreateTemp(tmp, "file-*")
		require.NoError(t, err)
		_, err = file.WriteString(file.Name())
		require.NoError(t, err)
		file.Close()
		created = append(created, file.Name())
	}

	return created
}

func requireBackups(t *testing.T, files []string, expectedCount int) {
	for _, file := range files {
		matches, err := filepath.Glob(file + "-*" + backupSuffix)
		require.NoError(t, err)
		require.Len(t, matches, expectedCount, "expected a different amount of created backups")
		for _, match := range matches {
			content, err := os.ReadFile(match)
			require.NoError(t, err)
			require.Equal(t, file, string(content))
		}
	}
}
