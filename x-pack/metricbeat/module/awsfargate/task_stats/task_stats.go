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
	s "github.com/elastic/beats/v7/libbeat/common/schema"
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
	ecsURI, ok := os.LookupEnv("ECS_CONTAINER_METADATA_URI_V4")
	if !ok {
		err := fmt.Errorf("lookup $ECS_CONTAINER_METADATA_URI_V4 failed")
		m.logger.Error(err)
		return err
	}

	// Get response from ${ECS_CONTAINER_METADATA_URI_V4}/task/stats
	taskStatsEndpoint := fmt.Sprintf("%s/task/stats", ecsURI)
	taskStatsOutput, err := queryTaskMetadataEndpoint(taskStatsEndpoint)
	if err != nil {
		err = fmt.Errorf("queryTaskMetadataEndpoint %s failed: %w", taskStatsEndpoint, err)
		m.logger.Error(err)
		return err
	}

	// Collect container metadata information from ${ECS_CONTAINER_METADATA_URI_V4}/task
	taskEndpoint := fmt.Sprintf("%s/task", ecsURI)
	taskOutput, err := queryTaskMetadataEndpoint(taskEndpoint)
	if err != nil {
		err = fmt.Errorf("queryTaskMetadataEndpoint %s failed: %w", taskEndpoint, err)
		m.logger.Error(err)
		return err
	}

	containersMetadata := collectContainerMetadata(taskOutput)

	// Create events for all containers in the same task
	for id, taskStats := range taskStatsOutput {
		metadata := containersMetadata[id]
		event := m.createEvent(taskStats, metadata)

		// report events
		if reported := report.Event(event); !reported {
			m.logger.Debug("Fetch interrupted, failed to emit event")
			return nil
		}
	}
	return nil
}

func queryTaskMetadataEndpoint(taskMetadataEndpoint string) (map[string]interface{}, error) {
	resp, err := http.Get(taskMetadataEndpoint)
	if err != nil {
		return nil, fmt.Errorf("http.Get failed: %w", err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ioutil.ReadAll failed: %w", err)
	}

	var output map[string]interface{}
	err = json.Unmarshal(body, &output)
	if err != nil {
		return nil, fmt.Errorf("json.Unmarshal failed: %w", err)
	}
	return output, nil
}

func collectContainerMetadata(taskOutput map[string]interface{}) map[string]common.MapStr {
	containersMetadata := make(map[string]common.MapStr)
	for _, container := range taskOutput["Containers"].([]interface{}) {
		// basic metadata
		containerMap := container.(map[string]interface{})
		resultContainer, err := schemaContainer.Apply(containerMap, s.FailOnRequired)
		if err != nil {
			continue
		}

		// labels
		labels := containerMap["Labels"].(map[string]interface{})
		labelsMap := map[string]interface{}{}
		for k, v := range labels {
			labelsMap[common.DeDot(k)] = v
		}
		resultContainer.Put("labels", labelsMap)

		// limits
		resultContainer.Put("limits", containerMap["Limits"].(map[string]interface{}))

		dockerID, err := resultContainer.GetValue("docker_id")
		if err != nil {
			continue
		}
		containersMetadata[dockerID.(string)] = resultContainer
	}
	return containersMetadata
}

func (m *MetricSet) createEvent(taskStats interface{}, metadata common.MapStr) mb.Event {
	event := mb.Event{
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

	// name and id
	event.MetricSetFields.Put("name", taskMeta["name"])
	event.MetricSetFields.Put("id", taskMeta["id"])

	// cpu stats
	cpuStats := taskMeta["cpu_stats"].(map[string]interface{})
	resultCPUStats, err := schemaCPUStats.Apply(cpuStats, s.FailOnRequired)
	event.MetricSetFields.Put("cpu_stats", resultCPUStats)

	// precpu stats
	preCpuStats := taskMeta["precpu_stats"].(map[string]interface{})
	resultPreCPUStats, err := schemaCPUStats.Apply(preCpuStats, s.FailOnRequired)
	event.MetricSetFields.Put("precpu_stats", resultPreCPUStats)

	// memory stats
	resultMemoryStats, err := schemaMemoryStats.Apply(taskMeta["memory_stats"].(map[string]interface{}), s.FailOnRequired)
	event.MetricSetFields.Put("memory_stats", resultMemoryStats)

	// networks
	networks := taskMeta["networks"]
	resultNetworkStats := common.MapStr{}
	for name, network := range networks.(map[string]interface{}) {
		resultNetwork, err := schemaNetwork.Apply(network.(map[string]interface{}), s.FailOnRequired)
		if err != nil {
			continue
		}

		resultNetworkStats.Put(name, resultNetwork)
	}
	event.MetricSetFields.Put("networks", resultNetworkStats)

	// add container metadata
	event.MetricSetFields.Put("metadata", metadata)

	// calculate cpu.total.norm.pct
	event = calculateCPU(cpuStats, preCpuStats, resultCPUStats, resultPreCPUStats, event)
	return event
}

func calculateCPU(cpuStats map[string]interface{}, preCpuStats map[string]interface{}, resultCPUStats common.MapStr, resultPreCPUStats common.MapStr, event mb.Event) mb.Event {
	cpuUsage := cpuStats["cpu_usage"].(map[string]interface{})
	if cpuUsage != nil && cpuUsage["percpu_usage"] != nil {
		numCores := float64(len(cpuUsage["percpu_usage"].([]interface{})))
		event.MetricSetFields.Put("cpu.cores", numCores)

		cpuUsage := resultCPUStats["cpu_usage"].(common.MapStr)
		preCpuUsage := resultPreCPUStats["cpu_usage"].(common.MapStr)

		if cpuUsage["total_usage"] != nil && preCpuUsage["total_usage"] != nil && cpuStats["system_cpu_usage"] != nil && preCpuStats["system_cpu_usage"] != nil {
			deltaTotalUsage := cpuUsage["total_usage"].(int64) - preCpuUsage["total_usage"].(int64)
			deltaSystemUsage := cpuStats["system_cpu_usage"].(float64) - preCpuStats["system_cpu_usage"].(float64)
			event.MetricSetFields.Put("cpu.total.norm.pct", float64(deltaTotalUsage)/deltaSystemUsage*numCores)
		}
	}
	return event
}
