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
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var cfg = `
mockbeat:
name:
queue.mem:
  events: 4096
  flush.min_events: 8
  flush.timeout: 0.1s
output.console:
  code.json:
    pretty: true
keystore:
  path: %s
`

func TestKeystoreCreate(t *testing.T) {
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test", "keystore", "create")
	keystorePath := filepath.Join(mockbeat.TempDir(), "test.keystore")
	mockbeat.WriteConfigFile(fmt.Sprintf(cfg, keystorePath))
	mockbeat.Start()
	mockbeat.WaitStdOutContains("Created mockbeat keystore", 10*time.Second)
	require.FileExists(t, keystorePath)
}

func TestKeystoreCreateForce(t *testing.T) {
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test", "keystore", "create", "--force")
	keystorePath := filepath.Join(mockbeat.TempDir(), "test.keystore")
	mockbeat.WriteConfigFile(fmt.Sprintf(cfg, keystorePath))
	mockbeat.Start()
	mockbeat.WaitStdOutContains("Created mockbeat keystore", 10*time.Second)
	mockbeat.Stop()
	require.FileExists(t, keystorePath)
	keystore1, err := os.ReadFile(keystorePath)
	require.NoError(t, err)

	mockbeat.Start()
	mockbeat.WaitStdOutContains("Created mockbeat keystore", 10*time.Second)
	require.FileExists(t, keystorePath)
	keystore2, err := os.ReadFile(keystorePath)
	require.NoError(t, err)
	require.NotEqual(t, keystore1, keystore2, "keystores should be different")
}

func TestKeystoreRemoveNoKeyNoKeystore(t *testing.T) {
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test", "keystore", "remove", "mykey")
	keystorePath := filepath.Join(mockbeat.TempDir(), "test.keystore")
	mockbeat.WriteConfigFile(fmt.Sprintf(cfg, keystorePath))
	mockbeat.Start()
	mockbeat.WaitStdErrContains("keystore doesn't exist.", 10*time.Second)
}

func TestKeystoreRemoveNoExistingKey(t *testing.T) {
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	keystorePath := filepath.Join(mockbeat.TempDir(), "test.keystore")
	mockbeat.WriteConfigFile(fmt.Sprintf(cfg, keystorePath))
	mockbeat.Start("keystore", "create")
	mockbeat.WaitStdOutContains("Created mockbeat keystore", 10*time.Second)
	mockbeat.Stop()

	mockbeat.Start("keystore", "remove", "mykey")
	mockbeat.WaitStdErrContains("could not find key 'mykey' in the keystore", 10*time.Second)
}

func TestKeystoreRemoveMultipleExistingKeys(t *testing.T) {
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	keystorePath := filepath.Join(mockbeat.TempDir(), "test.keystore")
	mockbeat.WriteConfigFile(fmt.Sprintf(cfg, keystorePath))
	mockbeat.Start("keystore", "create")
	mockbeat.WaitStdOutContains("Created mockbeat keystore", 10*time.Second)
	mockbeat.Stop()

	mockbeat.Start("keystore", "add", "key1", "--stdin")

	fmt.Fprintf(mockbeat.stdin, "pass1")
	require.NoError(t, mockbeat.stdin.Close(), "could not close mockbeat stdin")
	procState, err := mockbeat.Process.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, procState.ExitCode(), "incorrect exit code")

	mockbeat.Start("keystore", "add", "key2", "--stdin")
	fmt.Fprintf(mockbeat.stdin, "pass2")
	require.NoError(t, mockbeat.stdin.Close(), "could not close mockbeat stdin")
	procState, err = mockbeat.Process.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, procState.ExitCode(), "incorrect exit code")

	mockbeat.Start("keystore", "add", "key3", "--stdin")
	fmt.Fprintf(mockbeat.stdin, "pass3")
	require.NoError(t, mockbeat.stdin.Close(), "could not close mockbeat stdin")
	procState, err = mockbeat.Process.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, procState.ExitCode(), "incorrect exit code")

	mockbeat.Start("keystore", "remove", "key2", "key3")
	procState, err = mockbeat.Process.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, procState.ExitCode(), "incorrect exit code")

	mockbeat.Start("keystore", "list")
	procState, err = mockbeat.Process.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, procState.ExitCode(), "incorrect exit code")
	mockbeat.WaitStdOutContains("key1", 10*time.Second)
}

