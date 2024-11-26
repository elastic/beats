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
	"path"
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
)

func TestSystemLogsCanUseLogInput(t *testing.T) {
	t.Skip("The system module is not using the system-logs input at the moment")
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)
	workDir := filebeat.TempDir()
	copyModulesDir(t, workDir)

	logFilePath := path.Join(workDir, "syslog")
	integration.GenerateLogFile(t, logFilePath, 5, false)
	yamlCfg := fmt.Sprintf(systemModuleCfg, logFilePath, logFilePath, workDir)

	filebeat.WriteConfigFile(yamlCfg)
	filebeat.Start()

	filebeat.WaitForLogs(
		"using log input because file(s) was(were) found",
		10*time.Second,
		"system-logs did not select the log input")
	filebeat.WaitForLogs(
		"Harvester started for paths:",
		10*time.Second,
		"system-logs did not start the log input")
}
