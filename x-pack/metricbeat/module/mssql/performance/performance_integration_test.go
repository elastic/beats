// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration
// +build integration

package performance

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	mtest "github.com/elastic/beats/v7/x-pack/metricbeat/module/mssql/testing"
	"github.com/elastic/elastic-agent-libs/logp"
)

type keyAssertion struct {
	key       string
	assertion func(v interface{}, key string)
}

func TestFetch(t *testing.T) {
	logp.TestingSetup()
	service := compose.EnsureUp(t, "mssql")

	f := mbtest.NewReportingMetricSetV2(t, mtest.GetConfig(service.Host(), "performance"))
	events, errs := mbtest.ReportingFetchV2(f)
	if len(errs) > 0 {
		t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
	}
	assert.NotEmpty(t, events)

	float64Assertion := func(f func(float64) bool) func(v interface{}, key string) {
		return func(v interface{}, key string) {
			value, ok := v.(float64)
			if !ok {
				t.Fatalf("%v is not a float64, but %T", key, v)
			}

			assert.Truef(t, f(value), "Value '%d' on field '%s' was not the expected value", value, key)
		}
	}

	int64Assertion := func(f func(int64) bool) func(v interface{}, key string) {
		return func(v interface{}, key string) {
			value, ok := v.(int64)
			if !ok {
				t.Fatalf("%v is not a int64, but %T", key, v)
			}

			assert.Truef(t, f(value), "Value '%d' on field '%s' was not the expected value", value, key)
		}
	}

	int64HigherThanZero := func(v int64) bool {
		return v > 0
	}

	int64EqualOrHigherThanZero := func(v int64) bool {
		return v >= 0
	}

	int64EqualZero := func(v int64) bool {
		return v == 0
	}

	float64HigherThanZero := func(v float64) bool {
		return v > 0
	}

	keys := []keyAssertion{
		{key: "page_splits_per_sec", assertion: int64Assertion(int64HigherThanZero)},
		{key: "buffer.page_life_expectancy.sec", assertion: int64Assertion(int64HigherThanZero)},
		{key: "lock_waits_per_sec", assertion: int64Assertion(int64EqualOrHigherThanZero)},
		{key: "user_connections", assertion: int64Assertion(int64HigherThanZero)},
		{key: "transactions", assertion: int64Assertion(int64EqualOrHigherThanZero)},
		{key: "active_temp_tables", assertion: int64Assertion(int64EqualZero)},
		{key: "connections_reset_per_sec", assertion: int64Assertion(int64HigherThanZero)},
		{key: "logouts_per_sec", assertion: int64Assertion(int64HigherThanZero)},
		{key: "logins_per_sec", assertion: int64Assertion(int64HigherThanZero)},
		{key: "recompilations_per_sec", assertion: int64Assertion(int64EqualOrHigherThanZero)},
		{key: "compilations_per_sec", assertion: int64Assertion(int64HigherThanZero)},
		{key: "batch_requests_per_sec", assertion: int64Assertion(int64HigherThanZero)},
		{key: "buffer.cache_hit.pct", assertion: float64Assertion(float64HigherThanZero)},
		{key: "buffer.checkpoint_pages_per_sec", assertion: int64Assertion(int64HigherThanZero)},
		{key: "buffer.database_pages", assertion: int64Assertion(int64HigherThanZero)},
		{key: "buffer.target_pages", assertion: int64Assertion(int64HigherThanZero)},
	}
	for _, keyAssertion := range keys {
		var found bool

		for _, event := range events {
			value, err := event.MetricSetFields.GetValue(keyAssertion.key)
			if err != nil {
				continue
			}
			found = true

			keyAssertion.assertion(value, keyAssertion.key)
		}

		if !found {
			t.Fatalf("Key '%s' not found", keyAssertion.key)
		}
	}
}
