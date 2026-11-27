// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azuread

import (
	"reflect"
	"testing"
	"time"

	"github.com/elastic/beats/v7/pkg/config"
	"github.com/elastic/beats/v7/pkg/logp/logptest"
	"github.com/elastic/beats/v7/pkg/transport/httpcommon"
	"github.com/elastic/beats/v7/pkg/transport/tlscommon"
	ecentraid "github.com/elastic/entcollect/provider/entraid"
)

// TestMinimalConfigRoundTrip verifies that every exported field of
// ecentraid.Config is represented in the localConf mirror inside
// minimalConfig. It does this in two ways:
//
//  1. Field-count assertion — if ecentraid.Config gains a field, the
//     count check fails, forcing the developer to update localConf
//     and this test.
//  2. Functional round-trip — every field, and the HTTP and tracer
//     settings, are set to non-default sentinel values; the mapped
//     config must match them, which catches a field whose config tag
//     is wrong.
func TestMinimalConfigRoundTrip(t *testing.T) {
	const wantFields = 14
	if got := reflect.TypeFor[ecentraid.Config]().NumField(); got != wantFields {
		t.Fatalf("ecentraid.Config has %d exported fields, want %d; "+
			"update localConf inside minimalConfig and this test", got, wantFields)
	}

	wantSync := 48 * time.Hour
	wantUpdate := 30 * time.Minute
	scratchDir := "/scratch"

	// The HTTP and tracer settings use the keys the Fleet integration
	// sends.
	cfg := config.MustNewConfigFrom(map[string]any{
		"tenant_id":       "test-tenant",
		"client_id":       "test-client",
		"secret":          "test-secret",
		"login_endpoint":  "https://login.example.com",
		"login_scopes":    []string{"https://graph.example.com/.default"},
		"api_endpoint":    "https://graph.example.com/v1.0",
		"dataset":         "users",
		"enrich_with":     []string{"mfa"},
		"select.users":    []string{"displayName", "mail"},
		"select.groups":   []string{"displayName"},
		"select.devices":  []string{"displayName", "operatingSystem"},
		"sync_interval":   wantSync.String(),
		"update_interval": wantUpdate.String(),
		"scratch_dir":     scratchDir,

		"proxy_url":             "http://proxy.example.com:3128",
		"timeout":               "45s",
		"ssl.verification_mode": "none",
		"tracer.filename":       "../../logs/azure-ad/http-request-trace-*.ndjson",
		"tracer.maxbackups":     5,
		"tracer.maxsize":        1,
	})

	ms, err := minimalProvider(cfg, logptest.NewTestingLogger(t, ""))
	if err != nil {
		t.Fatalf("minimalProvider returned unexpected error: %v", err)
	}
	if ms.FullSync != wantSync {
		t.Errorf("sync_interval: got %v, want %v", ms.FullSync, wantSync)
	}
	if ms.IncrementalSync != wantUpdate {
		t.Errorf("update_interval: got %v, want %v", ms.IncrementalSync, wantUpdate)
	}
	if ms.Client == nil || ms.Client.Timeout != 45*time.Second {
		t.Errorf("client = %+v; want a client with a 45s timeout", ms.Client)
	}

	mc, err := minimalConfig(cfg)
	if err != nil {
		t.Fatalf("minimalConfig returned unexpected error: %v", err)
	}
	want := ecentraid.Config{
		TenantID:       "test-tenant",
		ClientID:       "test-client",
		ClientSecret:   "test-secret",
		LoginEndpoint:  "https://login.example.com",
		LoginScopes:    []string{"https://graph.example.com/.default"},
		APIEndpoint:    "https://graph.example.com/v1.0",
		Dataset:        "users",
		EnrichWith:     []string{"mfa"},
		SelectUsers:    []string{"displayName", "mail"},
		SelectGroups:   []string{"displayName"},
		SelectDevices:  []string{"displayName", "operatingSystem"},
		SyncInterval:   wantSync,
		UpdateInterval: wantUpdate,
		ScratchDir:     scratchDir,
	}
	if !reflect.DeepEqual(mc.provider, want) {
		t.Errorf("minimalConfig() = %+v; want %+v", mc.provider, want)
	}
	if got := mc.transport.Timeout; got != 45*time.Second {
		t.Errorf("timeout = %v; want 45s", got)
	}
	if got := mc.transport.Proxy.URL; got == nil || got.String() != "http://proxy.example.com:3128" {
		t.Errorf("proxy_url = %v; want http://proxy.example.com:3128", got)
	}
	if mc.transport.TLS == nil || mc.transport.TLS.VerificationMode != tlscommon.VerifyNone {
		t.Errorf("ssl = %+v; want verification_mode none", mc.transport.TLS)
	}
	if mc.tracer == nil || mc.tracer.Filename != "../../logs/azure-ad/http-request-trace-*.ndjson" || mc.tracer.MaxBackups != 5 {
		t.Errorf("tracer = %+v; want the configured filename and maxbackups 5", mc.tracer)
	}
	if mc.expand {
		t.Error("expand = true; want false when no expand options are set")
	}
}

