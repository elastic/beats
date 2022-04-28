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

//go:build linux
// +build linux

package auditd

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/go-libaudit/v2"
	"github.com/elastic/go-libaudit/v2/aucoalesce"

	"github.com/elastic/beats/v7/metricbeat/mb"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
)

const (
	testDir       = "testdata"
	testExt       = ".log"
	testPattern   = "*" + testExt
	goldenSuffix  = "-expected.json"
	goldenPattern = testPattern + goldenSuffix
	fileTimeout   = 3 * time.Minute
	terminator    = "type=TEST msg=audit(0.0:585): msg=\"terminate\""
)

var (
	update = flag.Bool("update", false, "update golden data")

	knownUsers = []user.User{
		{Username: "vagrant", Uid: "1000"},
		{Username: "alice", Uid: "1001"},
		{Username: "oldbob", Uid: "1002"},
		{Username: "charlie", Uid: "1003"},
		{Username: "testuser", Uid: "1004"},
		{Username: "bob", Uid: "9999"},
	}

	knownGroups = []user.Group{
		{Name: "vagrant", Gid: "1000"},
		{Name: "alice", Gid: "1001"},
		{Name: "oldbob", Gid: "1002"},
		{Name: "charlie", Gid: "1003"},
		{Name: "testgroup", Gid: "1004"},
		{Name: "bob", Gid: "9999"},
	}
)

func readLines(path string) (lines []string, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func readGoldenFile(t testing.TB, path string) (events []mapstr.M) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatalf("can't read golden file '%s': %v", path, err)
	}
	if err = json.Unmarshal(data, &events); err != nil {
		t.Fatalf("error decoding JSON from golden file '%s': %v", path, err)
	}
	return
}

func normalize(t testing.TB, events []mb.Event) (norm []mapstr.M) {
	for _, ev := range events {
		var output mapstr.M
		data, err := json.Marshal(ev.BeatEvent(moduleName, metricsetName).Fields)
		if err != nil {
			t.Fatal(err)
		}
		json.Unmarshal(data, &output)
		norm = append(norm, output)
	}
	return norm
}

func configForGolden() map[string]interface{} {
	return map[string]interface{}{
		"module":                  "auditd",
		"failure_mode":            "log",
		"socket_type":             "unicast",
		"include_warnings":        true,
		"include_raw_message":     true,
		"resolve_ids":             true,
		"stream_buffer_consumers": 1,
	}
}

type (
	TerminateFn        func(mb.Event) bool
	terminableReporter struct {
		events []mb.Event
		ctx    context.Context
		cancel context.CancelFunc
		err    error
		isLast TerminateFn
	}
)

func (r *terminableReporter) Event(event mb.Event) bool {
	if r.ctx.Err() != nil {
		return false
	}
	if r.isLast(event) {
		r.cancel()
		return false
	}
	r.events = append(r.events, event)
	return true
}

func (r *terminableReporter) Error(err error) bool {
	if r.ctx.Err() != nil && r.err != nil {
		r.err = err
		r.cancel()
	}
	return true
}

func (r *terminableReporter) Done() <-chan struct{} {
	return r.ctx.Done()
}

func runTerminableReporter(timeout time.Duration, ms mb.PushMetricSetV2, isLast TerminateFn) []mb.Event {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	reporter := terminableReporter{
		ctx:    ctx,
		cancel: cancel,
		isLast: isLast,
	}
	go ms.Run(&reporter)
	<-ctx.Done()
	return reporter.events
}

func isTestEvent(event mb.Event) bool {
	mt, ok := event.ModuleFields["message_type"]
	return ok && mt == "test"
}

func TestGoldenFiles(t *testing.T) {
	// Add testing users and groups to test with resolve_ids enabled.
	aucoalesce.HardcodeUsers(knownUsers...)
	aucoalesce.HardcodeGroups(knownGroups...)

	sourceFiles, err := filepath.Glob(filepath.Join(testDir, testPattern))
	if err != nil {
		t.Fatal(err)
	}

	for _, file := range sourceFiles {
		testName := strings.TrimSuffix(filepath.Base(file), testExt)
		t.Run(testName, func(t *testing.T) {
			lines, err := readLines(file)
			if err != nil {
				t.Fatalf("error reading log file '%s': %v", file, err)
			}
			mock := NewMock().
				// Get Status response for initClient
				returnACK().returnStatus().
				// Send expected ACKs for initialization
				returnACK().returnACK().returnACK().returnACK().returnACK().
				// Send audit messages
				returnMessage(lines...).
				// Send stream terminator
				returnMessage(terminator)

			ms := mbtest.NewPushMetricSetV2(t, configForGolden())
			auditMetricSet := ms.(*MetricSet)
			auditMetricSet.client.Close()
			auditMetricSet.client = &libaudit.AuditClient{Netlink: mock}
			mbEvents := runTerminableReporter(fileTimeout, ms, isTestEvent)
			t.Logf("Received %d events for %d audit records", len(mbEvents), len(lines))
			assertNoErrors(t, mbEvents)
			events := normalize(t, mbEvents)
			goldenPath := file + goldenSuffix
			if *update {
				data, err := json.MarshalIndent(events, "", "  ")
				if err != nil {
					t.Fatal(err)
				}
				if err = ioutil.WriteFile(goldenPath, data, 0o644); err != nil {
					t.Fatalf("failed writing golden file '%s': %v", goldenPath, err)
				}
			}
			golden := readGoldenFile(t, goldenPath)
			assert.EqualValues(t, golden, events)
		})
	}
}
