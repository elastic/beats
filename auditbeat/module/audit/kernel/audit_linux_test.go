package kernel

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/go-libaudit"
)

// Specify the -audit flag when running these tests to interact with the real
// kernel instead of mocks. If running in Docker this requires being in the
// host PID namespace (--pid=host) and having CAP_AUDIT_CONTROL and
// CAP_AUDIT_WRITE (so use --privileged).
var audit = flag.Bool("audit", false, "interact with the real audit framework")

var userLoginMsg = `type=USER_LOGIN msg=audit(1492896301.818:19955): pid=12635 uid=0 auid=4294967295 ses=4294967295 msg='op=login acct=28696E76616C6964207573657229 exe="/usr/sbin/sshd" hostname=? addr=179.38.151.221 terminal=sshd res=failed'`

func TestData(t *testing.T) {
	// Create a mock netlink client that provides the expected responses.
	mock := NewMock().
		// Get Status response for initClient
		returnACK().returnStatus().
		// Send a single audit message from the kernel.
		returnMessage(userLoginMsg)

	// Replace the default AuditClient with a mock.
	ms := mbtest.NewPushMetricSet(t, getConfig())
	auditMetricSet := ms.(*MetricSet)
	auditMetricSet.client.Close()
	auditMetricSet.client = &libaudit.AuditClient{Netlink: mock}

	events, errs := mbtest.RunPushMetricSet(time.Second, ms)
	if len(errs) > 0 {
		t.Fatalf("received errors: %+v", errs)
	}
	if len(events) == 0 {
		t.Fatal("received no events")
	}

	fullEvent := mbtest.CreateFullEvent(ms, events[0])
	mbtest.WriteEventToDataJSON(t, fullEvent)
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":              "audit",
		"metricsets":          []string{"kernel"},
		"kernel.failure_mode": "log",
		"kernel.socket_type":  "unicast",
	}
}

func TestMulticastClient(t *testing.T) {
	if !*audit {
		t.Skip("-audit was not specified")
	}

	if !hasMulticastSupport() {
		t.Skip("no multicast support")
	}

	c := map[string]interface{}{
		"module":             "audit",
		"metricsets":         []string{"kernel"},
		"kernel.socket_type": "multicast",
		"kernel.audit_rules": fmt.Sprintf(`
		   -a always,exit -F arch=b64 -F ppid=%d -S execve -k exec
		`, os.Getpid()),
	}

	// Any commands executed by this process will generate events due to the
	// PPID filter we applied to the rule.
	time.AfterFunc(time.Second, func() { exec.Command("cat", "/proc/self/status").Output() })

	ms := mbtest.NewPushMetricSet(t, c)
	events, errs := mbtest.RunPushMetricSet(5*time.Second, ms)
	if len(errs) > 0 {
		t.Fatalf("received errors: %+v", errs)
	}

	// The number of events is non-deterministic so there is no validation.
	t.Logf("received %d messages via multicast", len(events))
}

func TestUnicastClient(t *testing.T) {
	if !*audit {
		t.Skip("-audit was not specified")
	}

	c := map[string]interface{}{
		"module":             "audit",
		"metricsets":         []string{"kernel"},
		"kernel.socket_type": "unicast",
		"kernel.audit_rules": fmt.Sprintf(`
		   -a always,exit -F arch=b64 -F ppid=%d -S execve -k exec
		`, os.Getpid()),
	}

	// Any commands executed by this process will generate events due to the
	// PPID filter we applied to the rule.
	time.AfterFunc(time.Second, func() { exec.Command("cat", "/proc/self/status").Output() })

	ms := mbtest.NewPushMetricSet(t, c)
	events, errs := mbtest.RunPushMetricSet(5*time.Second, ms)
	if len(errs) > 0 {
		t.Fatalf("received errors: %+v", errs)
	}

	t.Log(events)
	assert.Len(t, events, 1)
}

func TestKernelVersion(t *testing.T) {
	major, minor, full, err := kernelVersion()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("major=%v, minor=%v, full=%v", major, minor, full)
}
