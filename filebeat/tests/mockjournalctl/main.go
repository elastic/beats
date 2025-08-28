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

package main

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"time"
)

// This simple mock for journalclt that can be used to test error conditions.
// If a file called 'exit' exists in the same folder as the Filebeat binary
// then this mock will exit immediately, otherwise it will generate errors
// randomly and eventually exit.
//
// The easiest way to use this mock is to compile it as 'journalctl' and
// manipulate the $PATH environment variable from the Filebeat process you're
// testing.
func main() {
	if _, err := os.Stat("exit"); err == nil {
		os.Exit(42)
	}

	fatalChance := 10
	stdoutTicker := time.NewTicker(time.Second)
	stderrTicker := time.NewTicker(time.Second)
	fatalTicker := time.NewTicker(time.Second)

	jsonEncoder := json.NewEncoder(os.Stdout)
	count := uint64(0)
	for {
		count++
		select {
		case t := <-stdoutTicker.C:
			mockData["MESSAGE"] = fmt.Sprintf("Count: %010d", count)
			mockData["__CURSOR"] = fmt.Sprintf("cursor-%010d-now-%s", count, time.Now().Format(time.RFC3339Nano))
			mockData["__REALTIME_TIMESTAMP"] = fmt.Sprintf("%d", t.UnixMicro())
			jsonEncoder.Encode(mockData) //nolint:errcheck // it will never fail and it's a mock for testing.
		case t := <-stderrTicker.C:
			fmt.Fprintf(os.Stderr, "a random error at %s, count: %010d\n", t.Format(time.RFC3339), count)
		case t := <-fatalTicker.C:
			chance := rand.IntN(100)
			if chance < fatalChance {
				fmt.Fprintf(os.Stderr, "fatal error, exiting at %s\n", t.Format(time.RFC3339))
				os.Exit(rand.IntN(125))
			}
		}
	}
}

var mockData = map[string]string{
	"MESSAGE":                "Count: 0000000001",
	"PRIORITY":               "6",
	"SYSLOG_IDENTIFIER":      "TestRestartsJournalctlOnError",
	"_AUDIT_LOGINUID":        "1000",
	"_AUDIT_SESSION":         "2",
	"_BOOT_ID":               "567980bb85ae41da8518f409570b0cb9",
	"_CAP_EFFECTIVE":         "0",
	"_CMDLINE":               "/bin/cat",
	"_COMM":                  "cat",
	"_EXE":                   "/usr/bin/cat",
	"_GID":                   "1000",
	"_HOSTNAME":              "millennium-falcon",
	"_MACHINE_ID":            "851f339d77174301b29e417ecb2ec6a8",
	"_PID":                   "235728",
	"_RUNTIME_SCOPE":         "system",
	"_STREAM_ID":             "92765bf7ba214e23a2ee986d76578bef",
	"_SYSTEMD_CGROUP":        "/user.slice/user-1000.slice/session-2.scope",
	"_SYSTEMD_INVOCATION_ID": "89e7dffc4a0140f086a3171235fae8d9",
	"_SYSTEMD_OWNER_UID":     "1000",
	"_SYSTEMD_SESSION":       "2",
	"_SYSTEMD_SLICE":         "user-1000.slice",
	"_SYSTEMD_UNIT":          "session-2.scope",
	"_SYSTEMD_USER_SLICE":    "-.slice",
	"_TRANSPORT":             "stdout",
	"_UID":                   "1000",
	"__CURSOR":               "s=e82795fad4ce42b79fb3da0866d91f7e;i=4eb1b1;b=567980bb85ae41da8518f409570b0cb9;m=2bd4e2166;t=6200adaf0a66a;x=d9b1ac66921eaac9",
	"__MONOTONIC_TIMESTAMP":  "11765948774",
	"__REALTIME_TIMESTAMP":   "1724080855230058",
	"__SEQNUM":               "5157297",
	"__SEQNUM_ID":            "e82795fad4ce42b79fb3da0866d91f7e",
}
