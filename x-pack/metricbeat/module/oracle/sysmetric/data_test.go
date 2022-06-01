// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package sysmetric

import (
	"database/sql"
	"testing"
)

func TestMetricSetTransform(t *testing.T) {
	tests := []struct {
		in   *collectedData
		want string
	}{
		{
			in: &collectedData{
				sysmetricMetrics: []sysmetricMetric{{
					name:  sql.NullString{String: "Buffer Cache Hit Ratio", Valid: true},
					value: sql.NullFloat64{Float64: 100, Valid: true},
				}},
			},
			want: `{"buffer_cache_hit_ratio":100}`,
		},
	}
	for _, tt := range tests {
		t.Run("Test func transform()", func(t *testing.T) {
			m := &MetricSet{}
			got := m.transform(tt.in)
			if got[0].MetricSetFields.String() != tt.want {
				t.Errorf("MetricSet.transform() = %v, want %v", got[0].MetricSetFields.String(), tt.want)
			}
		})
	}
}
