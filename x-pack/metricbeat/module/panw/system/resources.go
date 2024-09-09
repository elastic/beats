// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package system

import (
	"encoding/xml"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const resourceQuery = "<show><system><resources></resources></system></show>"

var logger *logp.Logger

func getResourceEvents(m *MetricSet) ([]mb.Event, error) {
	// Set logger so all the parse functions have access
	logger = m.logger

	var response ResourceResponse
	output, err := m.client.Op(resourceQuery, vsys, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to execute operation: %w", err)
	}

	err = xml.Unmarshal(output, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal XML response: %w", err)
	}

	events := formatResourceEvents(m, response.Result)

	return events, nil
}

/*
Output from the XML API call is the standard "top" output:

top - 07:51:37 up 108 days,  1:38,  0 users,  load average: 5.52, 5.79, 5.99
Tasks: 189 total,   7 running, 182 sleeping,   0 stopped,   0 zombie
%Cpu(s): 73.0 us,  4.6 sy,  0.0 ni, 21.7 id,  0.0 wa,  0.0 hi,  0.7 si,  0.0 st
MiB Mem :   5026.9 total,    414.2 free,   2541.5 used,   2071.1 buff/cache
MiB Swap:   5961.0 total,   4403.5 free,   1557.6 used.   1530.0 avail Mem

	 PID USER      PR  NI    VIRT    RES    SHR S  %CPU  %MEM     TIME+ COMMAND
	5692       20   0  121504   8396   6644 R  94.4   0.2 155491:08 pan_task
	5695       20   0  121504   7840   6632 R  94.4   0.2 155491:01 pan_task
	5696       20   0  121504   8360   6816 R  94.4   0.2 155486:29 pan_task
	5699       20   0  121504   8132   6676 R  94.4   0.2 155236:17 pan_task
	5700       20   0  146304  18424   6780 R  88.9   0.4 155491:40 pan_task

22360 nobody    20   0  459836  40592  10148 R  22.2   0.8   0:38.65 httpd

	6374       17  -3 1078156  18272   9716 S   5.6   0.4 215:46.41 routed

14227       20   0   18108   7184   2172 R   5.6   0.1   0:00.04 top

	1       20   0    2532    696    656 S   0.0   0.0   3:48.14 init
	2       20   0       0      0      0 S   0.0   0.0   0:00.83 kthreadd
*/
func formatResourceEvents(m *MetricSet, input string) []mb.Event {
	timestamp := time.Now()
	events := make([]mb.Event, 0)

	// We only need the top 5 lines
	lines := strings.Split(input, "\n")
	lines = lines[:5]

	systemInfo := parseSystemInfo(lines[0])
	taskInfo := parseTaskInfo(lines[1])
	cpuInfo := parseCPUInfo(lines[2])
	memoryInfo := parseMemoryInfo(lines[3])
	swapInfo := parseSwapInfo(lines[4])

	event := mb.Event{
		Timestamp: timestamp,
		MetricSetFields: mapstr.M{
			"uptime": mapstr.M{
				"days":    systemInfo.Uptime.Days,
				"hours":   systemInfo.Uptime.Hours,
				"minutes": systemInfo.Uptime.Minutes,
			},
			"user_count": systemInfo.UserCount,
			"load_average": mapstr.M{
				"1m":  systemInfo.LoadAverage.OneMinute,
				"5m":  systemInfo.LoadAverage.FiveMinute,
				"15m": systemInfo.LoadAverage.FifteenMinute,
			},
			"tasks": mapstr.M{
				"total":    taskInfo.Total,
				"running":  taskInfo.Running,
				"sleeping": taskInfo.Sleeping,
				"stopped":  taskInfo.Stopped,
				"zombie":   taskInfo.Zombie,
			},
			"cpu": mapstr.M{
				"user":       cpuInfo.User,
				"system":     cpuInfo.System,
				"nice":       cpuInfo.Nice,
				"idle":       cpuInfo.Idle,
				"wait":       cpuInfo.Wait,
				"hi":         cpuInfo.Hi,
				"system_int": cpuInfo.SystemInt,
				"steal":      cpuInfo.Steal,
			},
			"memory": mapstr.M{
				"total":        memoryInfo.Total,
				"free":         memoryInfo.Free,
				"used":         memoryInfo.Used,
				"buffer_cache": memoryInfo.BufferCache,
			},
			"swap": mapstr.M{
				"total":     swapInfo.Total,
				"free":      swapInfo.Free,
				"used":      swapInfo.Used,
				"available": swapInfo.Available,
			},
		},
		RootFields: mapstr.M{
			"observer.ip":     m.config.HostIp,
			"host.ip":         m.config.HostIp,
			"observer.vendor": "Palo Alto",
			"observer.type":   "firewall",
		},
	}

	events = append(events, event)
	return events
}

func parseLoadAverage(line string) SystemLoad {
	reLoadAvg := regexp.MustCompile(`load average:\s+([\d.]+),\s+([\d.]+),\s+([\d.]+)`)
	var load1, load5, load15 float64

	if matches := reLoadAvg.FindStringSubmatch(line); matches != nil {
		load1 = parseFloat("load1", matches[1])
		load5 = parseFloat("load5", matches[2])
		load15 = parseFloat("load15", matches[3])
	}

	return SystemLoad{load1, load5, load15}
}

func parseUserCount(line string) int {
	reUserCount := regexp.MustCompile(`(\d+)\s+user`)
	var userCount int
	if matches := reUserCount.FindStringSubmatch(line); matches != nil {
		userCount = parseInt("userCount", matches[1])
	}

	return userCount
}

func parseUptime(line string) Uptime {
	// Uptime less than 1 hour
	// 	top - 15:03:02 up 4 min,  1 user,  load average: 1.77, 2.74, 1.34
	// Uptime less than 1 day
	// 	top - 16:16:29 up  1:18,  1 user,  load average: 0.00, 0.02, 0.01
	// Uptime 1 day or more
	// 	top - 11:08:26 up 2 days, 23:02,  1 user,  load average: 0.40, 0.23, 0.37

	// Regular expressions to match different uptime formats
	// up < 1 hour
	reMin := regexp.MustCompile(`up\s+(\d+)\s+min`)
	// up >= 1 hour < 1 day
	reHourMin := regexp.MustCompile(`up\s+(\d+):(\d+)`)
	// up >= 1 day
	reDayHourMin := regexp.MustCompile(`up\s+(\d+)\s+days?,\s+(\d+):(\d+)`)

	var days, hours, minutes int
	var matches []string

	if matches = reMin.FindStringSubmatch(line); matches != nil {
		minutes = parseInt("minutes", matches[1])
	} else if matches = reHourMin.FindStringSubmatch(line); matches != nil {
		hours = parseInt("hours", matches[1])
		minutes = parseInt("minutes", matches[2])
	} else if matches = reDayHourMin.FindStringSubmatch(line); matches != nil {
		days = parseInt("days", matches[1])
		hours = parseInt("hours", matches[2])
		minutes = parseInt("minutes", matches[3])
	}

	if matches == nil {
		logger.Errorf("Failed to parse uptime: %s", line)
		return Uptime{}
	}

	return Uptime{days, hours, minutes}
}

func parseSystemInfo(line string) SystemInfo {

	uptime := parseUptime(line)
	users := parseUserCount(line)
	SystemLoad := parseLoadAverage(line)

	return SystemInfo{
		Uptime:      uptime,
		UserCount:   users,
		LoadAverage: SystemLoad,
	}
}

func parseTaskInfo(line string) TaskInfo {
	//Tasks: 189 total,   7 running, 182 sleeping,   0 stopped,   0 zombie

	re := regexp.MustCompile(`Tasks:\s*(\d+)\s*total,\s*(\d+)\s*running,\s*(\d+)\s*sleeping,\s*(\d+)\s*stopped,\s*(\d+)\s*zombie`)
	matches := re.FindStringSubmatch(line)
	if matches == nil {
		logger.Errorf("Failed to parse task info: %s", line)
		return TaskInfo{}
	}

	total := parseInt("total", matches[1])
	running := parseInt("running", matches[2])
	sleeping := parseInt("sleeping", matches[3])
	stopped := parseInt("stopped", matches[4])
	zombie := parseInt("zombie", matches[5])

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
	re := regexp.MustCompile(`(\d+\.\d+) us, (\d+\.\d+) sy, (\d+\.\d+) ni, (\d+\.\d+) id, (\d+\.\d+) wa, (\d+\.\d+) hi, (\d+\.\d+) si, (\d+\.\d+) st`)
	matches := re.FindStringSubmatch(line)
	if matches == nil {
		logger.Errorf("Failed to parse CPU info: %s", line)
		return CPUInfo{}
	}

	user := parseFloat("user", matches[1])
	system := parseFloat("system", matches[2])
	nice := parseFloat("nice", matches[3])
	idle := parseFloat("idle", matches[4])
	wait := parseFloat("wait", matches[5])
	hi := parseFloat("hi", matches[6])
	sysint := parseFloat("sysint", matches[7])
	steal := parseFloat("steal", matches[8])

	return CPUInfo{
		User:      user,
		System:    system,
		Nice:      nice,
		Idle:      idle,
		Wait:      wait,
		Hi:        hi,
		SystemInt: sysint,
		Steal:     steal,
	}
}

func parseMemoryInfo(line string) MemoryInfo {
	//MiB Mem :   5026.9 total,    414.2 free,   2541.5 used,   2071.1 buff/cache
	re := regexp.MustCompile(`(\d+\.\d+)\s+total,\s+(\d+\.\d+)\s+free,\s+(\d+\.\d+)\s+used,\s+(\d+\.\d+)\s+buff/cache`)
	matches := re.FindStringSubmatch(line)

	total := parseFloat("total", matches[1])
	free := parseFloat("free", matches[2])
	used := parseFloat("used", matches[3])
	bufferCache := parseFloat("bufferCache", matches[4])

	return MemoryInfo{
		Total:       total,
		Free:        free,
		Used:        used,
		BufferCache: bufferCache,
	}
}

func parseSwapInfo(line string) SwapInfo {
	//MiB Swap:   5961.0 total,   4403.5 free,   1557.6 used.   1530.0 avail Mem
	// Note: the punctuation after the "used" is a ".", not a ","
	re := regexp.MustCompile(`(\d+\.\d+)\s+total,\s+(\d+\.\d+)\s+free,\s+(\d+\.\d+)\s+used[,\.].\s+(\d+\.\d+)\s+avail Mem`)
	matches := re.FindStringSubmatch(line)

	total := parseFloat("total", matches[1])
	free := parseFloat("free", matches[2])
	used := parseFloat("used", matches[3])
	available := parseFloat("available", matches[4])

	return SwapInfo{
		Total:     total,
		Free:      free,
		Used:      used,
		Available: available,
	}
}

func parseFloat(field string, value string) float64 {
	var result float64
	result, err := strconv.ParseFloat(value, 64)
	if err != nil {
		logger.Errorf("parseFloat failed to parse field %s: %v", field, err)
		return -1
	}

	return result
}

func parseInt(field string, value string) int {
	result, err := strconv.Atoi(value)
	if err != nil {
		logger.Errorf("parseInt failed to parse field %s: %v", field, err)
		return -1
	}
	return result
}
