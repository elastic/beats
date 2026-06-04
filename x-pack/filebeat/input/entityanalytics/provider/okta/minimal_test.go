// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package okta

import (
	"reflect"
	"testing"
	"time"

	"github.com/elastic/elastic-agent-libs/config"
	ecokta "github.com/elastic/entcollect/provider/okta"
)

// TestMinimalConfigRoundTrip verifies that every exported field of
// ecokta.Config is represented in the localConf mirror inside
// minimalProvider. It does this in two ways:
//
//  1. Field-count assertion — if ecokta.Config gains a field, the count
//     check fails, forcing the developer to update localConf and this test.
//  2. Functional round-trip — all fields are set to non-default sentinel
//     values; the returned sync intervals must match, confirming that at
//     least the duration fields are wired through end-to-end.
func TestMinimalConfigRoundTrip(t *testing.T) {
	const wantFields = 11
	if got := reflect.TypeOf(ecokta.Config{}).NumField(); got != wantFields {
		t.Fatalf("ecokta.Config has %d exported fields, want %d; "+
			"update localConf inside minimalProvider and this test", got, wantFields)
	}

	wantSync := 48 * time.Hour
	wantUpdate := 30 * time.Minute
	limitFixed := 50

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
