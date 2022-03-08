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

package diskio

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/metricbeat/module/docker"
)

var blkioService BlkioService
var oldBlkioRaw = make([]BlkioRaw, 3)
var newBlkioRaw = make([]BlkioRaw, 3)

func TestDeltaMultipleContainers(t *testing.T) {
	var apiContainer1 docker.Stat
	var apiContainer2 docker.Stat
	metrics := types.BlkioStatEntry{
		Major: 123,
		Minor: 123,
		Op:    "Total",
		Value: 123,
	}
	jsonContainers := `[
     {
             "Id": "8dfafdbc3a40",
			 "Names": ["container"]
     },{
             "Id": "8dfafdbc3a41",
			 "Names": ["container1"]
     }]`
	var containers []types.Container
	err := json.Unmarshal([]byte(jsonContainers), &containers)
	if err != nil {
		t.Fatal(err)
	}

	apiContainer1.Stats.Read = time.Now()
	apiContainer1.Container = &containers[0]
	apiContainer1.Stats.BlkioStats.IoServicedRecursive = append(apiContainer1.Stats.BlkioStats.IoServicedRecursive, metrics)
	apiContainer2.Stats.Read = time.Now()
	apiContainer2.Container = &containers[1]
	apiContainer2.Stats.BlkioStats.IoServicedRecursive = append(apiContainer2.Stats.BlkioStats.IoServicedRecursive, metrics)
	dockerStats := []docker.Stat{apiContainer1, apiContainer2}
	stats := blkioService.getBlkioStatsList(dockerStats, true, []uint64{})
	totals := make([]float64, 2)
	for _, stat := range stats {
		totals[0] = stat.totals
	}

	dockerStats[0].Stats.BlkioStats.IoServicedRecursive[0].Value = 1000
	dockerStats[0].Stats.Read = dockerStats[0].Stats.Read.Add(time.Second * 10)
	dockerStats[1].Stats.BlkioStats.IoServicedRecursive[0].Value = 1000
	dockerStats[1].Stats.Read = dockerStats[0].Stats.Read.Add(time.Second * 10)
	stats = blkioService.getBlkioStatsList(dockerStats, true, []uint64{})
	for _, stat := range stats {
		totals[1] = stat.totals
		if stat.totals < totals[0] {
			t.Errorf("getBlkioStatsList(%v) => %v, want value bigger than %v", dockerStats, stat.totals, totals[0])
		}
	}

	dockerStats[0].Stats.Read = dockerStats[0].Stats.Read.Add(time.Second * 15)
	dockerStats[0].Stats.BlkioStats.IoServicedRecursive[0].Value = 2000
	dockerStats[1].Stats.BlkioStats.IoServicedRecursive[0].Value = 2000
	dockerStats[1].Stats.Read = dockerStats[0].Stats.Read.Add(time.Second * 15)
	stats = blkioService.getBlkioStatsList(dockerStats, true, []uint64{})
	for _, stat := range stats {
		if stat.totals < totals[1] || stat.totals < totals[0] {
			t.Errorf("getBlkioStatsList(%v) => %v, want value bigger than %v", dockerStats, stat.totals, totals[1])
		}
	}

}

