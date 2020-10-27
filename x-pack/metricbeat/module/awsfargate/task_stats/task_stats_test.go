// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package task_stats

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/awsfargate"
)

func TestMappingEvent(t *testing.T) {
	cpuStatsExpected := map[string]interface{}{
		"cpu_usage": map[string]interface{}{
			"total_usage":         484750075,
			"usage_in_kernelmode": 60000000,
			"usage_in_usermode":   330000000,
		},
		"throttling_data": map[string]interface{}{
			"periods":           0,
			"throttled_periods": 0,
			"throttled_time":    0,
		},
	}

	memoryStatsExpected := map[string]interface{}{
		"usage":     20606976,
		"max_usage": 20729856,
		"stats": map[string]interface{}{
			"pgpgout":                   5518,
			"total_unevictable":         0,
			"total_pgfault":             13920,
			"inactive_anon":             0,
			"pgmajfault":                0,
			"total_cache":               53248,
			"total_inactive_file":       49152,
			"total_rss":                 18931712,
			"active_anon":               18911232,
			"rss":                       18931712,
			"total_dirty":               4096,
			"total_writeback":           0,
			"cache":                     53248,
			"writeback":                 0,
			"hierarchical_memory_limit": 2147483648,
			"total_pgpgout":             5518,
			"pgfault":                   13920,
			"pgpgin":                    10153,
			"total_pgpgin":              10153,
			"dirty":                     4096,
			"active_file":               24576,
			"total_active_anon":         18911232,
			"inactive_file":             49152,
			"total_pgmajfault":          0,
			"total_inactive_anon":       0,
			"total_rss_huge":            0,
			"hierarchical_memsw_limit":  9223372036854772000,
			"mapped_file":               0,
			"rss_huge":                  0,
			"unevictable":               0,
			"total_mapped_file":         0,
			"total_active_file":         24576,
		},
		"limit": 3937787904,
	}

	networkEth0Expected := map[string]interface{}{
		"tx_dropped": 0,
		"rx_bytes":   220597168,
		"rx_packets": 151398,
		"rx_errors":  0,
		"rx_dropped": 0,
		"tx_bytes":   1473393,
		"tx_packets": 25392,
		"tx_errors":  0,
	}

	taskMetadata := map[string]interface{}{
		"/ecs-test-metricbeat": map[string]interface{}{
			"name":         "query-metadata",
			"id":           "/ecs-test-metricbeat",
			"read":         "2020-04-06T16:12:01.090148907Z",
			"cpu_stats":    cpuStatsExpected,
			"memory_stats": memoryStatsExpected,
			"networks": map[string]interface{}{
				"eth0": networkEth0Expected,
			},
		},
	}

	m := MetricSet{
		&awsfargate.MetricSet{},
		logp.NewLogger("test"),
	}

	event := m.createEvent(taskMetadata["/ecs-test-metricbeat"])
	name, err := event.MetricSetFields.GetValue("name")
	assert.NoError(t, err)
	assert.Equal(t, "query-metadata", name)

	cpuStatsOutput, err := event.MetricSetFields.GetValue("cpu_stats")
	assert.NoError(t, err)
	assert.NotEmpty(t, cpuStatsOutput)

	memStatsOutput, err := event.MetricSetFields.GetValue("memory_stats")
	assert.NoError(t, err)
	assert.NotEmpty(t, memStatsOutput)

	networksOutput, err := event.MetricSetFields.GetValue("networks")
	assert.NotEmpty(t, networksOutput)
	assert.NoError(t, err)
}