func TestKeystoreList(t *testing.T) {
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	keystorePath := filepath.Join(mockbeat.TempDir(), "test.keystore")
	mockbeat.WriteConfigFile(fmt.Sprintf(cfg, keystorePath))
	mockbeat.Start("keystore", "create")
	mockbeat.WaitStdOutContains("Created mockbeat keystore", 10*time.Second)
	mockbeat.Stop()

	mockbeat.Start("keystore", "add", "key1", "--stdin")
	fmt.Fprintf(mockbeat.stdin, "pass1")
	require.NoError(t, mockbeat.stdin.Close(), "could not close mockbeat stdin")
	procState, err := mockbeat.Process.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, procState.ExitCode(), "incorrect exit code")

	mockbeat.Start("keystore", "add", "key2", "--stdin")
	fmt.Fprintf(mockbeat.stdin, "pass2")
	require.NoError(t, mockbeat.stdin.Close(), "could not close mockbeat stdin")
	procState, err = mockbeat.Process.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, procState.ExitCode(), "incorrect exit code")

	mockbeat.Start("keystore", "add", "key3", "--stdin")
	fmt.Fprintf(mockbeat.stdin, "pass3")
	require.NoError(t, mockbeat.stdin.Close(), "could not close mockbeat stdin")
	procState, err = mockbeat.Process.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, procState.ExitCode(), "incorrect exit code")

	mockbeat.Start("keystore", "list")
	procState, err = mockbeat.Process.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, procState.ExitCode(), "incorrect exit code")

	mockbeat.WaitStdOutContains("key1", 10*time.Second)
	mockbeat.WaitStdOutContains("key2", 10*time.Second)
	mockbeat.WaitStdOutContains("key3", 10*time.Second)
}

func TestKeystoreListEmptyKeystore(t *testing.T) {
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	keystorePath := filepath.Join(mockbeat.TempDir(), "test.keystore")
	mockbeat.WriteConfigFile(fmt.Sprintf(cfg, keystorePath))
	mockbeat.Start("keystore", "list")
	procState, err := mockbeat.Process.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, procState.ExitCode(), "incorrect exit code")
}

func TestKeystoreAddSecretFromStdin(t *testing.T) {
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	keystorePath := filepath.Join(mockbeat.TempDir(), "test.keystore")
	mockbeat.WriteConfigFile(fmt.Sprintf(cfg, keystorePath))

	mockbeat.Start("keystore", "create")
	procState, err := mockbeat.Process.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, procState.ExitCode(), "incorrect exit code")

	mockbeat.Start("keystore", "add", "key1", "--stdin")
	fmt.Fprintf(mockbeat.stdin, "pass1")
	require.NoError(t, mockbeat.stdin.Close(), "could not close mockbeat stdin")
	procState, err = mockbeat.Process.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, procState.ExitCode(), "incorrect exit code")
}

func TestKeystoreUpdateForce(t *testing.T) {
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	keystorePath := filepath.Join(mockbeat.TempDir(), "test.keystore")
	mockbeat.WriteConfigFile(fmt.Sprintf(cfg, keystorePath))
	mockbeat.Start("keystore", "create")
	procState, err := mockbeat.Process.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, procState.ExitCode(), "incorrect exit code")

	mockbeat.Start("keystore", "add", "key1", "--stdin")
	fmt.Fprintf(mockbeat.stdin, "pass1")
	require.NoError(t, mockbeat.stdin.Close(), "could not close mockbeat stdin")
	procState, err = mockbeat.Process.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, procState.ExitCode(), "incorrect exit code")

	mockbeat.Start("keystore", "add", "key1", "--force", "--stdin")
	fmt.Fprintf(mockbeat.stdin, "pass2")
	require.NoError(t, mockbeat.stdin.Close(), "could not close mockbeat stdin")
	procState, err = mockbeat.Process.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, procState.ExitCode(), "incorrect exit code")
}
