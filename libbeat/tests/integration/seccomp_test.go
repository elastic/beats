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
	"runtime"
	"strings"
	"testing"
	"time"
)

// isSeccompSupported returns true if the current system supports seccomp:
// Linux kernel >= 3.17 on i386, amd64, or arm64.
func isSeccompSupported() bool {
	if runtime.GOOS != "linux" {
		return false
	}

	switch runtime.GOARCH {
	case "386", "amd64", "arm64":
	default:
		return false
	}

	data, err := os.ReadFile("/proc/sys/kernel/osrelease")
	if err != nil {
		return false
	}

	release := strings.TrimSpace(string(data))
	var major, minor int
	if _, err := fmt.Sscanf(release, "%d.%d", &major, &minor); err != nil {
		return false
	}

	return major > 3 || (major == 3 && minor >= 17)
}

var SeccompCfg = `
mockbeat:
name:
queue.mem:
  events: 4096
  flush.min_events: 8
  flush.timeout: 0.1s
output.console:
  code.json:
    pretty: false
logging:
  level: debug
`

func TestSeccompInstalled(t *testing.T) {
	if !isSeccompSupported() {
		t.Skip("Requires Linux 3.17 or greater and i386/amd64/arm64 architecture")
	}

	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(SeccompCfg)
	mockbeat.Start("-N")
	mockbeat.WaitLogsContains("Syscall filter successfully installed", 60*time.Second)
	mockbeat.Stop()
}
