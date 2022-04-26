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

package auditd

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/prometheus/procfs"

	"github.com/elastic/beats/v7/auditbeat/core"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/mapping"
	"github.com/elastic/beats/v7/metricbeat/mb"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/go-libaudit/v2"
	"github.com/elastic/go-libaudit/v2/auparse"
)

// Specify the -audit flag when running these tests to interact with the real
// kernel instead of mocks. If running in Docker this requires being in the
// host PID namespace (--pid=host) and having CAP_AUDIT_CONTROL and
// CAP_AUDIT_WRITE (so use --privileged).
var audit = flag.Bool("audit", false, "interact with the real audit framework")

var (
	userLoginFailMsg    = `type=USER_LOGIN msg=audit(1492896301.818:19955): pid=12635 uid=0 auid=4294967295 ses=4294967295 msg='op=login acct=28696E76616C6964207573657229 exe="/usr/sbin/sshd" hostname=? addr=179.38.151.221 terminal=sshd res=failed'`
	userLoginSuccessMsg = `type=USER_LOGIN msg=audit(1492896303.915:19956): pid=12635 uid=0 auid=4294967295 ses=4294967295 msg='op=login acct=61647269616E exe="/usr/sbin/sshd" hostname=? addr=179.38.151.221 terminal=sshd res=success'`
	userAuthMsg         = `type=USER_AUTH msg=audit(1552714590.571:21114): pid=11312 uid=0 auid=0 ses=62 msg='op=PAM:authentication acct="root" exe="/bin/su" hostname="test" addr="127.0.0.1" terminal=/dev/pts/0 res=success'`

	execveMsgs = []string{
		`type=SYSCALL msg=audit(1492752522.985:8972): arch=c000003e syscall=59 success=yes exit=0 a0=10812c8 a1=1070208 a2=1152008 a3=59a items=2 ppid=10027 pid=10043 auid=1001 uid=1001 gid=1002 euid=1001 suid=1001 fsuid=1001 egid=1002 sgid=1002 fsgid=1002 tty=pts0 ses=11 comm="uname" exe="/bin/uname" key="key=user_commands"`,
		`type=EXECVE msg=audit(1492752522.985:8972): argc=2 a0="uname" a1="-a"`,
		`type=CWD msg=audit(1492752522.985:8972): cwd="/home/andrew_kroh"`,
		`type=PATH msg=audit(1492752522.985:8972): item=0 name="/bin/uname" inode=155 dev=08:01 mode=0100755 ouid=0 ogid=0 rdev=00:00 nametype=NORMAL`,
		`type=PATH msg=audit(1492752522.985:8972): item=1 name="/lib64/ld-linux-x86-64.so.2" inode=1923 dev=08:01 mode=0100755 ouid=0 ogid=0 rdev=00:00 nametype=NORMAL`,
		`type=PROCTITLE msg=audit(1492752522.985:8972): proctitle=756E616D65002D61`,
		`type=EOE msg=audit(1492752522.985:8972):`,
	}

	acceptMsgs = []string{
		`type=SYSCALL msg=audit(1492752520.441:8832): arch=c000003e syscall=43 success=yes exit=5 a0=3 a1=7ffd0dc80040 a2=7ffd0dc7ffd0 a3=0 items=0 ppid=1 pid=1663 auid=4294967295 uid=0 gid=0 euid=0 suid=0 fsuid=0 egid=0 sgid=0 fsgid=0 tty=(none) ses=4294967295 comm="sshd" exe="/usr/sbin/sshd" key="key=net"`,
		`type=SOCKADDR msg=audit(1492752520.441:8832): saddr=0200E31C4853E6640000000000000000`,
		`type=PROCTITLE msg=audit(1492752520.441:8832): proctitle="(sshd)"`,
		`type=EOE msg=audit(1492752520.441:8832):`,
	}
)

