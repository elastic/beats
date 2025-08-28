// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package task_stats

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	taskStatsJson = `{
		"query-metadata-1": {
			"read": "2020-04-06T16:12:01.090148907Z",
			"preread": "2020-04-06T16:12:01.090148907Z",
			"cpu_stats": {
				"cpu_usage": {
					"percpu_usage": [1800000000, 500000000, 0, 0, 0, 0, 0, 0],
					"total_usage": 2300000000, "usage_in_kernelmode": 1520000000, "usage_in_usermode": 490000000
				},
				"online_cpus": 2,
				"system_cpu_usage": 1420180000000,
				"throttling_data": {"periods": 0, "throttled_periods": 0, "throttled_time": 0}},
			"precpu_stats": {
				"cpu_usage": {
					"percpu_usage": [1600000000, 300000000, 0, 0, 0, 0, 0, 0],
					"total_usage": 1900000000, "usage_in_kernelmode": 1520000000, "usage_in_usermode": 490000000
				},
				"online_cpus": 2,
				"system_cpu_usage": 1418180000000,
				"throttling_data": {"periods": 0, "throttled_periods": 0, "throttled_time": 0}},
			"memory_stats": {"limit": 8362348544, "usage": 4390912, "max_usage": 6488064, "stats": {"total_rss": 278528}},
			"name": "query-metadata-1",
			"id": "query-metadata-1",
			"networks": {"eth0": {"rx_bytes": 1802, "rx_packets": 19, "rx_errors": 0, "rx_dropped": 0,
            	"tx_bytes": 567, "tx_packets": 7, "tx_errors": 0, "tx_dropped": 0}}
		}}`

	taskRespJson = `{
		"Cluster": "arn:aws:ecs:us-west-2:123:cluster/default",
		"TaskARN": "arn:aws:ecs:us-west-2:123:task/default/febee207c04a",
		"Family": "query-metadata-1",
		"Revision": "7",
        "DesiredStatus": "RUNNING",
        "KnownStatus": "ACTIVATING",
		"Limits": {
		  "Memory": 7168
		},
		"Containers": [{
			"DockerId": "query-metadata-1",
			"Name": "query-metadata",
			"Image": "mreferre/eksutils",
			"Labels": {
				"com.amazonaws.ecs.cluster": "arn:aws:ecs:us-west-2:111122223333:cluster/default",
				"com.amazonaws.ecs.container-name": "query-metadata",
				"com.amazonaws.ecs.task-arn": "arn:aws:ecs:us-west-2:111122223333:task/default/febee046097849aba589d4435207c04a",
				"com.amazonaws.ecs.task-definition-family": "query-metadata",
				"com.amazonaws.ecs.task-definition-version": "7"},
			"Limits": {
				"Memory": 3328
			}
		}]
	}`
)

func TestGetTaskStats(t *testing.T) {
	taskStatsResp := &http.Response{
		Body: io.NopCloser(bytes.NewReader([]byte(taskStatsJson))),
	}

	taskStatsOutput, err := getTaskStats(taskStatsResp)
	assert.NoError(t, err)
	assert.Equal(t, uint64(2300000000), taskStatsOutput["query-metadata-1"].CPUStats.CPUUsage.TotalUsage)
}

func TestGetTask(t *testing.T) {
	taskResp := &http.Response{
		Body: io.NopCloser(bytes.NewReader([]byte(taskRespJson))),
	}

	taskOutput, err := getTask(taskResp)
	assert.NoError(t, err)

	assert.Equal(t, "arn:aws:ecs:us-west-2:123:cluster/default", taskOutput.Cluster)
	assert.Equal(t, "arn:aws:ecs:us-west-2:123:task/default/febee207c04a", taskOutput.TaskARN)
	assert.Equal(t, "query-metadata-1", taskOutput.Family)
	assert.Equal(t, "7", taskOutput.Revision)
	assert.Equal(t, "RUNNING", taskOutput.DesiredStatus)
	assert.Equal(t, "ACTIVATING", taskOutput.KnownStatus)

	assert.Equal(t, uint64(7168), taskOutput.Limit.Memory)
	assert.Equal(t, 1, len(taskOutput.Containers))
	assert.Equal(t, "query-metadata-1", taskOutput.Containers[0].DockerId)
	assert.Equal(t, "query-metadata", taskOutput.Containers[0].Name)
	assert.Equal(t, "mreferre/eksutils", taskOutput.Containers[0].Image)
	assert.Equal(t, 5, len(taskOutput.Containers[0].Labels))
	assert.Equal(t, uint64(3328), taskOutput.Containers[0].Limits.Memory)
}

func TestGetStatsList(t *testing.T) {
	taskStatsResp := &http.Response{
		Body: io.NopCloser(bytes.NewReader([]byte(taskStatsJson))),
	}

	taskStatsOutput, err := getTaskStats(taskStatsResp)
	assert.NoError(t, err)

	taskResp := &http.Response{
		Body: io.NopCloser(bytes.NewReader([]byte(taskRespJson))),
	}

	taskOutput, err := getTask(taskResp)
	assert.NoError(t, err)

	formattedStats := getStatsList(taskStatsOutput, taskOutput)
	assert.Equal(t, 1, len(formattedStats))
}

func TestGetCPUStats(t *testing.T) {
	taskStatsResp := &http.Response{
		Body: io.NopCloser(bytes.NewReader([]byte(taskStatsJson))),
	}

	taskStatsOutput, err := getTaskStats(taskStatsResp)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(taskStatsOutput))

	cpuStats := getCPUStats(taskStatsOutput["query-metadata-1"])
	assert.Equal(t, 0.4, cpuStats.TotalUsage)
	assert.Equal(t, 0.2, cpuStats.TotalUsageNormalized)
}

func TestGetRegionAndClusterName(t *testing.T) {
	labels := map[string]string{}
	labels["com_amazonaws_ecs_cluster"] = "arn:aws:ecs:us-east-1:1234:cluster/metricbeat-fargate-1"
	regionName, clusterName := getRegionAndClusterName(labels)
	assert.Equal(t, "us-east-1", regionName)
	assert.Equal(t, "metricbeat-fargate-1", clusterName)
}

func TestGenerateIdentifier(t *testing.T) {
	assert.Equal(t, "container-name/123", generateIdentifier("container-name", "123"))
}
