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

// +build darwin freebsd linux openbsd windows

package memory

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	sysinfo "github.com/elastic/go-sysinfo"
	sysinfotypes "github.com/elastic/go-sysinfo/types"
	sigar "github.com/elastic/gosigar"
)

// MemStat includes the memory usage statistics and ratios of usage and total memory usage
type MemStat struct {
	sigar.Mem
	UsedPercent       float64 `json:"used_p"`
	ActualUsedPercent float64 `json:"actual_used_p"`
}

// Get returns the memory stats of the host
func Get() (*MemStat, error) {
	mem := sigar.Mem{}
	err := mem.Get()
	if err != nil {
		return nil, err
	}

	return &MemStat{Mem: mem}, nil
}

// AddMemPercentage calculates the ratio of used and total size of memory
func AddMemPercentage(m *MemStat) {
	if m.Mem.Total == 0 {
		return
	}

	perc := float64(m.Mem.Used) / float64(m.Mem.Total)
	m.UsedPercent = common.Round(perc, common.DefaultDecimalPlacesCount)

	actualPerc := float64(m.Mem.ActualUsed) / float64(m.Mem.Total)
	m.ActualUsedPercent = common.Round(actualPerc, common.DefaultDecimalPlacesCount)
}

// SwapStat includes the current swap usage and the ratio of used and total swap size
type SwapStat struct {
	sigar.Swap
	UsedPercent float64 `json:"used_p"`
}

// GetSwap returns the swap usage of the host
func GetSwap() (*SwapStat, error) {
	swap := sigar.Swap{}
	err := swap.Get()
	if err != nil {
		return nil, err
	}

	// This shouldn't happen, but it has been reported to happen and
	// this can provoke too big values for used swap.
	// Workaround this by assuming that all swap is free in that case.
	if swap.Free > swap.Total || swap.Used > swap.Total {
		logp.Debug("memory",
			"Unexpected values for swap memory - total: %v free: %v used: %v.  Setting swap used to 0.",
			swap.Total, swap.Free, swap.Used)
		swap.Free = swap.Total
		swap.Used = 0
	}

	return &SwapStat{Swap: swap}, nil
}

// GetMemoryEvent returns the event created from memory statistics
func GetMemoryEvent(memStat *MemStat) common.MapStr {
	return common.MapStr{
		"total":         memStat.Total,
		"used":          memStat.Used,
		"free":          memStat.Free,
		"actual_used":   memStat.ActualUsed,
		"actual_free":   memStat.ActualFree,
		"used_p":        memStat.UsedPercent,
		"actual_used_p": memStat.ActualUsedPercent,
	}
}

// GetSwapEvent returns the event created from swap usage
func GetSwapEvent(swapStat *SwapStat) common.MapStr {
	return common.MapStr{
		"total":  swapStat.Total,
		"used":   swapStat.Used,
		"free":   swapStat.Free,
		"used_p": swapStat.UsedPercent,
	}
}

// AddSwapPercentage calculates the ratio of used and total swap size
func AddSwapPercentage(s *SwapStat) {
	if s.Swap.Total == 0 {
		return
	}

	perc := float64(s.Swap.Used) / float64(s.Swap.Total)
	s.UsedPercent = common.Round(perc, common.DefaultDecimalPlacesCount)
}

// HugeTLBPagesStat includes metrics about huge pages usage
type HugeTLBPagesStat struct {
	sigar.HugeTLBPages
	UsedPercent float64 `json:"used_p"`
}

// GetHugeTLBPages returns huge pages usage metrics
func GetHugeTLBPages() (*HugeTLBPagesStat, error) {
	pages := sigar.HugeTLBPages{}
	err := pages.Get()

	if err == nil {
		return &HugeTLBPagesStat{HugeTLBPages: pages}, nil
	}

	if sigar.IsNotImplemented(err) {
		return nil, nil
	}

	return nil, err
}

// AddHugeTLBPagesPercentage calculates ratio of used huge pages
func AddHugeTLBPagesPercentage(s *HugeTLBPagesStat) {
	if s.Total == 0 {
		return
	}

	perc := float64(s.Total-s.Free+s.Reserved) / float64(s.Total)
	s.UsedPercent = common.Round(perc, common.DefaultDecimalPlacesCount)
}

// GetVMStat gets linux vmstat metrics
func GetVMStat() (*sysinfotypes.VMStatInfo, error) {
	h, err := sysinfo.Host()
	if err != nil {
		return nil, errors.Wrap(err, "failed to read self process information")
	}
	if vmstatHandle, ok := h.(sysinfotypes.VMStat); ok {
		info, err := vmstatHandle.VMStat()
		if err != nil {
			return nil, errors.Wrap(err, "error getting VMStat info")
		}
		return info, nil
	}
	return nil, nil

}
