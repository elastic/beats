// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package system

import (
	"encoding/xml"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/PaloAltoNetworks/pango"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/panos"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const (
	metricsetName = "system"
	vsys          = ""
	query         = "<show><system><resources></resources></system></show>"
)

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	config panos.Config
	logger *logp.Logger
	client *pango.Firewall
}

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host is defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet(panos.ModuleName, metricsetName, New)
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The panos licenses metricset is beta.")

	config := panos.Config{}
	logger := logp.NewLogger(base.FullyQualifiedName())

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}
	logger.Debugf("panos_licenses metricset config: %v", config)

	client := &pango.Firewall{Client: pango.Client{Hostname: config.HostIp, ApiKey: config.ApiKey}}

	return &MetricSet{
		BaseMetricSet: base,
		config:        config,
		logger:        logger,
		client:        client,
	}, nil
}

// Fetch method implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	log := m.Logger()
	var response Response

	// Initialize the client
	if err := m.client.Initialize(); err != nil {
		log.Error("Failed to initialize client: %s", err)
		return err
	}
	log.Infof("panos_licenses.Fetch initialized client")

	output, err := m.client.Op(query, vsys, nil, nil)
	if err != nil {
		log.Error("Error: %s", err)
		return err
	}

	err = xml.Unmarshal(output, &response)
	if err != nil {
		log.Error("Error: %s", err)
		return err
	}

	event := getEvent(m, response.Result)
	report.Event(*event)

	return nil
}

func getEvent(m *MetricSet, input string) *mb.Event {
	currentTime := time.Now()

	// The output is standard "top" output, and we only need the top 5 lines
	lines := strings.Split(input, "\n")
	lines = lines[:5]

	systemInfo := parseSystemInfo(lines[0])
	taskInfo := parseTaskInfo(lines[1])
	cpuInfo := parseCPUInfo(lines[2])
	memoryInfo := parseMemoryInfo(lines[3])
	swapInfo := parseSwapInfo(lines[4])

	event := mb.Event{MetricSetFields: mapstr.M{
		"uptime.days":         systemInfo.Uptime.Days,
		"uptime.hours":        systemInfo.Uptime.Hours,
		"user_count":          systemInfo.UserCount,
		"load_average.1m":     systemInfo.LoadAverage.one_minute,
		"load_average.5m":     systemInfo.LoadAverage.five_minute,
		"load_average.15m":    systemInfo.LoadAverage.fifteen_minute,
		"tasks.total":         taskInfo.Total,
		"tasks.running":       taskInfo.Running,
		"tasks.sleeping":      taskInfo.Sleeping,
		"tasks.stopped":       taskInfo.Stopped,
		"tasks.zombie":        taskInfo.Zombie,
		"cpu.user":            cpuInfo.User,
		"cpu.system":          cpuInfo.System,
		"cpu.nice":            cpuInfo.Nice,
		"cpu.idle":            cpuInfo.Idle,
		"cpu.wait":            cpuInfo.Wait,
		"cpu.hi":              cpuInfo.Hi,
		"cpu.system_int":      cpuInfo.SystemInt,
		"cpu.steal":           cpuInfo.Steal,
		"memory.total":        memoryInfo.Total,
		"memory.free":         memoryInfo.Free,
		"memory.used":         memoryInfo.Used,
		"memory.buffer_cache": memoryInfo.BufferCache,
		"swap.total":          swapInfo.Total,
		"swap.free":           swapInfo.Free,
		"swap.used":           swapInfo.Used,
		"swap.available":      swapInfo.Available,
	}}
	event.Timestamp = currentTime
	event.RootFields = mapstr.M{
		"observer.ip":     m.config.HostIp,
		"host.ip":         m.config.HostIp,
		"observer.vendor": "Palo Alto",
		"observer.type":   "firewall",
	}

	return &event
}

func parseSystemInfo(line string) SystemInfo {
	// top - 07:51:37 up 108 days,  1:38,  0 users,  load average: 5.52, 5.79, 5.99
	re := regexp.MustCompile(`\s+`)
	normal := re.ReplaceAllString(line, " ")
	fields := strings.Split(normal, " ")

	uptime := Uptime{parseInt(fields[4]), fields[6]}
	loadAverage := strings.Split(normal, ": ")[1]
	loadAverageValues := strings.Split(loadAverage, ", ")

	users := fields[7]
	var loadAverageFloat []float64
	for _, value := range loadAverageValues {
		loadAverageFloat = append(loadAverageFloat, parseFloat(value))
	}

	SystemLoad := SystemLoad{loadAverageFloat[0], loadAverageFloat[1], loadAverageFloat[2]}
	return SystemInfo{
		Uptime:      uptime,
		UserCount:   parseInt(users),
		LoadAverage: SystemLoad,
	}
}

func parseTaskInfo(line string) TaskInfo {
	//Tasks: 189 total,   7 running, 182 sleeping,   0 stopped,   0 zombie
	values := strings.Fields(line)

	total := parseInt(values[1])
	running := parseInt(values[3])
	sleeping := parseInt(values[5])
	stopped := parseInt(values[7])
	zombie := parseInt(values[9])

	return TaskInfo{
		Total:    total,
		Running:  running,
		Sleeping: sleeping,
		Stopped:  stopped,
		Zombie:   zombie,
	}
}

func parseCPUInfo(line string) CPUInfo {
	//%Cpu(s): 73.0 us,  4.6 sy,  0.0 ni, 21.7 id,  0.0 wa,  0.0 hi,  0.7 si,  0.0 st
	values := strings.Fields(line)

	user := parseFloat(values[1])
	system := parseFloat(values[3])
	nice := parseFloat(values[5])
	idle := parseFloat(values[7])
	wait := parseFloat(values[9])
	hi := parseFloat(values[11])
	systemInt := parseFloat(values[13])
	steal := parseFloat(values[15])

	return CPUInfo{
		User:      user,
		System:    system,
		Nice:      nice,
		Idle:      idle,
		Wait:      wait,
		Hi:        hi,
		SystemInt: systemInt,
		Steal:     steal,
	}
}

func parseMemoryInfo(line string) MemoryInfo {
	//MiB Mem :   5026.9 total,    414.2 free,   2541.5 used,   2071.1 buff/cache
	values := strings.Fields(line)

	total := parseFloat(values[3])
	free := parseFloat(values[5])
	used := parseFloat(values[7])
	bufferCache := parseFloat(values[9])

	return MemoryInfo{
		Total:       total,
		Free:        free,
		Used:        used,
		BufferCache: bufferCache,
	}
}

func parseSwapInfo(line string) SwapInfo {
	//MiB Swap:   5961.0 total,   4403.5 free,   1557.6 used.   1530.0 avail Mem
	values := strings.Fields(line)

	total := parseFloat(values[2])
	free := parseFloat(values[4])
	used := parseFloat(values[6])
	available := parseFloat(values[8])

	return SwapInfo{
		Total:     total,
		Free:      free,
		Used:      used,
		Available: available,
	}
}

func parseFloat(value string) float64 {
	var result float64
	fmt.Sscanf(value, "%f", &result)
	return result
}

func parseInt(value string) int {
	var result int
	fmt.Sscanf(value, "%d", &result)
	return result
}
