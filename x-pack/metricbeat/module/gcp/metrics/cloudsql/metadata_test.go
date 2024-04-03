// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cloudsql

import (
	"testing"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"google.golang.org/genproto/googleapis/api/metric"
	"google.golang.org/genproto/googleapis/api/monitoredres"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gotest.tools/assert"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

var fake = &monitoring.TimeSeries{
	Resource: &monitoredres.MonitoredResource{
		Type: "gce_instance",
		Labels: map[string]string{
			"database_id": "db",
			"project_id":  "elastic-metricbeat",
			"region":      "us-central1",
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
			StartTime: &timestamppb.Timestamp{
				Seconds: 1569932700,
			},
			EndTime: &timestamppb.Timestamp{
				Seconds: 1569932700,
			},
		},
	}, {
		Value: &monitoring.TypedValue{
			Value: &monitoring.TypedValue_DoubleValue{DoubleValue: 0.004205757571772513},
		},
		Interval: &monitoring.TimeInterval{
			StartTime: &timestamppb.Timestamp{
				Seconds: 1569932640,
			},
			EndTime: &timestamppb.Timestamp{
				Seconds: 1569932640,
			},
		},
	}},
}

var m = &metadataCollector{
	projectID: "projectID",
}

func TestInstanceID(t *testing.T) {
	instanceID := m.instanceID(fake)
	assert.Equal(t, "db", instanceID)
}

func TestInstanceRegion(t *testing.T) {
	zone := m.instanceRegion(fake)
	assert.Equal(t, "us-central1", zone)
}

func TestGetDatabaseNameAndVersion(t *testing.T) {
	cases := []struct {
		name     string
		db       string
		expected mapstr.M
	}{
		{
			name: "sql unspecified",
			db:   "SQL_DATABASE_VERSION_UNSPECIFIED",
			expected: mapstr.M{
				"name":    "sql",
				"version": "unspecified",
			},
		},
		{
			name: "mysql 5.1",
			db:   "MYSQL_5_1",
			expected: mapstr.M{
				"name":    "mysql",
				"version": "5.1",
			},
		},
		{
			name: "mysql 5.5",
			db:   "MYSQL_5_5",
			expected: mapstr.M{
				"name":    "mysql",
				"version": "5.5",
			},
		},
		{
			name: "mysql 5.6",
			db:   "MYSQL_5_6",
			expected: mapstr.M{
				"name":    "mysql",
				"version": "5.6",
			},
		},
		{
			name: "mysql 5.7",
			db:   "MYSQL_5_7",
			expected: mapstr.M{
				"name":    "mysql",
				"version": "5.7",
			},
		},
		{
			name: "postgres 9.6",
			db:   "POSTGRES_9_6",
			expected: mapstr.M{
				"name":    "postgres",
				"version": "9.6",
			},
		},
		{
			name: "postgres 11",
			db:   "POSTGRES_11",
			expected: mapstr.M{
				"name":    "postgres",
				"version": "11",
			},
		},
		{
			name: "SQLSERVER_2017_STANDARD",
			db:   "SQLSERVER_2017_STANDARD",
			expected: mapstr.M{
				"name":    "sqlserver",
				"version": "2017_standard",
			},
		},
		{
			name: "SQLSERVER_2017_ENTERPRISE",
			db:   "SQLSERVER_2017_ENTERPRISE",
			expected: mapstr.M{
				"name":    "sqlserver",
				"version": "2017_enterprise",
			},
		},
		{
			name: "SQLSERVER_2017_EXPRESS",
			db:   "SQLSERVER_2017_EXPRESS",
			expected: mapstr.M{
				"name":    "sqlserver",
				"version": "2017_express",
			},
		},
		{
			name: "SQLSERVER_2017_WEB",
			db:   "SQLSERVER_2017_WEB",
			expected: mapstr.M{
				"name":    "sqlserver",
				"version": "2017_web",
			},
		},
		{
			name: "POSTGRES_10",
			db:   "POSTGRES_10",
			expected: mapstr.M{
				"name":    "postgres",
				"version": "10",
			},
		},
		{
			name: "POSTGRES_12",
			db:   "POSTGRES_12",
			expected: mapstr.M{
				"name":    "postgres",
				"version": "12",
			},
		},
		{
			name: "MYSQL_8_0",
			db:   "MYSQL_8_0",
			expected: mapstr.M{
				"name":    "mysql",
				"version": "8.0",
			},
		},
		{
			name: "MYSQL_8_0_18",
			db:   "MYSQL_8_0_18",
			expected: mapstr.M{
				"name":    "mysql",
				"version": "8.0.18",
			},
		},
		{
			name: "SQLSERVER_2019_STANDARD",
			db:   "SQLSERVER_2019_STANDARD",
			expected: mapstr.M{
				"name":    "sqlserver",
				"version": "2019_standard",
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			db := getDatabaseNameAndVersion(c.db)
			assert.DeepEqual(t, db, c.expected)
		})
	}
}