func TestData(t *testing.T) {
	logp.TestingSetup()

	// Create a mock netlink client that provides the expected responses.
	mock := NewMock().
		// Get Status response for initClient
		returnACK().returnStatus().
		// Send expected ACKs for initialization
		returnACK().returnACK().returnACK().returnACK().returnACK().
		// Send three auditd messages.
		returnMessage(userLoginFailMsg).
		returnMessage(execveMsgs...).
		returnMessage(acceptMsgs...)

	// Replace the default AuditClient with a mock.
	ms := mbtest.NewPushMetricSetV2(t, getConfig())
	auditMetricSet := ms.(*MetricSet)
	auditMetricSet.client.Close()
	auditMetricSet.client = &libaudit.AuditClient{Netlink: mock}

	events := mbtest.RunPushMetricSetV2(10*time.Second, 3, ms)
	if len(events) != 3 {
		t.Fatalf("expected 3 events, but received %d", len(events))
	}
	assertNoErrors(t, events)

	assertFieldsAreDocumented(t, events)

	beatEvent := mbtest.StandardizeEvent(ms, events[0], core.AddDatasetToEvent)
	mbtest.WriteEventToDataJSON(t, beatEvent, "")
}

func TestLoginType(t *testing.T) {
	logp.TestingSetup()

	// Create a mock netlink client that provides the expected responses.
	mock := NewMock().
		// Get Status response for initClient
		returnACK().returnStatus().
		// Send expected ACKs for initialization
		returnACK().returnACK().returnACK().returnACK().returnACK().
		// Send an authentication failure and a success.
		returnMessage(userLoginFailMsg).
		returnMessage(userLoginSuccessMsg).
		returnMessage(userAuthMsg)

	// Replace the default AuditClient with a mock.
	ms := mbtest.NewPushMetricSetV2(t, getConfig())
	auditMetricSet := ms.(*MetricSet)
	auditMetricSet.client.Close()
	auditMetricSet.client = &libaudit.AuditClient{Netlink: mock}

	const expectedEvents = 3
	events := mbtest.RunPushMetricSetV2(10*time.Second, expectedEvents, ms)
	if len(events) != expectedEvents {
		t.Fatalf("expected %d events, but received %d", expectedEvents, len(events))
	}
	assertNoErrors(t, events)

	assertFieldsAreDocumented(t, events)

	sort.Slice(events,
		func(i, j int) bool {
			return events[i].ModuleFields["sequence"].(uint32) < events[j].ModuleFields["sequence"].(uint32)
		})

	for idx, expected := range []mapstr.M{
		{
			"event.category":      []string{"authentication"},
			"event.type":          []string{"start"},
			"event.outcome":       "failure",
			"user.effective.name": "(invalid user)",
			"user.id":             nil,
			"session":             nil,
		},
		{
			"event.category":      []string{"authentication"},
			"event.type":          []string{"start"},
			"event.outcome":       "success",
			"user.effective.name": "adrian",
			"user.audit.id":       nil,
			"auditd.session":      nil,
		},
		{
			"event.category": []string{"authentication"},
			"event.type":     []string{"info"},
			"event.outcome":  "success",
			"user.name":      "root",
			"user.id":        "0",
			"user.audit.id":  "0",
			"auditd.session": "62",
		},
	} {
		beatEvent := mbtest.StandardizeEvent(ms, events[idx], core.AddDatasetToEvent)
		mbtest.WriteEventToDataJSON(t, beatEvent, "")
		for k, v := range expected {
			msg := fmt.Sprintf("%s[%d]", k, idx)
			cur, err := beatEvent.GetValue(k)
			if v != nil {
				assert.NoError(t, err, msg)
				assert.Equal(t, v, cur, msg)
			} else {
				_, err := beatEvent.GetValue(k)
				assert.Equal(t, common.ErrKeyNotFound, err, msg)
			}
		}
	}
}

// assertFieldsAreDocumented mimics assert_fields_are_documented in Python system tests.
func assertFieldsAreDocumented(t *testing.T, events []mb.Event) {
	fieldsYml, err := mapping.LoadFieldsYaml("../../fields.yml")
	if err != nil {
		t.Fatal(err)
	}
	documentedFields := fieldsYml.GetKeys()

	for _, e := range events {
		beatEvent := e.BeatEvent(moduleName, metricsetName, core.AddDatasetToEvent)
		for eventFieldName := range beatEvent.Fields.Flatten() {
			found := false
			for _, documentedFieldName := range documentedFields {
				// Have to use HasPrefix and not "==" since fields in auditd.paths.* get flattened
				// to auditd.paths which does not exist in fields.yml.
				if strings.HasPrefix(documentedFieldName, eventFieldName) {
					found = true
					break
				}
			}
			if !found {
				assert.Fail(t, "Field not documented", "Key '%v' found in event is not documented.", eventFieldName)
			}
		}
	}
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":              "auditd",
		"failure_mode":        "log",
		"socket_type":         "unicast",
		"include_warnings":    true,
		"include_raw_message": true,
	}
}

