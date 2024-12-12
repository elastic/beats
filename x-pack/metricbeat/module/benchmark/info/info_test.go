// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package info

import (
	"testing"

	"github.com/stretchr/testify/assert"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
)

func TestFetch(t *testing.T) {
	tests := map[string]struct {
		count uint
	}{
		"one count":  {count: uint(1)},
		"five count": {count: uint(5)},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			config := map[string]interface{}{
				"module":     "benchmark",
				"period":     "1s",
				"metricsets": []string{"info"},
				"count":      tc.count,
			}
			t.Cleanup(func() {
				if t.Failed() {
					t.Logf("Contents of config:\n%v", config)
				}
			})

			f := mbtest.NewFetcher(t, config)
			events, errs := f.FetchEvents()
			assert.Emptyf(t, errs, "errs should be empty, err: %v", errs)
			assert.Equalf(t, int(tc.count), len(events), "events should have %d events not %d, events: %v", int(tc.count), len(events), events)

			for i, event := range events {
				msf := event.MetricSetFields

				ok, err := msf.HasKey("counter")
				assert.Truef(t, ok, "MetricSetFields must contain \"counter\", msf: %v", msf)
				assert.NoErrorf(t, err, "MetricSetFields must contain \"counter\", msf: %v", msf)

				v, err := msf.GetValue("counter")
				assert.NoErrorf(t, err, "MetricSetFields must contain \"counter\", msf: %v", msf)
				assert.Equalf(t, uint(i+1), v, "counter should be %d, was %v", i+1, v)
			}
		})
	}
}