func TestMinimalConfigDefaults(t *testing.T) {
	cfg := config.MustNewConfigFrom(map[string]any{
		"tenant_id": "test-tenant",
		"client_id": "test-client",
		"secret":    "test-secret",
	})
	mc, err := minimalConfig(cfg)
	if err != nil {
		t.Fatalf("minimalConfig returned unexpected error: %v", err)
	}
	d := ecentraid.DefaultConfig()
	if !reflect.DeepEqual(mc.provider.SelectUsers, d.SelectUsers) {
		t.Errorf("select.users = %q; want the default %q", mc.provider.SelectUsers, d.SelectUsers)
	}
	if !reflect.DeepEqual(mc.provider.SelectGroups, d.SelectGroups) {
		t.Errorf("select.groups = %q; want the default %q", mc.provider.SelectGroups, d.SelectGroups)
	}
	if !reflect.DeepEqual(mc.provider.SelectDevices, d.SelectDevices) {
		t.Errorf("select.devices = %q; want the default %q", mc.provider.SelectDevices, d.SelectDevices)
	}
	if got, want := mc.transport.Timeout, httpcommon.DefaultHTTPTransportSettings().Timeout; got != want {
		t.Errorf("timeout = %v; want the default %v", got, want)
	}
	if mc.tracer != nil {
		t.Errorf("tracer = %+v; want nil", mc.tracer)
	}
}

// TestMinimalConfigShortSelect checks that a configured select list
// replaces the default instead of being merged into it by index.
func TestMinimalConfigShortSelect(t *testing.T) {
	cfg := config.MustNewConfigFrom(map[string]any{
		"tenant_id":    "test-tenant",
		"client_id":    "test-client",
		"secret":       "test-secret",
		"select.users": []string{"onlyThis"},
	})
	mc, err := minimalConfig(cfg)
	if err != nil {
		t.Fatalf("minimalConfig returned unexpected error: %v", err)
	}
	if want := []string{"onlyThis"}; !reflect.DeepEqual(mc.provider.SelectUsers, want) {
		t.Errorf("select.users = %q; want %q", mc.provider.SelectUsers, want)
	}
}

// TestMinimalConfigExpand checks that the expand list the Fleet
// integration sends is accepted, recorded so a warning can be logged,
// and does not change the provider config.
func TestMinimalConfigExpand(t *testing.T) {
	cfg := config.MustNewConfigFrom(map[string]any{
		"tenant_id":    "test-tenant",
		"client_id":    "test-client",
		"secret":       "test-secret",
		"expand.users": []string{"directReports", "appRoleAssignments"},
	})
	mc, err := minimalConfig(cfg)
	if err != nil {
		t.Fatalf("minimalConfig returned unexpected error: %v", err)
	}
	if !mc.expand {
		t.Error("expand = false; want true when expand options are set")
	}

	log, logs := logptest.NewTestingLoggerWithObserver(t, "")
	if _, err := minimalProvider(cfg, log); err != nil {
		t.Fatalf("minimalProvider returned unexpected error: %v", err)
	}
	if n := logs.FilterMessageSnippet("expand options are not supported").Len(); n != 1 {
		t.Errorf("got %d expand warnings; want 1", n)
	}
}
