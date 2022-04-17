// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration
// +build integration

package stats

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/menderesk/beats/v7/libbeat/tests/compose"
	mbtest "github.com/menderesk/beats/v7/metricbeat/mb/testing"
)

func TestData(t *testing.T) {
	service := compose.EnsureUp(t, "stan")

	m := mbtest.NewFetcher(t, getConfig(service.Host()))
	m.WriteEvents(t, "")
}

func TestFetch(t *testing.T) {
	service := compose.EnsureUp(t, "stan")

	m := mbtest.NewFetcher(t, getConfig(service.Host()))
	events, errs := m.FetchEvents()
	if len(errs) > 0 {
		t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
	}
	assert.NotEmpty(t, events)
	t.Logf("%s/%s event: %+v", m.Module().Name(), m.Name(), events[0])
}

func getConfig(host string) map[string]interface{} {
	return map[string]interface{}{
		"module":     "stan",
		"metricsets": []string{"stats"},
		"hosts":      []string{host},
	}
}
