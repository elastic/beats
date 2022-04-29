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
		in   *extractedData
		want string
	}{
		{
			in: &extractedData{
				sysmetricMetrics: []sysmetricMetric{{
					beginTime:   sql.NullString{String: "2022-04-27T12:18:57+05:30", Valid: true},
					endTime:     sql.NullString{String: "2022-04-27T12:19:57+05:30", Valid: true},
					intsizeCsec: sql.NullFloat64{Float64: 6021, Valid: true},
					groupId:     sql.NullInt64{Int64: 2, Valid: true},
					metricId:    sql.NullInt64{Int64: 2000, Valid: true},
					name:        sql.NullString{String: "Buffer Cache Hit Ratio", Valid: true},
					value:       sql.NullFloat64{Float64: 100, Valid: true},
					metricUnit:  sql.NullString{String: "% (LogRead - PhyRead)/LogRead", Valid: true},
					conId:       sql.NullFloat64{Float64: 0, Valid: true}}},
			},
			want: `{"metrics":{"begin_time":"2022-04-27T12:18:57+05:30","container_id":0,"end_time":"2022-04-27T12:19:57+05:30","group_id":2,"interval_size_csec":6021,"metric_id":2000,"metric_unit":"% (LogRead - PhyRead)/LogRead","name":"Buffer Cache Hit Ratio","value":100}}`,
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
