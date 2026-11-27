// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package entityanalytics

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/pkg/config"
	"github.com/elastic/beats/v7/pkg/logp/logptest"
	"github.com/elastic/beats/v7/pkg/monitoring"
	"github.com/elastic/beats/v7/pkg/paths"
)

// TestLegacyInput_RunFullSyncRetry checks that every legacy provider retries
// a failed full sync well before the full sync interval, and stays degraded
// by the full sync in the meantime.
func TestLegacyInput_RunFullSyncRetry(t *testing.T) {
	// Provider requests fail quickly, since the server closes every
	// connection.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	t.Cleanup(func() { l.Close() })
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()
	addr := l.Addr().String()

	tests := map[string]string{
		"activedirectory": `
ad_url: ldap://%s
ad_base_dn: DC=example,DC=com
ad_user: user
ad_password: pass
`,
		"azure-ad": `
tenant_id: tenant
client_id: client
secret: secret
login_endpoint: http://%[1]s
api_endpoint: http://%[1]s
`,
		"jamf": `
jamf_tenant: %s
jamf_username: user
jamf_password: pass
request.retry.max_attempts: 1
request.retry.wait_min: 1ms
request.retry.wait_max: 1ms
`,
		"okta": `
okta_domain: %s
okta_token: token
request.retry.max_attempts: 1
request.retry.wait_min: 1ms
request.retry.wait_max: 1ms
`,
	}
	for name, cfg := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			log := logptest.NewTestingLogger(t, "test")
			path := &paths.Path{Data: t.TempDir(), Logs: t.TempDir()}
			in, err := (&manager{logger: log, path: path}).Create(config.MustNewConfigFrom(
				"id: test-" + name + "\nprovider: " + name + "\nsync_interval: 1h\nupdate_interval: 10ms\n" + fmt.Sprintf(cfg, addr),
			))
			if err != nil {
				t.Fatalf("failed to create input: %v", err)
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			reg := monitoring.NewRegistry()
			fullSyncs := func() uint64 {
				n, _ := reg.Get("sync_total").(*monitoring.Uint)
				if n == nil {
					return 0
				}
				return n.Get()
			}
			reporter := &statusRecorder{seen: make(chan struct{}), done: func([]statusUpdate) bool {
				return fullSyncs() >= 2
			}}
			done := make(chan error, 1)
			go func() {
				done <- in.Run(
					v2.Context{
						Logger:          log,
						ID:              "test-" + name,
						IDWithoutName:   "test-" + name,
						Cancelation:     ctx,
						MetricsRegistry: reg,
					}.WithStatusReporter(reporter),
					&fakeConnector{client: &ackingClient{inner: &fakeClient{}}},
				)
			}()
			select {
			case <-reporter.seen:
			case err := <-done:
				t.Fatalf("Run returned %v early; updates: %v", err, reporter.updates())
			case <-time.After(30 * time.Second):
				t.Errorf("timed out; updates: %v", reporter.updates())
			}
			// Stop Run before the test ends, so it does not log after that.
			cancel()
			if err := <-done; err != nil {
				t.Errorf("Run = %v; want nil", err)
			}

			if got := fullSyncs(); got < 2 {
				t.Errorf("sync_total = %d; want the failed full sync to be retried", got)
			}
			failed := false
			for _, u := range reporter.updates() {
				full := u.status == status.Degraded && strings.HasPrefix(u.msg, "Error running full sync: ")
				failed = failed || full
				if failed && !full && u.status != status.Stopping {
					t.Errorf("status %v %q reported after a full sync failed; want degraded by the full sync", u.status, u.msg)
				}
			}
			if !failed {
				t.Errorf("no degraded status for the failed full sync in %v", reporter.updates())
			}
		})
	}
}
