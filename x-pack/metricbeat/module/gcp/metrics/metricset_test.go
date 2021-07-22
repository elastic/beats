// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package metrics

import "testing"

var fakeMetricsConfig = []metricsConfig{
	{"billing", "", []string{}, ""},
	{"billing", "foobar/", []string{}, ""},
	{"billing", "foobar", []string{}, ""},
}

func Test_metricsConfig_AddPrefixTo(t *testing.T) {
	metric := "awesome/metric"
	tests := []struct {
		name   string
		fields metricsConfig
		want   string
	}{
		{
			name:   "only service name",
			fields: fakeMetricsConfig[0],
			want:   "billing.googleapis.com/" + metric,
		},
		{
			name:   "service metric prefix override",
			fields: fakeMetricsConfig[1],
			want:   "foobar/" + metric,
		},
		{
			name:   "service metric prefix override (without trailing /)",
			fields: fakeMetricsConfig[2],
			want:   "foobar/" + metric,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.fields.AddPrefixTo(metric); got != tt.want {
				t.Errorf("metricsConfig.AddPrefixTo(%s) = %v, want %v", metric, got, tt.want)
			}
		})
	}
}
