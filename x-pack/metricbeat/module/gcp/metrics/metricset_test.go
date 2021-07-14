// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package metrics

import "testing"

func Test_metricsConfig_MetricPrefix(t *testing.T) {
	type fields struct {
		ServiceName         string
		ServiceMetricPrefix string
		MetricTypes         []string
		Aligner             string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name:   "only service name",
			fields: fields{"billing", "", []string{}, ""},
			want:   "billing.googleapis.com/",
		},
		{
			name:   "service metric prefix override",
			fields: fields{"billing", "foobar/", []string{}, ""},
			want:   "foobar/",
		},
		{
			name:   "service metric prefix override (without trailing /)",
			fields: fields{"billing", "foobar", []string{}, ""},
			want:   "foobar/",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := metricsConfig{
				ServiceName:         tt.fields.ServiceName,
				ServiceMetricPrefix: tt.fields.ServiceMetricPrefix,
				MetricTypes:         tt.fields.MetricTypes,
				Aligner:             tt.fields.Aligner,
			}
			if got := mc.MetricPrefix(); got != tt.want {
				t.Errorf("metricsConfig.MetricPrefix() = %v, want %v", got, tt.want)
			}
		})
	}
}
