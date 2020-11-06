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

	"github.com/elastic/beats/v7/libbeat/common"
	helpers "github.com/elastic/beats/v7/libbeat/common/docker"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/docker"
	"github.com/elastic/beats/v7/metricbeat/module/docker/cpu"
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

// container is a struct representation of a container
type container struct {
	DockerId string
	Name     string
	Image    string
	Labels   map[string]string
}

type cpuStats struct {
	PerCPUUsage                           common.MapStr
	TotalUsage                            float64
	TotalUsageNormalized                  float64
	UsageInKernelmode                     uint64
	UsageInKernelmodePercentage           float64
	UsageInKernelmodePercentageNormalized float64
	UsageInUsermode                       uint64
	UsageInUsermodePercentage             float64
	UsageInUsermodePercentageNormalized   float64
	SystemUsage                           uint64
	SystemUsagePercentage                 float64
	SystemUsagePercentageNormalized       float64
}

type memoryStats struct {
	Failcnt   uint64
	Limit     uint64
	MaxUsage  uint64
	TotalRss  uint64
	TotalRssP float64
	Usage     uint64
	UsageP    float64
	//Raw stats from the cgroup subsystem
	Stats map[string]uint64
	//Windows-only memory stats
	Commit            uint64
	CommitPeak        uint64
	PrivateWorkingSet uint64
}

type networkStats struct {
	NameInterface string
	Total         types.NetworkStats
}

// BlkioRaw sums raw Blkio stats
type BlkioRaw struct {
	reads  uint64
	writes uint64
	totals uint64
}

type blkioStats struct {
	reads  float64
	writes float64
	totals float64

	serviced      BlkioRaw
	servicedBytes BlkioRaw
	servicedTime  BlkioRaw
	waitTime      BlkioRaw
	queued        BlkioRaw
}

// Stats is a struct represents information regarding a container
type Stats struct {
	Time         common.Time
	Container    *container
	cpuStats     cpuStats
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

// ContainerMetadata is an struct represents container metadata
type ContainerMetadata struct {
	Cluster   string
	TaskARN   string
	Family    string
	Revision  string
	Container *container
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	logger := logp.NewLogger(metricsetName)
	metricSet, err := awsfargate.NewMetricSet(base)
	if err != nil {
		return nil, fmt.Errorf("error creating %s metricset: %w", metricsetName, err)
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

	taskStatsEndpoint := fmt.Sprintf("%s/task/stats", ecsURI)
	taskEndpoint := fmt.Sprintf("%s/task", ecsURI)
	formattedStats, err := queryTaskMetadataEndpoints(taskStatsEndpoint, taskEndpoint)
	if err != nil {
		err := fmt.Errorf("queryTaskMetadataEndpoints failed: %w", err)
		m.logger.Error(err)
		return err
	}

	eventsMapping(report, formattedStats)
	return nil
}

func queryTaskMetadataEndpoints(taskStatsEndpoint string, taskEndpoint string) ([]Stats, error) {
	// Get response from ${ECS_CONTAINER_METADATA_URI_V4}/task/stats
	taskStatsResp, err := http.Get(taskStatsEndpoint)
	if err != nil {
		return nil, fmt.Errorf("http.Get failed: %w", err)
	}
	taskStatsOutput, err := getTaskStats(taskStatsResp)
	if err != nil {
		return nil, fmt.Errorf("getTaskStats failed: %w", err)
	}

	// Collect container metadata information from ${ECS_CONTAINER_METADATA_URI_V4}/task
	taskResp, err := http.Get(taskEndpoint)
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
			formattedStats = append(formattedStats, getStats(taskStats, cInfo.Container))
		}
	}
	return formattedStats
}

