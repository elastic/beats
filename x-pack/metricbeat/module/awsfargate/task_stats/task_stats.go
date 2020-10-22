// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package task_stats

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/awsfargate"
)

var metricsetName = "task_stats"

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet(awsfargate.ModuleName, metricsetName, New,
		mb.DefaultMetricSet(),
	)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	*awsfargate.MetricSet
	logger *logp.Logger
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	logger := logp.NewLogger(metricsetName)
	metricSet, err := awsfargate.NewMetricSet(base)
	if err != nil {
		return nil, errors.Wrap(err, "error creating aws metricset")
	}

	return &MetricSet{
		MetricSet: metricSet,
		logger:    logger,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	// Get response from ${ECS_CONTAINER_METADATA_URI_V4}/task/stats
	ecsURI, ok := os.LookupEnv("ECS_CONTAINER_METADATA_URI_V4")
	if !ok {
		err := fmt.Errorf("lookup $ECS_CONTAINER_METADATA_URI_V4 failed")
		m.logger.Error(err)
		return err
	}

	resp, err := http.Get(fmt.Sprintf("%s/task/stats", ecsURI))
	if err != nil {
		err := fmt.Errorf("http.Get failed: %w", err)
		m.logger.Error(err)
		return err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err := fmt.Errorf("ioutil.ReadAll failed: %w", err)
		m.logger.Error(err)
		return err
	}

	var output map[string]interface{}
	err = json.Unmarshal(body, &output)
	if err != nil {
		err := fmt.Errorf("json.Unmarshal failed: %w", err)
		m.logger.Error(err)
		return err
	}

	var events []mb.Event
	for queryID, taskStats := range output {
		event := mb.Event{
			ID:              queryID,
			MetricSetFields: common.MapStr{},
		}

		taskMeta := taskStats.(map[string]interface{})
		readTimestamp := taskMeta["read"].(string)
		timestamp, err := time.Parse(time.RFC3339, readTimestamp)
		if err != nil {
			m.logger.Warn(fmt.Errorf("parsing timestamp %s failed, use current timestamp in event instead", readTimestamp))
		} else {
			event.Timestamp = timestamp
		}

		cpuStats := taskMeta["cpu_stats"].(map[string]interface{})
		memoryStats := taskMeta["memory_stats"].(map[string]interface{})
		networks := taskMeta["networks"].(map[string]interface{})

		cpuUsage := cpuStats["cpu_usage"].(map[string]interface{})
		throttlingData := cpuStats["throttling_data"].(map[string]interface{})
		memStats := memoryStats["stats"].(map[string]interface{})

		cpuStatsData := common.MapStr{
			"cpu_usage": common.MapStr{
				"total_usage":         cpuUsage["total_usage"],
				"usage_in_kernelmode": cpuUsage["usage_in_kernelmode"],
				"usage_in_usermode":   cpuUsage["usage_in_usermode"],
			},
			"throttling_data": common.MapStr{
				"periods":           throttlingData["periods"],
				"throttled_periods": throttlingData["throttled_periods"],
				"throttled_time":    throttlingData["throttled_time"],
			},
		}

		memoryStatsData := common.MapStr{
			"usage":     memoryStats["usage"],
			"max_usage": memoryStats["max_usage"],
			"stats": common.MapStr{
				"active_anon":               memStats["active_anon"],
				"active_file":               memStats["active_file"],
				"cache":                     memStats["cache"],
				"dirty":                     memStats["dirty"],
				"hierarchical_memory_limit": memStats["hierarchical_memory_limit"],
				"hierarchical_memsw_limit":  memStats["hierarchical_memsw_limit"],
				"inactive_anon":             memStats["inactive_anon"],
				"inactive_file":             memStats["inactive_file"],
				"mapped_file":               memStats["mapped_file"],
				"pgfault":                   memStats["pgfault"],
				"pgmajfault":                memStats["pgmajfault"],
				"pgpgin":                    memStats["pgpgin"],
				"pgpgout":                   memStats["pgpgout"],
				"rss":                       memStats["rss"],
				"rss_huge":                  memStats["rss_huge"],
				"total_active_anon":         memStats["total_active_anon"],
				"total_active_file":         memStats["total_active_file"],
				"total_cache":               memStats["total_cache"],
				"total_dirty":               memStats["total_dirty"],
				"total_inactive_anon":       memStats["total_inactive_anon"],
				"total_inactive_file":       memStats["total_inactive_file"],
				"total_mapped_file":         memStats["total_mapped_file"],
				"total_pgfault":             memStats["total_pgfault"],
				"total_pgmajfault":          memStats["total_pgmajfault"],
				"total_pgpgin":              memStats["total_pgpgin"],
				"total_pgpgout":             memStats["total_pgpgout"],
				"total_rss":                 memStats["total_rss"],
				"total_rss_huge":            memStats["total_rss_huge"],
				"total_unevictable":         memStats["total_unevictable"],
				"total_writeback":           memStats["total_writeback"],
				"unevictable":               memStats["unevictable"],
				"writeback":                 memStats["writeback"],
			},
			"limit": memoryStats["limit"],
		}

		networksData := common.MapStr{}
		for networkName, network := range networks {
			networkData := network.(map[string]interface{})
			dataPerInterface := common.MapStr{
				"rx_bytes": networkData["rx_bytes"],
				"rx_packets": networkData["rx_packets"],
				"rx_errors": networkData["rx_errors"],
				"rx_dropped": networkData["rx_dropped"],
				"tx_bytes": networkData["tx_bytes"],
				"tx_packets": networkData["tx_packets"],
				"tx_errors": networkData["tx_errors"],
				"tx_dropped": networkData["tx_dropped"],
			}
			networksData.Put(networkName, dataPerInterface)
		}

		event.MetricSetFields.Put("name", taskMeta["name"])
		event.MetricSetFields.Put("cpu_stats", cpuStatsData)
		event.MetricSetFields.Put("memory_stats", memoryStatsData)
		event.MetricSetFields.Put("networks", networksData)
		events = append(events, event)
	}

	// report events
	for _, event := range events {
		if reported := report.Event(event); !reported {
			m.logger.Debug("Fetch interrupted, failed to emit event")
			return nil
		}
	}

	return nil
}
