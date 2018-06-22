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
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/module/docker"
)

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
}

type MemoryService struct{}

func (s *MemoryService) getMemoryStatsList(rawStats []docker.Stat, dedot bool) []MemoryData {
	formattedStats := []MemoryData{}
	for _, myRawStats := range rawStats {
		formattedStats = append(formattedStats, s.getMemoryStats(myRawStats, dedot))
	}

	return formattedStats
}

func (s *MemoryService) getMemoryStats(myRawStat docker.Stat, dedot bool) MemoryData {
	totalRSS := myRawStat.Stats.MemoryStats.Stats["total_rss"]
	return MemoryData{
		Time:      common.Time(myRawStat.Stats.Read),
		Container: docker.NewContainer(myRawStat.Container, dedot),
		Failcnt:   myRawStat.Stats.MemoryStats.Failcnt,
		Limit:     myRawStat.Stats.MemoryStats.Limit,
		MaxUsage:  myRawStat.Stats.MemoryStats.MaxUsage,
		TotalRss:  totalRSS,
		TotalRssP: float64(totalRSS) / float64(myRawStat.Stats.MemoryStats.Limit),
		Usage:     myRawStat.Stats.MemoryStats.Usage,
		UsageP:    float64(myRawStat.Stats.MemoryStats.Usage) / float64(myRawStat.Stats.MemoryStats.Limit),
	}
}
