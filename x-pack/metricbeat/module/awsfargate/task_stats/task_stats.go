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
	"github.com/pkg/errors"

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

// BlkioRaw sums raw Blkio stats
type BlkioRaw struct {
	reads  uint64
	writes uint64
	totals uint64
}

// Stats is a struct represents information regarding a container
type Stats struct {
	Time      common.Time
	Container *container

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

	NameInterface string
	RxBytes       float64
	RxDropped     float64
	RxErrors      float64
	RxPackets     float64
	TxBytes       float64
	TxDropped     float64
	TxErrors      float64
	TxPackets     float64
	Total         *types.NetworkStats

	reads  float64
	writes float64
	totals float64

	serviced      BlkioRaw
	servicedBytes BlkioRaw
	servicedTime  BlkioRaw
	waitTime      BlkioRaw
	queued        BlkioRaw
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

	taskStatsEndpoint := fmt.Sprintf("%s/task/stats", ecsURI)
	taskEndpoint := fmt.Sprintf("%s/task", ecsURI)
	formattedStats, err := queryTaskMetadataEndpoints(taskStatsEndpoint, taskEndpoint)
	if err != nil {
		err := fmt.Errorf("queryTaskMetadataEndpoints failed")
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

	formattedStats := getCPUStatsList(taskStatsOutput, taskOutput)
	return formattedStats, nil
}

func getTaskStats(taskStatsResp *http.Response) (map[string]types.Stats, error) {
	taskStatsBody, err := ioutil.ReadAll(taskStatsResp.Body)
	if err != nil {
		return nil, fmt.Errorf("ioutil.ReadAll failed: %w", err)
	}

	var taskStatsOutput map[string]types.Stats
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

func getCPUStatsList(taskStatsOutput map[string]types.Stats, taskOutput TaskMetadata) []Stats {
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
		var dockerStat docker.Stat
		dockerStat.Stats = types.StatsJSON{
			Stats: taskStats,
			ID:    id,
		}

		if cInfo, ok := containersInfo[id]; ok {
			formattedStats = append(formattedStats, getCPUStats(&dockerStat, cInfo.Container))
		}
	}
	return formattedStats
}

func getCPUStats(myRawStat *docker.Stat, c *container) Stats {
	usage := cpu.CPUUsage{Stat: myRawStat}

	stats := Stats{
		Time:                                  common.Time(myRawStat.Stats.Read),
		TotalUsage:                            usage.Total(),
		TotalUsageNormalized:                  usage.TotalNormalized(),
		UsageInKernelmode:                     myRawStat.Stats.CPUStats.CPUUsage.UsageInKernelmode,
		UsageInKernelmodePercentage:           usage.InKernelMode(),
		UsageInKernelmodePercentageNormalized: usage.InKernelModeNormalized(),
		UsageInUsermode:                       myRawStat.Stats.CPUStats.CPUUsage.UsageInUsermode,
		UsageInUsermodePercentage:             usage.InUserMode(),
		UsageInUsermodePercentageNormalized:   usage.InUserModeNormalized(),
		SystemUsage:                           myRawStat.Stats.CPUStats.SystemUsage,
		SystemUsagePercentage:                 usage.System(),
		SystemUsagePercentageNormalized:       usage.SystemNormalized(),
	}

	stats.Container = &container{
		DockerId: c.DockerId,
		Image:    c.Image,
		Name:     helpers.ExtractContainerName([]string{c.Name}),
		Labels:   deDotLabels(c.Labels),
	}
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