func TestBlkIOSkip(t *testing.T) {
	/*
	   For context, here's what a "raw" event looks like coming in from the docker stats API.
	   Note the repeated values across the two different major devices. Here 8:0 is a /dev/sda disk, 253:0 is the device mapper that exposes that disk's storage space.
	           "io_service_bytes_recursive": [
	               {
	                   "major": 8,
	                   "minor": 0,
	                   "op": "read",
	                   "value": 2359296
	               },
	               {
	                   "major": 8,
	                   "minor": 0,
	                   "op": "write",
	                   "value": 94544896
	               },
	               {
	                   "major": 253,
	                   "minor": 0,
	                   "op": "read",
	                   "value": 2359296
	               },
	               {
	                   "major": 253,
	                   "minor": 0,
	                   "op": "write",
	                   "value": 94544896
	               }
	           ],
	*/

	var readVal uint64 = 2359296
	var writeVal uint64 = 94544896
	testInt := []types.BlkioStatEntry{
		{
			Major: 8,
			Minor: 0,
			Op:    "Read",
			Value: readVal,
		},
		{
			Major: 8,
			Minor: 0,
			Op:    "Write",
			Value: writeVal,
		},
		{
			Major: 253,
			Minor: 0,
			Op:    "Read",
			Value: readVal,
		},
		{
			Major: 253,
			Minor: 0,
			Op:    "Write",
			Value: writeVal,
		},
	}

	skip := []uint64{253}

	combined := getNewStats(skip, time.Now(), testInt)
	assert.Equal(t, readVal, combined.reads)
	assert.Equal(t, writeVal, combined.writes)

}

func TestDeltaOneContainer(t *testing.T) {
	var apiContainer docker.Stat
	metrics := types.BlkioStatEntry{
		Major: 123,
		Minor: 123,
		Op:    "Total",
		Value: 123,
	}
	jsonContainers := `
     {
             "Id": "8dfafdbc3a40",
			 "Names": ["container"]
     }`
	var containers types.Container
	err := json.Unmarshal([]byte(jsonContainers), &containers)
	if err != nil {
		t.Fatal(err)
	}

	apiContainer.Stats.Read = time.Now()
	apiContainer.Container = &containers
	apiContainer.Stats.BlkioStats.IoServicedRecursive = append(apiContainer.Stats.BlkioStats.IoServicedRecursive, metrics)
	dockerStats := []docker.Stat{apiContainer}
	stats := blkioService.getBlkioStatsList(dockerStats, true, []uint64{})
	totals := make([]float64, 2)
	for _, stat := range stats {
		totals[0] = stat.totals
	}

	dockerStats[0].Stats.BlkioStats.IoServicedRecursive[0].Value = 1000
	dockerStats[0].Stats.Read = dockerStats[0].Stats.Read.Add(time.Second * 10)
	stats = blkioService.getBlkioStatsList(dockerStats, true, []uint64{})
	for _, stat := range stats {
		if stat.totals < totals[0] {
			t.Errorf("getBlkioStatsList(%v) => %v, want value bigger than %v", dockerStats, stat.totals, totals[0])
		}
	}

	dockerStats[0].Stats.BlkioStats.IoServicedRecursive[0].Value = 2000
	dockerStats[0].Stats.Read = dockerStats[0].Stats.Read.Add(time.Second * 15)
	stats = blkioService.getBlkioStatsList(dockerStats, true, []uint64{})
	for _, stat := range stats {
		if stat.totals < totals[1] || stat.totals < totals[0] {
			t.Errorf("getBlkioStatsList(%v) => %v, want value bigger than %v", dockerStats, stat.totals, totals[1])
		}
	}

}

func TestWritePs(t *testing.T) {
	oldWritePs := []uint64{220, 951, 0}
	newWritePs := []uint64{120, 2951, 0}
	for index := range oldBlkioRaw {
		setTime(index)
		oldBlkioRaw[index].writes = oldWritePs[index]
		newBlkioRaw[index].writes = newWritePs[index]
	}
	writePsTest := []struct {
		givenOld BlkioRaw
		givenNew BlkioRaw
		expected float64
	}{
		{oldBlkioRaw[0], newBlkioRaw[0], 0},
		{oldBlkioRaw[1], newBlkioRaw[1], 1000},
		{oldBlkioRaw[2], newBlkioRaw[2], 0},
	}
	for _, tt := range writePsTest {
		out := blkioService.getWritePs(&tt.givenOld, &tt.givenNew)
		if out != tt.expected {
			t.Errorf("getWritePs(%v,%v) => %v, want %v", tt.givenOld, tt.givenNew, out, tt.expected)
		}
	}
}

