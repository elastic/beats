// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package memory

import (
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/metricbeat/module/docker"
)

// MemoryData contains parsed container memory info
type MemoryData struct {
	Time      common.Time
	Container *docker.Container
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

// MemoryService is placeholder for the the memory stat parsers
type MemoryService struct{}

func (s *MemoryService) getMemoryStatsList(containers []docker.Stat, dedot bool) []MemoryData {
	formattedStats := []MemoryData{}
	for _, containerStats := range containers {
		//There appears to be a race where a container will report with a stat object before it actually starts
		//during this time, there doesn't appear to be any meaningful data,
		// and Limit will never be 0 unless the container is not running
		//and there's no cgroup data, and CPU usage should be greater than 0 for any running container.
		if containerStats.Stats.MemoryStats.Limit == 0 || containerStats.Stats.PreCPUStats.CPUUsage.TotalUsage == 0 {
			continue
		}
		formattedStats = append(formattedStats, s.getMemoryStats(containerStats, dedot))
	}

	return formattedStats
}

func (s *MemoryService) getMemoryStats(myRawStat docker.Stat, dedot bool) MemoryData {
	totalRSS := myRawStat.Stats.MemoryStats.Stats["total_rss"]

	// Emulate newer docker releases and exclude cache values from memory usage
	// See here for a little more context. usage - cache won't work, as it includes shared mappings that can't be dropped
	// like regular cache.
	// This formula is used by both cadvisor and containerd
	// https://github.com/google/cadvisor/commit/307d1b1cb320fef66fab02db749f07a459245451
	var fileUsage, memUsage uint64
	// use this a shortcut to see if the underlying system is V1 or V2
	totalInactive, isV1 := myRawStat.Stats.MemoryStats.Stats["total_inactive_file"]
	if isV1 && totalInactive < myRawStat.Stats.MemoryStats.Usage {
		fileUsage = totalInactive
	}
	if fileInactive, ok := myRawStat.Stats.MemoryStats.Stats["inactive_file"]; !isV1 && ok && fileInactive < myRawStat.Stats.MemoryStats.Usage {
		fileUsage = fileInactive
	}

	memUsage = myRawStat.Stats.MemoryStats.Usage - fileUsage
	return MemoryData{
		Time:      common.Time(myRawStat.Stats.Read),
		Container: docker.NewContainer(myRawStat.Container, dedot),
		Failcnt:   myRawStat.Stats.MemoryStats.Failcnt,
		Limit:     myRawStat.Stats.MemoryStats.Limit,
		MaxUsage:  myRawStat.Stats.MemoryStats.MaxUsage,
		TotalRss:  totalRSS,
		TotalRssP: float64(totalRSS) / float64(myRawStat.Stats.MemoryStats.Limit),
		Usage:     memUsage,
		UsageP:    float64(memUsage) / float64(myRawStat.Stats.MemoryStats.Limit),
		Stats:     myRawStat.Stats.MemoryStats.Stats,
		//Windows memory statistics
		Commit:            myRawStat.Stats.MemoryStats.Commit,
		CommitPeak:        myRawStat.Stats.MemoryStats.CommitPeak,
		PrivateWorkingSet: myRawStat.Stats.MemoryStats.PrivateWorkingSet,
	}
}
