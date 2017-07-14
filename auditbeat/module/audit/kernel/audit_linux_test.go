package kernel

import (
	"testing"
	"time"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/go-libaudit"
)

var userLoginMsg = `type=USER_LOGIN msg=audit(1492896301.818:19955): pid=12635 uid=0 auid=4294967295 ses=4294967295 msg='op=login acct=28696E76616C6964207573657229 exe="/usr/sbin/sshd" hostname=? addr=179.38.151.221 terminal=sshd res=failed'`

func TestData(t *testing.T) {
	// Create a mock netlink client that provides the expected responses.
	mock := NewMock().
		// GetRules response with zero rules. Used by DeleteAll rules.
		returnACK().returnDone().
		// AddRule response.
		returnACK().
		// AddRule response.
		returnACK().
		// Get Status response for initClient
		returnACK().returnStatus().
		// Send a single audit message from the kernel.
		returnMessage(userLoginMsg)

	// Replace the default AuditClient with a mock.
	ms := mbtest.NewPushMetricSet(t, getConfig())
	auditMetricSet := ms.(*MetricSet)
	auditMetricSet.client.Close()
	auditMetricSet.client = &libaudit.AuditClient{mock}

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
		"kernel.audit_rules": `
		   -w /etc/passwd -p wa -k auth
		   -a always,exit -F arch=b64 -S execve -k exec
		`,
	}
}
