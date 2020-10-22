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

	for queryID, taskStats := range output {
		event := m.createEvent(queryID, taskStats)
		// report events
		if reported := report.Event(event); !reported {
			m.logger.Debug("Fetch interrupted, failed to emit event")
			return nil
		}
	}
	return nil
}

func (m *MetricSet) createEvent(queryID string, taskStats interface{}) mb.Event {
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

	// name and id
	event.MetricSetFields.Put("name", taskMeta["name"])
	event.MetricSetFields.Put("id", taskMeta["id"])

	// cpu
	resultCPUStats, err := schemaCPUStats.Apply(taskMeta["cpu_stats"].(map[string]interface{}), s.FailOnRequired)
	event.MetricSetFields.Put("cpu_stats", resultCPUStats)

	// memory
	resultMemoryStats, err := schemaMemoryStats.Apply(taskMeta["memory_stats"].(map[string]interface{}), s.FailOnRequired)
	event.MetricSetFields.Put("memory_stats", resultMemoryStats)

	// network
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
	return event
}
