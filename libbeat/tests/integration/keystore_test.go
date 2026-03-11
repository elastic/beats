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
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func addKeystoreSecret(t *testing.T, mockbeat *BeatProc, key, value string) {
	t.Helper()
	mockbeat.Start("keystore", "add", key, "--stdin")
	fmt.Fprint(mockbeat.stdin, value)
	require.NoError(t, mockbeat.stdin.Close(), "could not close mockbeat stdin")
	err := mockbeat.Cmd.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, mockbeat.Cmd.ProcessState.ExitCode(), "incorrect exit code")
}

func TestKeystoreWithPresentKey(t *testing.T) {
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	keystorePath := filepath.Join(mockbeat.TempDir(), "test.keystore")

	key := "mysecretpath"
	// Include a comma in the path, regression test for https://github.com/elastic/beats/issues/29789
	secret := filepath.Join(mockbeat.TempDir(), "thisisultra,secretpath")

	keystoreCfg := fmt.Sprintf(`
mockbeat:
name:
queue.mem:
  events: 32
  flush.min_events: 8
  flush.timeout: 0.1s
logging:
  level: debug
output.file:
  path: "${%s}"
  filename: "mockbeat"
  rotate_every_kb: 1000
keystore:
  path: %s
`, key, keystorePath)

	mockbeat.WriteConfigFile(keystoreCfg)

	mockbeat.Start("keystore", "create")
	mockbeat.WaitStdOutContains("Created mockbeat keystore", 10*time.Second)
	mockbeat.Stop()

	addKeystoreSecret(t, mockbeat, key, secret)

	mockbeat.Start()
	mockbeat.WaitLogsContains("ackloop:  done send ack", 60*time.Second)
	mockbeat.Stop()

	require.DirExists(t, secret)
}

func TestKeystoreWithKeyNotPresent(t *testing.T) {
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	keystorePath := filepath.Join(mockbeat.TempDir(), "test.keystore")

	keystoreCfg := fmt.Sprintf(`
mockbeat:
name:
queue.mem:
  events: 32
  flush.min_events: 8
  flush.timeout: 0.1s
logging:
  level: debug
output.elasticsearch:
  hosts: ["${elasticsearch_host}:9200"]
keystore:
  path: %s
`, keystorePath)

	mockbeat.WriteConfigFile(keystoreCfg)

	mockbeat.Start()
	err := mockbeat.Cmd.Wait()
	require.Error(t, err, "mockbeat must exit with an error")
	require.Equal(t, 1, mockbeat.Cmd.ProcessState.ExitCode(), "incorrect exit code")
	mockbeat.WaitStdErrContains("missing field", 10*time.Second)
}

func TestKeystoreWithNestedKey(t *testing.T) {
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	keystorePath := filepath.Join(mockbeat.TempDir(), "test.keystore")

	key := "output.elasticsearch.hosts.0"
	secret := filepath.Join(mockbeat.TempDir(), "myelasticsearchsecrethost")

	keystoreCfg := fmt.Sprintf(`
mockbeat:
name:
logging:
  level: debug
queue.mem:
  events: 32
  flush.min_events: 8
  flush.timeout: 0.1s
output.file:
  path: "${%s}"
  filename: "mockbeat"
  rotate_every_kb: 1000
keystore:
  path: %s
`, key, keystorePath)

	mockbeat.WriteConfigFile(keystoreCfg)

	mockbeat.Start("keystore", "create")
	mockbeat.WaitStdOutContains("Created mockbeat keystore", 10*time.Second)
	mockbeat.Stop()

	addKeystoreSecret(t, mockbeat, key, secret)

	mockbeat.Start()
	mockbeat.WaitLogsContains("ackloop:  done send ack", 60*time.Second)
	mockbeat.Stop()

	require.DirExists(t, secret)
}

func TestExportConfigWithKeystore(t *testing.T) {
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	keystorePath := filepath.Join(mockbeat.TempDir(), "test.keystore")

	key := "output.console.bulk_max_size"
	secret := "42"

	keystoreCfg := fmt.Sprintf(`
mockbeat:
name:
queue.mem:
  events: 32
  flush.min_events: 8
  flush.timeout: 0.1s
logging:
  level: debug
output.console:
  codec.json:
    pretty: true
keystore:
  path: %s
`, keystorePath)

	mockbeat.WriteConfigFile(keystoreCfg)

	mockbeat.Start("keystore", "create")
	mockbeat.WaitStdOutContains("Created mockbeat keystore", 10*time.Second)
	mockbeat.Stop()

	addKeystoreSecret(t, mockbeat, key, secret)

	mockbeat.Start("export", "config")
	err := mockbeat.Cmd.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, mockbeat.Cmd.ProcessState.ExitCode(), "incorrect exit code")

	stdout, err := mockbeat.ReadStdout()
	require.NoError(t, err)
	require.NotContains(t, stdout, secret, "exported config must not contain keystore secret value")
}
