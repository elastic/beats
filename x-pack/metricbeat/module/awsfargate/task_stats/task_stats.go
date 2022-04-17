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

	"github.com/docker/docker/api/types"

	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/logp"
	"github.com/menderesk/beats/v7/metricbeat/mb"
	"github.com/menderesk/beats/v7/metricbeat/module/docker/cpu"
	"github.com/menderesk/beats/v7/x-pack/metricbeat/module/awsfargate"
)

var (
	metricsetName = "task_stats"
	taskStatsPath = "task/stats"
	taskPath      = "task"
)

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
	logger            *logp.Logger
	taskStatsEndpoint string
	taskEndpoint      string
}

// Stats is a struct represents information regarding a container
type Stats struct {
	Time         common.Time
	Container    *container
	cpuStats     cpu.CPUStats
	memoryStats  memoryStats
	networkStats []networkStats
	blkioStats   blkioStats
}

// TaskMetadata is an struct represents response body from ${ECS_CONTAINER_METADATA_URI_V4}/task
type TaskMetadata struct {
	Cluster    string       `json:"Cluster"`
	TaskARN    string       `json:"TaskARN"`
	Family     string       `json:"Family"`
	Revision   string       `json:"Revision"`
	Containers []*container `json:"Containers"`
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	logger := logp.NewLogger(metricsetName)
	metricSet, err := awsfargate.NewMetricSet(base)
	if err != nil {
		return nil, fmt.Errorf("error creating %s metricset: %w", metricsetName, err)
	}

	ecsURI, ok := os.LookupEnv("ECS_CONTAINER_METADATA_URI_V4")
	if !ok {
		return nil, fmt.Errorf("lookup $ECS_CONTAINER_METADATA_URI_V4 failed")
	}

	return &MetricSet{
		MetricSet:         metricSet,
		logger:            logger,
		taskStatsEndpoint: fmt.Sprintf("%s/%s", ecsURI, taskStatsPath),
		taskEndpoint:      fmt.Sprintf("%s/%s", ecsURI, taskPath),
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	formattedStats, err := m.queryTaskMetadataEndpoints()
	if err != nil {
		err := fmt.Errorf("queryTaskMetadataEndpoints failed: %w", err)
		m.logger.Error(err)
		return err
	}

	eventsMapping(report, formattedStats)
	return nil
}

func (m *MetricSet) queryTaskMetadataEndpoints() ([]Stats, error) {
	// Get response from ${ECS_CONTAINER_METADATA_URI_V4}/task/stats
	taskStatsResp, err := http.Get(m.taskStatsEndpoint)
	if err != nil {
		return nil, fmt.Errorf("http.Get failed: %w", err)
	}
	taskStatsOutput, err := getTaskStats(taskStatsResp)
	if err != nil {
		return nil, fmt.Errorf("getTaskStats failed: %w", err)
	}

	// Collect container metadata information from ${ECS_CONTAINER_METADATA_URI_V4}/task
	taskResp, err := http.Get(m.taskEndpoint)
	if err != nil {
		return nil, fmt.Errorf("http.Get failed: %w", err)
	}
	taskOutput, err := getTask(taskResp)
	if err != nil {
		return nil, fmt.Errorf("getTask failed: %w", err)
	}

	formattedStats := getStatsList(taskStatsOutput, taskOutput)
	return formattedStats, nil
}

func getTaskStats(taskStatsResp *http.Response) (map[string]types.StatsJSON, error) {
	taskStatsBody, err := ioutil.ReadAll(taskStatsResp.Body)
	if err != nil {
		return nil, fmt.Errorf("ioutil.ReadAll failed: %w", err)
	}

	var taskStatsOutput map[string]types.StatsJSON
	err = json.Unmarshal(taskStatsBody, &taskStatsOutput)
	if err != nil {
		return nil, fmt.Errorf("json.Unmarshal failed: %w", err)
	}
	return taskStatsOutput, nil
}

func getTask(taskResp *http.Response) (TaskMetadata, error) {
	taskBody, err := ioutil.ReadAll(taskResp.Body)
	if err != nil {
		return TaskMetadata{}, fmt.Errorf("ioutil.ReadAll failed: %w", err)
	}

	var taskOutput TaskMetadata
	err = json.Unmarshal(taskBody, &taskOutput)
	if err != nil {
		return TaskMetadata{}, fmt.Errorf("json.Unmarshal failed: %w", err)
	}
	return taskOutput, nil
}

func getStatsList(taskStatsOutput map[string]types.StatsJSON, taskOutput TaskMetadata) []Stats {
	containersInfo := map[string]ContainerMetadata{}
	for _, c := range taskOutput.Containers {
		// Skip ~internal~ecs~pause container
		if c.Name == "~internal~ecs~pause" {
			continue
		}

		containerMetadata := ContainerMetadata{
			Container: c,
			Family:    taskOutput.Family,
			TaskARN:   taskOutput.TaskARN,
			Cluster:   taskOutput.Cluster,
			Revision:  taskOutput.Revision,
		}
		containersInfo[c.DockerId] = containerMetadata
	}

	var formattedStats []Stats
	for id, taskStats := range taskStatsOutput {
		if cInfo, ok := containersInfo[id]; ok {
			statsPerContainer := Stats{
				Time:         common.Time(taskStats.Stats.Read),
				Container:    getContainerStats(cInfo.Container),
				cpuStats:     getCPUStats(taskStats),
				memoryStats:  getMemoryStats(taskStats),
				networkStats: getNetworkStats(taskStats),
				blkioStats:   getBlkioStats(taskStats.BlkioStats),
			}

			formattedStats = append(formattedStats, statsPerContainer)
		}
	}
	return formattedStats
}
