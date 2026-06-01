// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azuread

import (
	"reflect"
	"testing"
	"time"

	"github.com/elastic/elastic-agent-libs/config"
	ecentraid "github.com/elastic/entcollect/provider/entraid"
)

// TestMinimalConfigRoundTrip verifies that every exported field of
// ecentraid.Config is represented in the localConf mirror inside
// minimalProvider. It does this in two ways:
//
//  1. Field-count assertion — if ecentraid.Config gains a field, the
//     count check fails, forcing the developer to update localConf
//     and this test.
//  2. Functional round-trip — all fields are set to non-default sentinel
//     values; the returned sync intervals must match, confirming that
//     at least the duration fields are wired through end-to-end.
func TestMinimalConfigRoundTrip(t *testing.T) {
	const wantFields = 13
	if got := reflect.TypeOf(ecentraid.Config{}).NumField(); got != wantFields {
		t.Fatalf("ecentraid.Config has %d exported fields, want %d; "+
			"update localConf inside minimalProvider and this test", got, wantFields)
	}

	wantSync := 48 * time.Hour
	wantUpdate := 30 * time.Minute

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
	})

	_, gotSync, gotUpdate, err := minimalProvider(cfg, nil)
	if err != nil {
		t.Fatalf("minimalProvider returned unexpected error: %v", err)
	}
	if gotSync != wantSync {
		t.Errorf("sync_interval: got %v, want %v", gotSync, wantSync)
	}
	if gotUpdate != wantUpdate {
		t.Errorf("update_interval: got %v, want %v", gotUpdate, wantUpdate)
	}
}
