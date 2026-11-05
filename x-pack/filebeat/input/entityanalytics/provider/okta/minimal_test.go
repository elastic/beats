// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package okta

import (
	"reflect"
	"testing"
	"time"

	"github.com/elastic/beats/v7/pkg/config"
	"github.com/elastic/beats/v7/pkg/logp/logptest"
	ecokta "github.com/elastic/entcollect/provider/okta"
)

// TestMinimalConfigRoundTrip verifies that every exported field of
// ecokta.Config is represented in the localConf mirror inside
// minimalConfig. It does this in two ways:
//
//  1. Field-count assertion — if ecokta.Config gains a field, the count
//     check fails, forcing the developer to update localConf and this test.
//  2. Functional round-trip — the top-level fields are set to non-default
//     sentinel values; the returned ecokta.Config must match them, which
//     catches a field whose config tag is wrong. The oauth2 settings are
//     not covered here; the request and tracer settings are covered by
//     TestMinimalConfigRequest.
func TestMinimalConfigRoundTrip(t *testing.T) {
	const wantFields = 12
	if got := reflect.TypeFor[ecokta.Config]().NumField(); got != wantFields {
		t.Fatalf("ecokta.Config has %d exported fields, want %d; "+
			"update localConf inside minimalConfig and this test", got, wantFields)
	}

	wantSync := 48 * time.Hour
	wantUpdate := 30 * time.Minute
	limitFixed := 50
	scratchDir := "/scratch"

	cfg := config.MustNewConfigFrom(map[string]any{
		"okta_domain":     "test.okta.com",
		"okta_token":      "test-token",
		"dataset":         "users",
		"enrich_with":     []string{"groups", "factors"},
		"batch_size":      100,
		"sync_interval":   wantSync.String(),
		"update_interval": wantUpdate.String(),
		"idset_shards":    32,
		"limit_window":    "2m",
		"limit_fixed":     limitFixed,
		"scratch_dir":     scratchDir,
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

	// The provider hides its config, so check the mapping directly. A
	// field with a wrong struct tag is silently dropped by ucfg and would
	// otherwise only show up as a default value at run time.
	mc, err := minimalConfig(cfg)
	if err != nil {
		t.Fatalf("minimalConfig returned unexpected error: %v", err)
	}
	got := mc.provider
	want := ecokta.Config{
		Domain:         "test.okta.com",
		Token:          "test-token",
		Dataset:        "users",
		EnrichWith:     []string{"groups", "factors"},
		BatchSize:      100,
		SyncInterval:   wantSync,
		UpdateInterval: wantUpdate,
		IDSetShards:    32,
		LimitWindow:    2 * time.Minute,
		ScratchDir:     scratchDir,
	}
	// Compare the pointer field by value so a mismatch is readable, then
	// clear it for the whole-struct comparison.
	switch {
	case got.LimitFixed == nil:
		t.Errorf("limit_fixed: got nil, want %d", limitFixed)
	case *got.LimitFixed != limitFixed:
		t.Errorf("limit_fixed: got %d, want %d", *got.LimitFixed, limitFixed)
	}
	got.LimitFixed = nil
	if !reflect.DeepEqual(got, want) {
		t.Errorf("minimalConfig() = %+v; want %+v", got, want)
	}
}

func TestMinimalProvider_DualAuth(t *testing.T) {
	cfg := config.MustNewConfigFrom(map[string]any{
		"okta_domain":          "test.okta.com",
		"okta_token":           "legacy-token", //nolint:gosec // This is not a secret.
		"oauth2.enabled":       true,
		"oauth2.client.id":     "client-id",
		"oauth2.client.secret": "client-secret",
		"oauth2.token_url":     "https://test.okta.com/oauth2/v1/token",
		"oauth2.scopes":        []string{"okta.users.read"},
		"sync_interval":        "24h",
		"update_interval":      "15m",
	})

	_, err := minimalProvider(cfg, logptest.NewTestingLogger(t, ""))
	if err != nil {
		t.Fatalf("minimalProvider rejected dual-auth config: %v — "+
			"OAuth2 should take precedence over okta_token", err)
	}
}

func TestMinimalConfigRequest(t *testing.T) {
	cfg := config.MustNewConfigFrom(map[string]any{
		"okta_domain":                             "test.okta.com",
		"okta_token":                              "test-token",
		"request.timeout":                         "45s",
		"request.proxy_url":                       "http://proxy.example.com:3128",
		"request.keep_alive.disable":              true,
		"request.keep_alive.max_idle_connections": 5,
		"request.redirect.forward_headers":        true,
		"request.redirect.headers_ban_list":       []string{"X-Secret"},
		"request.redirect.max_redirects":          3,
		"request.retry.max_attempts":              2,
		"tracer.enabled":                          true,
		"tracer.filename":                         "../../logs/okta/http-request-trace-*.ndjson",
		"tracer.maxbackups":                       5,
	})
	mc, err := minimalConfig(cfg)
	if err != nil {
		t.Fatalf("minimalConfig returned unexpected error: %v", err)
	}
	req := mc.request
	timeout := 45 * time.Second
	if req.Transport.Timeout != timeout {
		t.Errorf("request.timeout = %v; want %v", req.Transport.Timeout, timeout)
	}
	if req.Transport.Proxy.URL == nil || req.Transport.Proxy.URL.String() != "http://proxy.example.com:3128" {
		t.Errorf("request.proxy_url = %v; want http://proxy.example.com:3128", req.Transport.Proxy.URL)
	}
	if req.KeepAlive.Disable == nil || !*req.KeepAlive.Disable {
		t.Errorf("request.keep_alive.disable = %v; want true", req.KeepAlive.Disable)
	}
	if req.KeepAlive.MaxIdleConns != 5 {
		t.Errorf("request.keep_alive.max_idle_connections = %d; want 5", req.KeepAlive.MaxIdleConns)
	}
	if !req.RedirectForwardHeaders {
		t.Error("request.redirect.forward_headers = false; want true")
	}
	if !reflect.DeepEqual(req.RedirectHeadersBanList, []string{"X-Secret"}) {
		t.Errorf("request.redirect.headers_ban_list = %q; want [X-Secret]", req.RedirectHeadersBanList)
	}
	if req.RedirectMaxRedirects != 3 {
		t.Errorf("request.redirect.max_redirects = %d; want 3", req.RedirectMaxRedirects)
	}
	if req.Retry == nil {
		t.Error("request.retry = nil; want it recorded so a warning can be logged")
	}
	if mc.tracer == nil {
		t.Fatal("tracer = nil; want it set")
	}
	if mc.tracer.Enabled == nil || !*mc.tracer.Enabled {
		t.Errorf("mc.tracer.enabled = %v; want true", mc.tracer.Enabled)
	}
	if mc.tracer.Filename != "../../logs/okta/http-request-trace-*.ndjson" {
		t.Errorf("mc.tracer.filename = %q; want ../../logs/okta/http-request-trace-*.ndjson", mc.tracer.Filename)
	}
	if mc.tracer.MaxBackups != 5 {
		t.Errorf("mc.tracer.maxbackups = %d; want 5", mc.tracer.MaxBackups)
	}
	if mc.tracer.MaxSize != 1 {
		t.Errorf("mc.tracer.maxsize = %d; want the 1MB default", mc.tracer.MaxSize)
	}
	if _, err := minimalProvider(cfg, logptest.NewTestingLogger(t, "")); err != nil {
		t.Errorf("minimalProvider returned unexpected error: %v", err)
	}
}

func TestMinimalConfigRequestDefaults(t *testing.T) {
	cfg := config.MustNewConfigFrom(map[string]any{
		"okta_domain": "test.okta.com",
		"okta_token":  "test-token",
	})
	mc, err := minimalConfig(cfg)
	if err != nil {
		t.Fatalf("minimalConfig returned unexpected error: %v", err)
	}
	if got, want := mc.request.Transport.Timeout, 30*time.Second; got != want {
		t.Errorf("request.timeout = %v; want %v", got, want)
	}
	if got, want := mc.request.RedirectMaxRedirects, 10; got != want {
		t.Errorf("request.redirect.max_redirects = %d; want %d", got, want)
	}
	if mc.tracer != nil {
		t.Errorf("tracer = %+v; want nil", mc.tracer)
	}
}
