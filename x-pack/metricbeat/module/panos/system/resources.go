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

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func getResourceEvents(m *MetricSet) ([]mb.Event, error) {
	var response ResourceResponse
	query := "<show><system><resources></resources></system></show>"

	output, err := m.client.Op(query, vsys, nil, nil)
	if err != nil {
		m.logger.Error("Error: %s", err)
		return nil, err
	}

	err = xml.Unmarshal(output, &response)
	if err != nil {
		m.logger.Error("Error: %s", err)
		return nil, err
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
	currentTime := time.Now()
	events := make([]mb.Event, 0, 1)

	// We only need the top 5 lines
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
	},
		RootFields: mapstr.M{
			"observer.ip":     m.config.HostIp,
			"host.ip":         m.config.HostIp,
			"observer.vendor": "Palo Alto",
			"observer.type":   "firewall",
			"@Timestamp":      currentTime,
		}}

	events = append(events, event)
	return events
}

func convertUptime(uptime string) (int, int) {
	// 07:51
	hourstr := strings.Split(uptime, ":")
	hours := parseInt(hourstr[0])
	minutes := parseInt(hourstr[1])

	return hours, minutes
}

func parseSystemInfo(line string) SystemInfo {
	// top - 07:51:37 up 108 days,  1:38,  0 users,  load average: 5.52, 5.79, 5.99
	re := regexp.MustCompile(`\s+`)
	normal := re.ReplaceAllString(line, " ")
	fields := strings.Split(normal, " ")
	upHours, upMinutes := convertUptime(fields[6])

	uptime := Uptime{parseInt(fields[4]), upHours, upMinutes}
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
