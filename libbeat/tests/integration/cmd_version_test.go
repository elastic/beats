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
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCmdVersion(t *testing.T) {
	cfg := `
mockbeat:
name:
logging:
  level: debug
queue.mem:
  events: 4096
  flush.min_events: 8
  flush.timeout: 0.1s
output.console:
  code.json:
    pretty: true
`
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(cfg)
	mockbeat.Start("version")
	procState, err := mockbeat.Process.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, procState.ExitCode(), "incorrect exit code")
	mockbeat.WaitStdOutContains("mockbeat", 10*time.Second)
	mockbeat.WaitStdOutContains("version", 10*time.Second)
	mockbeat.WaitStdOutContains("9.9.9", 10*time.Second)
}