func TestUnicastClient(t *testing.T) {
	if !*audit {
		t.Skip("-audit was not specified")
	}

	logp.TestingSetup()
	FailIfAuditdIsRunning(t)

	c := map[string]interface{}{
		"module":      "auditd",
		"socket_type": "unicast",
		"audit_rules": fmt.Sprintf(`
		   -a always,exit -F arch=b64 -F ppid=%d -S execve -k exec
		`, os.Getpid()),
	}

	// Any commands executed by this process will generate events due to the
	// PPID filter we applied to the rule.
	time.AfterFunc(time.Second, func() { exec.Command("cat", "/proc/self/status").Output() })

	ms := mbtest.NewPushMetricSetV2(t, c)
	events := mbtest.RunPushMetricSetV2(5*time.Second, 0, ms)
	assertNoErrors(t, events)
	assertHasBinCatExecve(t, events)
}

func TestMulticastClient(t *testing.T) {
	if !*audit {
		t.Skip("-audit was not specified")
	}

	if !hasMulticastSupport() {
		t.Skip("no multicast support")
	}

	logp.TestingSetup()
	FailIfAuditdIsRunning(t)

	c := map[string]interface{}{
		"module":      "auditd",
		"socket_type": "multicast",
		"audit_rules": fmt.Sprintf(`
		   -a always,exit -F arch=b64 -F ppid=%d -S execve -k exec
		`, os.Getpid()),
	}

	// Any commands executed by this process will generate events due to the
	// PPID filter we applied to the rule.
	time.AfterFunc(time.Second, func() { exec.Command("cat", "/proc/self/status").Output() })

	ms := mbtest.NewPushMetricSetV2(t, c)
	events := mbtest.RunPushMetricSetV2(5*time.Second, 0, ms)
	assertNoErrors(t, events)
	assertHasBinCatExecve(t, events)
}

func TestKernelVersion(t *testing.T) {
	major, minor, full, err := kernelVersion()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("major=%v, minor=%v, full=%v", major, minor, full)
}

func FailIfAuditdIsRunning(t testing.TB) {
	t.Helper()

	procs, err := procfs.AllProcs()
	if err != nil {
		t.Fatal(err)
	}

	for _, proc := range procs {
		comm, err := proc.Comm()
		if err != nil {
			t.Error(err)
			continue
		}

		if comm == "auditd" {
			t.Fatalf("auditd is running (pid=%d). This test cannot run while "+
				"auditd is running.", proc.PID)
		}
	}
}

func TestBuildMetricbeatEvent(t *testing.T) {
	if f := flag.Lookup("data"); f != nil && f.Value.String() == "false" {
		t.Skip("skip data generation tests")
	}
	buildSampleEvent(t, acceptMsgs, "_meta/accept.json")
	buildSampleEvent(t, execveMsgs, "_meta/execve.json")
}

func buildSampleEvent(t testing.TB, lines []string, filename string) {
	var msgs []*auparse.AuditMessage
	for _, txt := range lines {
		m, err := auparse.ParseLogLine(txt)
		if err != nil {
			t.Fatal(err)
		}
		msgs = append(msgs, m)
	}

	e := buildMetricbeatEvent(msgs, defaultConfig)
	beatEvent := e.BeatEvent(moduleName, metricsetName, core.AddDatasetToEvent)
	output, err := json.MarshalIndent(&beatEvent.Fields, "", "    ")
	if err != nil {
		t.Fatal(err)
	}

	if err := ioutil.WriteFile(filename, output, 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertHasBinCatExecve(t *testing.T, events []mb.Event) {
	t.Helper()

	for _, e := range events {
		v, err := e.RootFields.GetValue("process.executable")
		if err == nil {
			if exe, ok := v.(string); ok && exe == "/bin/cat" {
				return
			}
		}
	}
	assert.Fail(t, "expected an execve event for /bin/cat")
}

func assertNoErrors(t *testing.T, events []mb.Event) {
	t.Helper()

	for _, e := range events {
		t.Log(e)

		if !assert.Nil(t, e.Error) {
			t.Errorf("received error: %+v", e.Error)
		}
		errorMsgKey, err := e.RootFields.GetValue("error.message")
		if err == nil && !assert.Nil(t, errorMsgKey) {
			t.Errorf("event has error messages: %v", errorMsgKey)
		}
	}
}