func getStats(taskStats types.StatsJSON, c *container) Stats {
	usage := cpu.CPUUsage{Stat: &docker.Stat{Stats: taskStats}}
	cpuStats := cpuStats{
		TotalUsage:                            usage.Total(),
		TotalUsageNormalized:                  usage.TotalNormalized(),
		UsageInKernelmode:                     taskStats.Stats.CPUStats.CPUUsage.UsageInKernelmode,
		UsageInKernelmodePercentage:           usage.InKernelMode(),
		UsageInKernelmodePercentageNormalized: usage.InKernelModeNormalized(),
		UsageInUsermode:                       taskStats.Stats.CPUStats.CPUUsage.UsageInUsermode,
		UsageInUsermodePercentage:             usage.InUserMode(),
		UsageInUsermodePercentageNormalized:   usage.InUserModeNormalized(),
		SystemUsage:                           taskStats.Stats.CPUStats.SystemUsage,
		SystemUsagePercentage:                 usage.System(),
		SystemUsagePercentageNormalized:       usage.SystemNormalized(),
	}

	totalRSS := taskStats.Stats.MemoryStats.Stats["total_rss"]
	memoryStats := memoryStats{
		TotalRss:  totalRSS,
		MaxUsage:  taskStats.Stats.MemoryStats.MaxUsage,
		TotalRssP: float64(totalRSS) / float64(taskStats.Stats.MemoryStats.Limit),
		Usage:     taskStats.Stats.MemoryStats.Usage,
		UsageP:    float64(taskStats.Stats.MemoryStats.Usage) / float64(taskStats.Stats.MemoryStats.Limit),
		Stats:     taskStats.Stats.MemoryStats.Stats,
		//Windows memory statistics
		Commit:            taskStats.Stats.MemoryStats.Commit,
		CommitPeak:        taskStats.Stats.MemoryStats.CommitPeak,
		PrivateWorkingSet: taskStats.Stats.MemoryStats.PrivateWorkingSet,
	}

	stats := Stats{
		Time:        common.Time(taskStats.Stats.Read),
		cpuStats:    cpuStats,
		memoryStats: memoryStats,
	}

	stats.Container = &container{
		DockerId: c.DockerId,
		Image:    c.Image,
		Name:     helpers.ExtractContainerName([]string{c.Name}),
		Labels:   deDotLabels(c.Labels),
	}

	var networks []networkStats
	for nameInterface, rawNetStats := range taskStats.Networks {
		networks = append(networks, networkStats{
			NameInterface: nameInterface,
			Total:         rawNetStats,
		})
	}
	stats.networkStats = networks

	blkioStats := getBlkioStats(taskStats.BlkioStats)
	stats.blkioStats = blkioStats
	return stats
}

// deDotLabels returns a new map[string]string containing a copy of the labels
// where the dots have been converted into nested structure, avoiding possible
// mapping errors
func deDotLabels(labels map[string]string) map[string]string {
	outputLabels := map[string]string{}
	for k, v := range labels {
		// This is necessary so that ES does not interpret '.' fields as new
		// nested JSON objects, and also makes this compatible with ES 2.x.
		label := common.DeDot(k)
		outputLabels[label] = v
	}

	return outputLabels
}

// getBlkioStats collects diskio metrics from BlkioStats structures, that
// are not populated in Windows
func getBlkioStats(raw types.BlkioStats) blkioStats {
	return blkioStats{
		serviced:      getNewStats(raw.IoServicedRecursive),
		servicedBytes: getNewStats(raw.IoServiceBytesRecursive),
		servicedTime:  getNewStats(raw.IoServiceTimeRecursive),
		waitTime:      getNewStats(raw.IoWaitTimeRecursive),
		queued:        getNewStats(raw.IoQueuedRecursive),
	}
}

func getNewStats(blkioEntry []types.BlkioStatEntry) BlkioRaw {
	stats := BlkioRaw{
		reads:  0,
		writes: 0,
		totals: 0,
	}

	for _, myEntry := range blkioEntry {
		switch myEntry.Op {
		case "Write":
			stats.writes += myEntry.Value
		case "Read":
			stats.reads += myEntry.Value
		case "Total":
			stats.totals += myEntry.Value
		}
	}
	return stats
}