func TestReadPS(t *testing.T) {
	oldReasPs := []uint64{0, 951, 235}
	newReadPs := []uint64{120, 3951, 62}
	for index := range oldBlkioRaw {
		setTime(index)
		oldBlkioRaw[index].reads = oldReasPs[index]
		newBlkioRaw[index].reads = newReadPs[index]
	}
	readPsTest := []struct {
		givenOld BlkioRaw
		givenNew BlkioRaw
		expected float64
	}{
		{oldBlkioRaw[0], newBlkioRaw[0], 60},
		{oldBlkioRaw[1], newBlkioRaw[1], 1500},
		{oldBlkioRaw[2], newBlkioRaw[2], 0},
	}
	for _, tt := range readPsTest {
		out := blkioService.getReadPs(&tt.givenOld, &tt.givenNew)
		if out != tt.expected {
			t.Errorf("getReadPs(%v,%v) => %v, want %v", tt.givenOld, tt.givenNew, out, tt.expected)
		}
	}
}

func TestBlkioTotal(t *testing.T) {
	oldTotal := []uint64{40, 1954, 235}
	newTotal := []uint64{120, 1964, 62}
	for index := range oldBlkioRaw {
		setTime(index)
		oldBlkioRaw[index].totals = oldTotal[index]
		newBlkioRaw[index].totals = newTotal[index]
	}
	totalPsTest := []struct {
		givenOld BlkioRaw
		givenNew BlkioRaw
		expected float64
	}{
		{oldBlkioRaw[0], newBlkioRaw[0], 40},
		{oldBlkioRaw[1], newBlkioRaw[1], 5},
		{oldBlkioRaw[2], newBlkioRaw[2], 0},
	}
	for _, tt := range totalPsTest {
		out := blkioService.getTotalPs(&tt.givenOld, &tt.givenNew)
		if out != tt.expected {
			t.Errorf("getTotalPs(%v,%v) => %v, want %v", tt.givenOld, tt.givenNew, out, tt.expected)
		}
	}
}

func setTime(index int) {
	oldBlkioRaw[index].Time = time.Now()
	newBlkioRaw[index].Time = oldBlkioRaw[index].Time.Add(time.Duration(2000000000))
}

