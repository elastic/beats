// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package ec2

import (
	"os"

	"github.com/elastic/beats/v7/metricbeat/mb"

	// Register input module and metricset
	_ "github.com/elastic/beats/v7/x-pack/metricbeat/module/aws"
	_ "github.com/elastic/beats/v7/x-pack/metricbeat/module/aws/cloudwatch"
)

func init() {
	// To be moved to some kind of helper
	os.Setenv("BEAT_STRICT_PERMS", "false")
	mb.Registry.SetSecondarySource(mb.NewLightModulesSource("../../../module"))
}

func TestCreateCloudWatchEventsWithInstanceName(t *testing.T) {
	expectedEvent := mb.Event{
		RootFields: common.MapStr{
			"cloud": common.MapStr{
				"region":            regionName,
				"provider":          "aws",
				"instance":          common.MapStr{"id": "i-123", "name": "test-instance"},
				"machine":           common.MapStr{"type": "t2.medium"},
				"availability_zone": "us-west-1a",
			},
			"host": common.MapStr{
				"cpu": common.MapStr{"pct": 0.25},
				"id":  "i-123",
			},
		},
		MetricSetFields: common.MapStr{
			"tags": common.MapStr{
				"app_kubernetes_io/name": "foo",
				"helm_sh/chart":          "foo-chart",
				"Name":                   "test-instance",
			},
		},
	}
	svcEC2Mock := &MockEC2Client{}
	instanceIDs, instancesOutputs, err := getInstancesPerRegion(svcEC2Mock)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(instanceIDs))
	instanceID := instanceIDs[0]
	assert.Equal(t, instanceID, instanceID)
	timestamp := time.Now()

	getMetricDataOutput := []cloudwatch.MetricDataResult{
		{
			Id:         &id1,
			Label:      &label1,
			Values:     []float64{0.25},
			Timestamps: []time.Time{timestamp},
		},
	}

	metricSet := MetricSet{
		&aws.MetricSet{},
		logp.NewLogger("test"),
	}

	events, err := metricSet.createCloudWatchEvents(getMetricDataOutput, instancesOutputs, "us-west-1")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(events))

	assert.Equal(t, expectedEvent.MetricSetFields["tags"], events[instanceID].ModuleFields["tags"])

	hostID, err := events[instanceID].RootFields.GetValue("host.id")
	assert.NoError(t, err)
	assert.Equal(t, "i-123", hostID)

	instanceName, err := events[instanceID].RootFields.GetValue("cloud.instance.name")
	assert.NoError(t, err)
	assert.Equal(t, "test-instance", instanceName)
}

func TestNewLabel(t *testing.T) {
	instanceID := "i-123"
	metricName := "CPUUtilization"
	statistic := "Average"
	label := newLabel(instanceID, metricName, statistic).JSON()
	assert.Equal(t, "{\"InstanceID\":\"i-123\",\"MetricName\":\"CPUUtilization\",\"Statistic\":\"Average\"}", label)
}

func TestConvertLabel(t *testing.T) {
	labelStr := "{\"InstanceID\":\"i-123\",\"MetricName\":\"CPUUtilization\",\"Statistic\":\"Average\"}"
	label, err := newLabelFromJSON(labelStr)
	assert.NoError(t, err)
	assert.Equal(t, "i-123", label.InstanceID)
	assert.Equal(t, "CPUUtilization", label.MetricName)
	assert.Equal(t, "Average", label.Statistic)
}
