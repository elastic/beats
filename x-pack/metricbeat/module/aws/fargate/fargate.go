// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fargate

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/aws"
)

var (
	metricsetName  = "fargate"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet(aws.ModuleName, metricsetName, New,
		mb.DefaultMetricSet(),
	)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	*aws.MetricSet
	logger             *logp.Logger
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	logger := logp.NewLogger(metricsetName)
	metricSet, err := aws.NewMetricSet(base)
	if err != nil {
		return nil, fmt.Errorf("error creating aws metricset: %w", err)
	}

	return &MetricSet{
		MetricSet:          metricSet,
		logger:             logger,
	}, nil
}

type CPUUsage struct {
	TotalUsage                            float64 `json:"total_usage"`
	PerCPUUsage  common.MapStr `json:"percpu_usage"`
	UsageInKernelmode                     uint64 `json:"usage_in_kernelmode"`
	UsageInUsermode                       uint64 `json:"usage_in_usermode"`
}

type CPUStats struct {
	CPUUsage CPUUsage `json:"cpu_usage"`
	//PerCPUUsage                           common.MapStr
	//TotalUsage                            float64
	//TotalUsageNormalized                  float64
	//UsageInKernelmode                     uint64
	//UsageInKernelmodePercentage           float64
	//UsageInKernelmodePercentageNormalized float64
	//UsageInUsermode                       uint64
	//UsageInUsermodePercentage             float64
	//UsageInUsermodePercentageNormalized   float64
	//SystemUsage                           uint64
	//SystemUsagePercentage                 float64
	//SystemUsagePercentageNormalized       float64
}

type TaskMeta struct {
	ReadTimestamp string `json:"read"`
	PreReadTimestamp string `json:"preread"`
	CPUStats CPUStats `json:"cpu_stats"`
}
// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	var taskStatsOutputs map[string]interface{}

	// Get response from ${ECS_CONTAINER_METADATA_URI_V4}/task/stats
	resp, err := http.Get("${ECS_CONTAINER_METADATA_URI_V4}/task/stats")
	if err != nil {
		m.Logger().Error(err.Error())
		return err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		m.Logger().Error(err.Error())
		return err
	}

	err = json.Unmarshal(body, &taskStatsOutputs)
	if err != nil {
		m.Logger().Error(err.Error())
		return err
	}

	var events []mb.Event
	for queryID, taskStats := range taskStatsOutputs {
		taskMeta := taskStats.(TaskMeta)
		m.Logger().Info("queryID = ", queryID)
		m.Logger().Info("taskMeta.CPUStats = ", taskMeta.CPUStats)
		m.Logger().Info("taskMeta.ReadTimestamp = ", taskMeta.ReadTimestamp)
		fmt.Println("=========================")
		fmt.Println("queryID = ", queryID)
		fmt.Println("ReadTimestamp = ", taskMeta.ReadTimestamp)
		fmt.Println("CPUStats = ", taskMeta.CPUStats)

		event := mb.Event{}
		event.MetricSetFields.Put("cpu_usage", taskMeta.CPUStats.CPUUsage)
		events = append(events, event)
	}

	// report events
	for _, event := range events {
		if reported := report.Event(event); !reported {
			m.Logger().Debug("Fetch interrupted, failed to emit event")
			return nil
		}
	}
	return nil
}
