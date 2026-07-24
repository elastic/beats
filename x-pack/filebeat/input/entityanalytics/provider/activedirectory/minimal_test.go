// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package activedirectory

import (
	"reflect"
	"testing"
	"time"

	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	ecad "github.com/elastic/entcollect/provider/ad"
)

// TestMinimalConfigRoundTrip verifies that every exported field of
// ecad.Config is represented in the localConf mirror inside
// minimalProvider. It does this in two ways:
//
//  1. Field-count assertion — if ecad.Config gains a field, the count
//     check fails, forcing the developer to update localConf and this test.
//     The TLS field is excluded from the count because it is built by
//     buildTLS, not unpacked directly.
//  2. Functional round-trip — all fields are set to non-default sentinel
//     values; the returned sync intervals must match, confirming that at
//     least the duration fields are wired through end-to-end.
func TestMinimalConfigRoundTrip(t *testing.T) {
	const wantFields = 15 // includes TLS which is json:"-" but still a struct field
	if got := reflect.TypeOf(ecad.Config{}).NumField(); got != wantFields {
		t.Fatalf("ecad.Config has %d exported fields, want %d; "+
			"update localConf inside minimalProvider and this test", got, wantFields)
	}

	wantSync := 48 * time.Hour
	wantUpdate := 30 * time.Minute

	cfg := config.MustNewConfigFrom(map[string]any{
		"ad_url":               "ldap://dc.example.com",
		"ad_base_dn":           "DC=example,DC=com",
		"ad_user":              "cn=admin,dc=example,dc=com",
		"ad_password":          "secret",
		"dataset":              "users",
		"user_query":           "(&(objectCategory=person))",
		"device_query":         "(&(objectClass=computer))",
		"include_empty_groups": true,
		"user_attributes":      []string{"cn", "mail"},
		"group_attributes":     []string{"cn"},
		"ad_paging_size":       500,
		"idset_shards":         32,
		"sync_interval":        wantSync.String(),
		"update_interval":      wantUpdate.String(),
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

func TestMinimalConfigTLS(t *testing.T) {
	cfg := config.MustNewConfigFrom(map[string]any{
		"ad_url":      "ldaps://dc.example.com:636",
		"ad_base_dn":  "DC=example,DC=com",
		"ad_user":     "cn=admin,dc=example,dc=com",
		"ad_password": "secret",
		"ssl": map[string]any{
			"verification_mode": "none",
		},
	})

	log := logptest.NewTestingLogger(t, "test")
	p, _, _, err := minimalProvider(cfg, log)
	if err != nil {
		t.Fatalf("minimalProvider returned unexpected error: %v", err)
	}
	if p == nil {
		t.Fatal("provider is nil")
	}
}
