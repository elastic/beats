// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build integration

package stats

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/tests/compose"
	"github.com/elastic/beats/v7/metricbeat/mb"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"

	// Register input module and metricset
	_ "github.com/elastic/beats/v7/metricbeat/module/prometheus"
	_ "github.com/elastic/beats/v7/metricbeat/module/prometheus/collector"
)

func init() {
	// To be moved to some kind of helper
	os.Setenv("BEAT_STRICT_PERMS", "false")
	mb.Registry.SetSecondarySource(mb.NewLightModulesSource("../../../module"))
}

func TestFetch(t *testing.T) {
	service := compose.EnsureUp(t, "cockroachdb")

	f := mbtest.NewFetcher(t, getConfig(service.Host()))
	events, errs := f.FetchEvents()
	if len(errs) > 0 {
		t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
	}
	assert.NotEmpty(t, events)
	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), events[0])
}

func getConfig(host string) map[string]interface{} {
	return map[string]interface{}{
		"module":     "cockroachdb",
		"metricsets": []string{"status"},
		"hosts":      []string{host},
	}
}