func TestGetBlkioStatsList(t *testing.T) {
	start := time.Now()
	later := start.Add(10 * time.Second)

	blkioService := BlkioService{
		map[string]BlkioRaw{
			"cebada": {Time: start, reads: 100, writes: 200, totals: 300},
		},
	}

	dockerStats := []docker.Stat{{
		Container: &types.Container{
			ID:    "cebada",
			Names: []string{"test"},
		},
		Stats: types.StatsJSON{Stats: types.Stats{
			Read: later,
			BlkioStats: types.BlkioStats{
				IoServicedRecursive: []types.BlkioStatEntry{
					{Major: 1, Minor: 1, Op: "Read", Value: 100},
					{Major: 1, Minor: 1, Op: "Write", Value: 200},
					{Major: 1, Minor: 1, Op: "Total", Value: 300},
					{Major: 1, Minor: 2, Op: "Read", Value: 50},
					{Major: 1, Minor: 2, Op: "Write", Value: 100},
					{Major: 1, Minor: 2, Op: "Total", Value: 150},
				},
				IoServiceBytesRecursive: []types.BlkioStatEntry{
					{Major: 1, Minor: 1, Op: "Read", Value: 1000},
					{Major: 1, Minor: 1, Op: "Write", Value: 2000},
					{Major: 1, Minor: 1, Op: "Total", Value: 3000},
					{Major: 1, Minor: 2, Op: "Read", Value: 500},
					{Major: 1, Minor: 2, Op: "Write", Value: 1000},
					{Major: 1, Minor: 2, Op: "Total", Value: 1500},
				},
				IoServiceTimeRecursive: []types.BlkioStatEntry{
					{Major: 1, Minor: 1, Op: "Read", Value: 10000},
					{Major: 1, Minor: 1, Op: "Write", Value: 20000},
					{Major: 1, Minor: 1, Op: "Total", Value: 30000},
					{Major: 1, Minor: 2, Op: "Read", Value: 500},
					{Major: 1, Minor: 2, Op: "Write", Value: 1500},
					{Major: 1, Minor: 2, Op: "Total", Value: 2000},
				},
				IoWaitTimeRecursive: []types.BlkioStatEntry{
					{Major: 1, Minor: 1, Op: "Read", Value: 1000000},
					{Major: 1, Minor: 1, Op: "Write", Value: 25604332},
					{Major: 1, Minor: 1, Op: "Total", Value: 26604332},
					{Major: 1, Minor: 2, Op: "Read", Value: 500},
					{Major: 1, Minor: 2, Op: "Write", Value: 1500},
					{Major: 1, Minor: 2, Op: "Total", Value: 2000},
				},
				IoQueuedRecursive: []types.BlkioStatEntry{
					{Major: 1, Minor: 1, Op: "Read", Value: 100},
					{Major: 1, Minor: 1, Op: "Write", Value: 200},
					{Major: 1, Minor: 1, Op: "Total", Value: 300},
					{Major: 1, Minor: 2, Op: "Read", Value: 50},
					{Major: 1, Minor: 2, Op: "Write", Value: 100},
					{Major: 1, Minor: 2, Op: "Total", Value: 150},
				},
			},
		}},
	}}

	statsList := blkioService.getBlkioStatsList(dockerStats, true, []uint64{})
	stats := statsList[0]
	assert.Equal(t, float64(5), stats.reads)
	assert.Equal(t, float64(10), stats.writes)
	assert.Equal(t, float64(15), stats.totals)
	assert.Equal(t,
		BlkioRaw{Time: later, reads: 150, writes: 300, totals: 450},
		stats.serviced)
	assert.Equal(t,
		BlkioRaw{Time: later, reads: 1500, writes: 3000, totals: 4500},
		stats.servicedBytes)
	assert.Equal(t,
		BlkioRaw{Time: later, reads: 10500, writes: 21500, totals: 32000},
		stats.servicedTime)
	assert.Equal(t,
		BlkioRaw{Time: later, reads: 1000500, writes: 25605832, totals: 26606332},
		stats.waitTime)
	assert.Equal(t,
		BlkioRaw{Time: later, reads: 150, writes: 300, totals: 450},
		stats.queued)
}

func TestGetBlkioStatsListWindows(t *testing.T) {
	start := time.Now()
	later := start.Add(10 * time.Second)

	blkioService := BlkioService{
		map[string]BlkioRaw{
			"cebada": {Time: start, reads: 100, writes: 200, totals: 300},
		},
	}

	dockerStats := []docker.Stat{{
		Container: &types.Container{
			ID:    "cebada",
			Names: []string{"test"},
		},
		Stats: types.StatsJSON{Stats: types.Stats{
			Read: later,
			StorageStats: types.StorageStats{
				ReadCountNormalized:  150,
				WriteCountNormalized: 300,
				ReadSizeBytes:        1500,
				WriteSizeBytes:       3000,
			},
		}},
	}}

	statsList := blkioService.getBlkioStatsList(dockerStats, true, []uint64{})
	stats := statsList[0]
	assert.Equal(t, float64(5), stats.reads)
	assert.Equal(t, float64(10), stats.writes)
	assert.Equal(t, float64(15), stats.totals)
	assert.Equal(t,
		BlkioRaw{Time: later, reads: 150, writes: 300, totals: 450},
		stats.serviced)
	assert.Equal(t,
		BlkioRaw{Time: later, reads: 1500, writes: 3000, totals: 4500},
		stats.servicedBytes)
}
