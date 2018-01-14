package auditd

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/auditbeat/core"
	"github.com/elastic/beats/libbeat/logp"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/go-libaudit"
	"github.com/elastic/procfs"
)

// Specify the -audit flag when running these tests to interact with the real
// kernel instead of mocks. If running in Docker this requires being in the
// host PID namespace (--pid=host) and having CAP_AUDIT_CONTROL and
// CAP_AUDIT_WRITE (so use --privileged).
var audit = flag.Bool("audit", false, "interact with the real audit framework")

var userLoginMsg = `type=USER_LOGIN msg=audit(1492896301.818:19955): pid=12635 uid=0 auid=4294967295 ses=4294967295 msg='op=login acct=28696E76616C6964207573657229 exe="/usr/sbin/sshd" hostname=? addr=179.38.151.221 terminal=sshd res=failed'`

func TestData(t *testing.T) {
	logp.TestingSetup()

	// Create a mock netlink client that provides the expected responses.
	mock := NewMock().
		// Get Status response for initClient
		returnACK().returnStatus().
		// Send a single audit message from the kernel.
		returnMessage(userLoginMsg)

	// Replace the default AuditClient with a mock.
	ms := mbtest.NewPushMetricSetV2(t, getConfig())
	auditMetricSet := ms.(*MetricSet)
	auditMetricSet.client.Close()
	auditMetricSet.client = &libaudit.AuditClient{Netlink: mock}

	events := mbtest.RunPushMetricSetV2(10*time.Second, 1, ms)
	for _, e := range events {
		if e.Error != nil {
			t.Fatalf("received error: %+v", e.Error)
		}
	}
	if len(events) == 0 {
		t.Fatal("received no events")
	}

	beatEvent := mbtest.StandardizeEvent(ms, events[0], core.AddDatasetToEvent)
	mbtest.WriteEventToDataJSON(t, beatEvent)
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":       "auditd",
		"failure_mode": "log",
		"socket_type":  "unicast",
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
	for _, e := range events {
		t.Log(e)

		if e.Error != nil {
			t.Errorf("received error: %+v", e.Error)
		}
	}

	for _, e := range events {
		v, err := e.MetricSetFields.GetValue("thing.primary")
		if err == nil {
			if exe, ok := v.(string); ok && exe == "/bin/cat" {
				return
			}
		}
	}
	assert.Fail(t, "expected an execve event for /bin/cat")
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
	for _, e := range events {
		if e.Error != nil {
			t.Fatalf("received error: %+v", e.Error)
		}
	}

	// The number of events is non-deterministic so there is no validation.
	t.Logf("received %d messages via multicast", len(events))
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
