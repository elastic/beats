// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package jamf

import (
	"reflect"
	"testing"
	"time"

	"github.com/elastic/beats/v7/pkg/config"
	"github.com/elastic/beats/v7/pkg/logp/logptest"
	ecjamf "github.com/elastic/entcollect/provider/jamf"
)

// TestMinimalConfigRoundTrip verifies that every exported field of
// ecjamf.Config is represented in the localConf mirror inside
// minimalConfig. It does this in two ways:
//
//  1. Field-count assertion — if ecjamf.Config gains a field, the count
//     check fails, forcing the developer to update localConf and this test.
//  2. Functional round-trip — every field, and the request and tracer
//     settings, are set to non-default sentinel values; the mapped
//     config must match them, which catches a field whose config tag is
//     wrong.
func TestMinimalConfigRoundTrip(t *testing.T) {
	const wantFields = 8
	if got := reflect.TypeFor[ecjamf.Config]().NumField(); got != wantFields {
		t.Fatalf("ecjamf.Config has %d exported fields, want %d; "+
			"update localConf inside minimalConfig and this test", got, wantFields)
	}

	wantSync := 48 * time.Hour
	wantUpdate := 30 * time.Minute

	cfg := config.MustNewConfigFrom(map[string]any{
		"jamf_tenant":        "tenant.jamfcloud.com",
		"jamf_username":      "user",
		"jamf_password":      "pass",
		"page_size":          50,
		"idset_shards":       32,
		"token_grace_period": "2m",
		"sync_interval":      wantSync.String(),
		"update_interval":    wantUpdate.String(),

		"request.timeout":                         "45s",
		"request.proxy_url":                       "http://proxy.example.com:3128",
		"request.keep_alive.max_idle_connections": 5,
		"request.redirect.forward_headers":        true,
		"request.redirect.headers_ban_list":       []string{"X-Secret"},
		"request.redirect.max_redirects":          3,
		"request.retry.max_attempts":              2,
		"tracer.filename":                         "../../logs/jamf/http-request-trace-*.ndjson",
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
	want := ecjamf.Config{
		TenantID:       "tenant.jamfcloud.com",
		Username:       "user",
		Password:       "pass",
		PageSize:       50,
		IDSetShards:    32,
		TokenGrace:     2 * time.Minute,
		SyncInterval:   wantSync,
		UpdateInterval: wantUpdate,
	}
	if !reflect.DeepEqual(mc.provider, want) {
		t.Errorf("minimalConfig() = %+v; want %+v", mc.provider, want)
	}

	req := mc.request
	if req.Transport.Timeout != 45*time.Second {
		t.Errorf("request.timeout = %v; want 45s", req.Transport.Timeout)
	}
	if req.Transport.Proxy.URL == nil || req.Transport.Proxy.URL.String() != "http://proxy.example.com:3128" {
		t.Errorf("request.proxy_url = %v; want http://proxy.example.com:3128", req.Transport.Proxy.URL)
	}
	if req.KeepAlive.MaxIdleConns != 5 {
		t.Errorf("request.keep_alive.max_idle_connections = %d; want 5", req.KeepAlive.MaxIdleConns)
	}
	if !req.RedirectForwardHeaders || req.RedirectMaxRedirects != 3 || !reflect.DeepEqual(req.RedirectHeadersBanList, []string{"X-Secret"}) {
		t.Errorf("request.redirect = forward %t, max %d, ban %q; want true, 3, [X-Secret]",
			req.RedirectForwardHeaders, req.RedirectMaxRedirects, req.RedirectHeadersBanList)
	}
	if req.Retry == nil {
		t.Error("request.retry = nil; want it recorded so a warning can be logged")
	}
	if mc.tracer == nil || mc.tracer.Filename != "../../logs/jamf/http-request-trace-*.ndjson" {
		t.Errorf("tracer = %+v; want filename ../../logs/jamf/http-request-trace-*.ndjson", mc.tracer)
	}
}
