// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package stackdriver

import (
	"testing"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/stretchr/testify/assert"
	"google.golang.org/genproto/googleapis/api/metric"
	"google.golang.org/genproto/googleapis/api/monitoredres"
	"google.golang.org/genproto/googleapis/monitoring/v3"
)

var fake *monitoring.TimeSeries = &monitoring.TimeSeries{
	Resource: &monitoredres.MonitoredResource{
		Type: "gce_instance",
		Labels: map[string]string{
			"instance_id": "4624337448093162893",
			"project_id":  "elastic-metricbeat",
			"zone":        "us-central1-a",
		},
	},
	Metadata: &monitoredres.MonitoredResourceMetadata{
		UserLabels: map[string]string{
			"user": "label",
		},
	},
	Metric: &metric.Metric{
		Labels: map[string]string{
			"instance_name": "instance-1",
		},
		Type: "compute.googleapis.com/instance/cpu/usage_time",
	},
	MetricKind: metric.MetricDescriptor_GAUGE,
	ValueType:  metric.MetricDescriptor_DOUBLE,
	Points: []*monitoring.Point{{
		Value: &monitoring.TypedValue{
			Value: &monitoring.TypedValue_DoubleValue{DoubleValue: 0.0041224284852319215},
		},
		Interval: &monitoring.TimeInterval{
			StartTime: &timestamp.Timestamp{
				Seconds: 1569932700,
			},
			EndTime: &timestamp.Timestamp{
				Seconds: 1569932700,
			},
		},
	}, {
		Value: &monitoring.TypedValue{
			Value: &monitoring.TypedValue_DoubleValue{DoubleValue: 0.004205757571772513},
		},
		Interval: &monitoring.TimeInterval{
			StartTime: &timestamp.Timestamp{
				Seconds: 1569932640,
			},
			EndTime: &timestamp.Timestamp{
				Seconds: 1569932640,
			},
		},
	}},
}

var metrics = []string{
	"compute.googleapis.com/instance/cpu/utilization",
	"compute.googleapis.com/instance/disk/read_bytes_count",
	"compute.googleapis.com/http/server/response_latencies",
}

func TestCleanMetricNameString(t *testing.T) {
	cases := []struct {
		title              string
		metricType         string
		aligner            string
		expectedMetricName string
	}{
		{
			"test construct metric name with ALIGN_MEAN aligner",
			"compute.googleapis.com/instance/cpu/usage_time",
			"ALIGN_MEAN",
			"instance.cpu.usage_time.avg",
		},
		{
			"test construct metric name with ALIGN_NONE aligner",
			"compute.googleapis.com/instance/cpu/utilization",
			"ALIGN_NONE",
			"instance.cpu.utilization.value",
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			metricName := cleanMetricNameString(c.metricType, c.aligner)
			assert.Equal(t, c.expectedMetricName, metricName)
		})
	}
}
