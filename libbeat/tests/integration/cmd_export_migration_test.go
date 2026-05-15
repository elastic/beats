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

//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const migrationCfg = `
mockbeat:
output:
  console:
    enabled: true
logging:
  level: debug
`

// fieldsPath returns the absolute path to the testdata fields.yml used for
// migration alias tests.
func fieldsPath(t *testing.T) string {
	t.Helper()
	pwd, err := os.Getwd()
	require.NoError(t, err, "cannot get working directory")
	return filepath.Join(pwd, "../../template/testdata/fields.yml")
}

// stdoutContains returns true if match appears anywhere in the beat's stdout.
func stdoutContains(b *BeatProc, match string) bool {
	out, err := b.ReadStdout()
	if err != nil {
		return false
	}
	return strings.Contains(out, match)
}

// TestMigrationDefault verifies that when no migration flag is set, the
// exported template contains migration_alias_false but not migration_alias_true.
// Migration is disabled by default.
func TestMigrationDefault(t *testing.T) {
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(migrationCfg)
	mockbeat.Start(
		"export", "template",
		"-E", "setup.template.fields="+fieldsPath(t),
	)
	err := mockbeat.Cmd.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, mockbeat.Cmd.ProcessState.ExitCode(), "incorrect exit code")

	mockbeat.WaitStdOutContains("migration_alias_false", 5*time.Second)
	require.False(t, stdoutContains(mockbeat, "migration_alias_true"),
		"migration_alias_true should not appear when migration is disabled (default)")
}

// TestMigrationFalse verifies that when migration.6_to_7.enabled=false, the
// exported template contains migration_alias_false but not migration_alias_true.
func TestMigrationFalse(t *testing.T) {
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(migrationCfg)
	mockbeat.Start(
		"export", "template",
		"-E", "setup.template.fields="+fieldsPath(t),
		"-E", "migration.6_to_7.enabled=false",
	)
	err := mockbeat.Cmd.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, mockbeat.Cmd.ProcessState.ExitCode(), "incorrect exit code")

	mockbeat.WaitStdOutContains("migration_alias_false", 5*time.Second)
	require.False(t, stdoutContains(mockbeat, "migration_alias_true"),
		"migration_alias_true should not appear when migration.6_to_7.enabled=false")
}

// TestMigrationTrue verifies that when migration.6_to_7.enabled=true, the
// exported template contains both migration_alias_false and migration_alias_true.
func TestMigrationTrue(t *testing.T) {
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(migrationCfg)
	mockbeat.Start(
		"export", "template",
		"-E", "setup.template.fields="+fieldsPath(t),
		"-E", "migration.6_to_7.enabled=true",
	)
	err := mockbeat.Cmd.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, mockbeat.Cmd.ProcessState.ExitCode(), "incorrect exit code")

	mockbeat.WaitStdOutContains("migration_alias_false", 5*time.Second)
	mockbeat.WaitStdOutContains("migration_alias_true", 5*time.Second)
}
