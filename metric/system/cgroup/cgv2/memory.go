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

package cgv2

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/elastic/elastic-agent-libs/opt"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/cgroup/cgcommon"
)

// MemorySubsystem contains the metrics and limits from the "memory" subsystem.
type MemorySubsystem struct {
	ID   string `json:"id,omitempty"`   // ID of the cgroup.
	Path string `json:"path,omitempty"` // Path to the cgroup relative to the cgroup subsystem's mountpoint.

	Mem      MemoryData                   `json:"mem" struct:"mem"`                               // Memory usage by tasks in this cgroup.
	MemSwap  MemoryData                   `json:"memsw" struct:"memsw"`                           // Memory plus swap usage by tasks in this cgroup.
	Stats    MemoryStat                   `json:"stats" struct:"stats"`                           // A wide range of memory statistics.
	Pressure map[string]cgcommon.Pressure `json:"pressure,omitempty" struct:"pressure,omitempty"` // Shows pressure stall information for memory.
}

// MemoryData contains basic metrics for the V2 controller
type MemoryData struct {
	Events Events       `json:"events" struct:"events"`
	Usage  opt.Bytes    `json:"usage" struct:"usage"`
	Low    opt.Bytes    `json:"low" struct:"low"`
	High   opt.BytesOpt `json:"high,omitempty" struct:"high,omitempty"`
	Max    opt.BytesOpt `json:"max,omitempty" struct:"max,omitempty"`
}

// Events contains the data from *.events in the memory controller
type Events struct {
	Low     opt.Uint `json:"low,omitempty" struct:"low,omitempty"`
	High    uint64   `json:"high" struct:"high"`
	Max     uint64   `json:"max" struct:"max"`
	OOM     opt.Uint `json:"oom,omitempty" struct:"oom,omitempty"`
	OOMKill opt.Uint `json:"oom_kill,omitempty" struct:"oom_kill,omitempty"`
	Fail    opt.Uint `json:"fail,omitempty" struct:"fail,omitempty"`
}

// MemoryStat holds detailed stats for the memory controller
type MemoryStat struct {
	//Amount of memory used in anonymous mappings
	Anon opt.Bytes `json:"anon" struct:"anon" orig:"anon"`
	//Amount of memory used to cache filesystem data, including tmpfs and shared memory.
	File opt.Bytes `json:"file" struct:"file" orig:"file"`
	// Amount of memory allocated to kernel stacks.
	KernelStack opt.Bytes `json:"kernel_stack" struct:"kernel_stack" orig:"kernel_stack"`
	//Amount of memory allocated for page tables.
	Pagetables opt.Bytes `json:"page_tables" struct:"page_tables" orig:"pagetables"`
	// Amount of memory used for storing per-cpu kernel data structures.
	PerCPU opt.Bytes `json:"per_cpu" struct:"per_cpu" orig:"percpu"`
	// Amount of memory used in network transmission buffers
	Sock opt.Bytes `json:"sock" struct:"sock" orig:"sock"`
	// Amount of cached filesystem data that is swap-backed, such as tmpfs, shm segments, shared anonymous mmap()s
	Shmem opt.Bytes `json:"shmem" struct:"shmem" orig:"shmem"`
	// Amount of cached filesystem data mapped with mmap()
	FileMapped opt.Bytes `json:"file_mapped" struct:"file_mapped" orig:"file_mapped"`
	//Amount of cached filesystem data that was modified but not yet written back to disk
	FileDirty opt.Bytes `json:"file_dirty" struct:"file_dirty" orig:"file_dirty"`
	// Amount of cached filesystem data that was modified and is currently being written back to disk
	FileWriteback opt.Bytes `json:"file_writeback" struct:"file_writeback" orig:"file_writeback"`
	// Amount of swap cached in memory. The swapcache is accounted against both memory and swap usage.
	SwapCached opt.Bytes `json:"swap_cached" struct:"swap_cached" orig:"swapcached"`
	// Amount of memory used in anonymous mappings backed by transparent hugepages
	AnonTHP opt.Bytes `json:"anon_thp" struct:"anon_thp" orig:"anon_thp"`
	// Amount of cached filesystem data backed by transparent hugepages
	FileTHP opt.Bytes `json:"file_thp" struct:"file_thp" orig:"file_thp"`
	// Amount of shm, tmpfs, shared anonymous mmap()s backed by transparent hugepages
	ShmemTHP opt.Bytes `json:"shmem_thp" struct:"shmem_thp" orig:"shmem_thp"`
	// Anonymous and swap cache on inactive LRU list, including tmpfs (shmem), in bytes.
	InactiveAnon opt.Bytes `json:"inactive_anon" struct:"inactive_anon" orig:"inactive_anon"`
	// Anonymous and swap cache on active least-recently-used (LRU) list, including tmpfs (shmem), in bytes.
	ActiveAnon opt.Bytes `json:"active_anon" struct:"active_anon" orig:"active_anon"`
	// File-backed memory on inactive LRU list, in bytes.
	InactiveFile opt.Bytes `json:"inactive_file" struct:"inactive_file" orig:"inactive_file"`
	// File-backed memory on active LRU list, in bytes.
	ActiveFile opt.Bytes `json:"active_file" struct:"active_file" orig:"active_file"`
	// Memory that cannot be reclaimed, in bytes.
	Unevictable opt.Bytes `json:"unevictable" struct:"unevictable" orig:"unevictable"`
	// Part of "slab" that might be reclaimed, such as dentries and inodes.
	SlabReclaimable opt.Bytes `json:"slab_reclaimable" struct:"slab_reclaimable" orig:"slab_reclaimable"`
	// Part of "slab" that cannot be reclaimed on memory pressure.
	SlabUnreclaimable opt.Bytes `json:"slab_unreclaimable" struct:"slab_unreclaimable" orig:"slab_unreclaimable"`
	// Amount of memory used for storing in-kernel data structures.
	Slab opt.Bytes `json:"slab" struct:"slab" orig:"slab"`
	// Number of refaults of previously evicted anonymous pages.
	WorkingSetRefaultAnon uint64 `json:"workingset_refault_anon" struct:"workingset_refault_anon" orig:"workingset_refault_anon"`
	// Number of refaults of previously evicted file pages.
	WorkingSetRefaultFile uint64 `json:"workingset_refault_file" struct:"workingset_refault_file" orig:"workingset_refault_file"`
	// Number of refaulted anonymous pages that were immediately activated.
	WorkingSetActivateAnon uint64 `json:"workingset_activate_anon" struct:"workingset_activate_anon" orig:"workingset_activate_anon"`
	// Number of refaulted file pages that were immediately activated.
	WorkingSetActivateFile uint64 `json:"workingset_activate_file" struct:"workingset_activate_file" orig:"workingset_activate_file"`
	// Number of restored anonymous pages which have been detected as an active workingset before they got reclaimed.
	WorkingSetRestoreAnon uint64 `json:"workingset_restore_anon" struct:"workingset_restore_anon" orig:"workingset_restore_anon"`
	// Number of restored file pages which have been detected as an active workingset before they got reclaimed.
	WorkingSetRestoreFile uint64 `json:"workingset_restore_file" struct:"workingset_restore_file" orig:"workingset_restore_file"`
	// Number of times a shadow node has been reclaimed
	WorkingSetNodeReclaim uint64 `json:"workingset_node_reclaim" struct:"workingset_node_reclaim" orig:"workingset_nodereclaim"`
	//Total number of page faults incurred
	PageFaults uint64 `json:"page_faults" struct:"page_faults" orig:"pgfault"`
	// Number of times a task in the cgroup triggered a major page fault.
	MajorPageFaults uint64 `json:"major_page_faults" struct:"major_page_faults" orig:"pgmajfault"`
	// Amount of scanned pages (in an active LRU list)
	PageRefill uint64 `json:"page_refill" struct:"page_refill" orig:"pgrefill"`
	// Amount of scanned pages (in an inactive LRU list)
	PageScan uint64 `json:"page_scan" struct:"page_scan" orig:"pgscan"`
	// Amount of reclaimed pages
	PageSteal uint64 `json:"page_steal" struct:"page_steal" orig:"pgsteal"`
	//Amount of pages moved to the active LRU list
	PageActivate uint64 `json:"page_activate" struct:"page_activate" orig:"pgactivate"`
	// Amount of pages moved to the inactive LRU list
	PageDeactivate uint64 `json:"page_deactivate" struct:"page_deactivate" orig:"pgdeactivate"`
	// Amount of pages postponed to be freed under memory pressure
	PageLazyFree uint64 `json:"page_lazy_free" struct:"page_lazy_free" orig:"pglazyfree"`
	// Amount of reclaimed lazyfree pages
	PageLazyFreed uint64 `json:"page_lazy_freed" struct:"page_lazy_freed" orig:"pglazyfreed"`
	// Number of transparent hugepages which were allocated to satisfy a page fault.
	THPFaultAlloc uint64 `json:"thp_fault_alloc" struct:"thp_fault_alloc" orig:"thp_fault_alloc"`
	// Number of transparent hugepages which were allocated to allow collapsing an existing range of pages.
	THPCollapseAlloc uint64 `json:"htp_collapse_alloc" struct:"htp_collapse_alloc" orig:"thp_collapse_alloc"`
}

// Map of file key to setter function
var keySetters = map[string]func(*MemoryStat, uint64){
	"anon":                     func(stats *MemoryStat, v uint64) { stats.Anon = opt.Bytes{Bytes: v} },
	"file":                     func(stats *MemoryStat, v uint64) { stats.File = opt.Bytes{Bytes: v} },
	"kernel_stack":             func(stats *MemoryStat, v uint64) { stats.KernelStack = opt.Bytes{Bytes: v} },
	"pagetables":               func(stats *MemoryStat, v uint64) { stats.Pagetables = opt.Bytes{Bytes: v} },
	"percpu":                   func(stats *MemoryStat, v uint64) { stats.PerCPU = opt.Bytes{Bytes: v} },
	"sock":                     func(stats *MemoryStat, v uint64) { stats.Sock = opt.Bytes{Bytes: v} },
	"shmem":                    func(stats *MemoryStat, v uint64) { stats.Shmem = opt.Bytes{Bytes: v} },
	"file_mapped":              func(stats *MemoryStat, v uint64) { stats.FileMapped = opt.Bytes{Bytes: v} },
	"file_dirty":               func(stats *MemoryStat, v uint64) { stats.FileDirty = opt.Bytes{Bytes: v} },
	"file_writeback":           func(stats *MemoryStat, v uint64) { stats.FileWriteback = opt.Bytes{Bytes: v} },
	"swapcached":               func(stats *MemoryStat, v uint64) { stats.SwapCached = opt.Bytes{Bytes: v} },
	"anon_thp":                 func(stats *MemoryStat, v uint64) { stats.AnonTHP = opt.Bytes{Bytes: v} },
	"file_thp":                 func(stats *MemoryStat, v uint64) { stats.FileTHP = opt.Bytes{Bytes: v} },
	"shmem_thp":                func(stats *MemoryStat, v uint64) { stats.ShmemTHP = opt.Bytes{Bytes: v} },
	"inactive_anon":            func(stats *MemoryStat, v uint64) { stats.InactiveAnon = opt.Bytes{Bytes: v} },
	"active_anon":              func(stats *MemoryStat, v uint64) { stats.ActiveAnon = opt.Bytes{Bytes: v} },
	"inactive_file":            func(stats *MemoryStat, v uint64) { stats.InactiveFile = opt.Bytes{Bytes: v} },
	"active_file":              func(stats *MemoryStat, v uint64) { stats.ActiveFile = opt.Bytes{Bytes: v} },
	"unevictable":              func(stats *MemoryStat, v uint64) { stats.Unevictable = opt.Bytes{Bytes: v} },
	"slab_reclaimable":         func(stats *MemoryStat, v uint64) { stats.SlabReclaimable = opt.Bytes{Bytes: v} },
	"slab_unreclaimable":       func(stats *MemoryStat, v uint64) { stats.SlabUnreclaimable = opt.Bytes{Bytes: v} },
	"slab":                     func(stats *MemoryStat, v uint64) { stats.Slab = opt.Bytes{Bytes: v} },
	"workingset_refault_anon":  func(stats *MemoryStat, v uint64) { stats.WorkingSetRefaultAnon = v },
	"workingset_refault_file":  func(stats *MemoryStat, v uint64) { stats.WorkingSetRefaultFile = v },
	"workingset_activate_anon": func(stats *MemoryStat, v uint64) { stats.WorkingSetActivateAnon = v },
	"workingset_activate_file": func(stats *MemoryStat, v uint64) { stats.WorkingSetActivateFile = v },
	"workingset_restore_anon":  func(stats *MemoryStat, v uint64) { stats.WorkingSetRestoreAnon = v },
	"workingset_restore_file":  func(stats *MemoryStat, v uint64) { stats.WorkingSetRestoreFile = v },
	"workingset_nodereclaim":   func(stats *MemoryStat, v uint64) { stats.WorkingSetNodeReclaim = v },
	"pgfault":                  func(stats *MemoryStat, v uint64) { stats.PageFaults = v },
	"pgmajfault":               func(stats *MemoryStat, v uint64) { stats.MajorPageFaults = v },
	"pgrefill":                 func(stats *MemoryStat, v uint64) { stats.PageRefill = v },
	"pgscan":                   func(stats *MemoryStat, v uint64) { stats.PageScan = v },
	"pgsteal":                  func(stats *MemoryStat, v uint64) { stats.PageSteal = v },
	"pgactivate":               func(stats *MemoryStat, v uint64) { stats.PageActivate = v },
	"pgdeactivate":             func(stats *MemoryStat, v uint64) { stats.PageDeactivate = v },
	"pglazyfree":               func(stats *MemoryStat, v uint64) { stats.PageLazyFree = v },
	"pglazyfreed":              func(stats *MemoryStat, v uint64) { stats.PageLazyFreed = v },
	"thp_fault_alloc":          func(stats *MemoryStat, v uint64) { stats.THPFaultAlloc = v },
	"thp_collapse_alloc":       func(stats *MemoryStat, v uint64) { stats.THPCollapseAlloc = v },
}

// Get fetches memory subsystem metrics for V2 cgroups
func (mem *MemorySubsystem) Get(path string) error {
	var err error
	mem.Pressure, err = cgcommon.GetPressure(filepath.Join(path, "memory.pressure"))
	// Not all systems have pressure stats. Treat this as a soft error.
	if os.IsNotExist(err) {
		err = nil
	}
	if err != nil {
		return fmt.Errorf("error fetching memory.pressure data: %w", err)
	}

	mem.Mem, err = memoryData(path, "memory")
	if err != nil {
		return fmt.Errorf("error reading memory stats: %w", err)
	}

	mem.MemSwap, err = memoryData(path, "memory.swap")
	if err != nil {
		return fmt.Errorf("error reading memory.swap stats: %w", err)
	}

	mem.Stats, err = fillStatStruct(path)
	if err != nil {
		return fmt.Errorf("error fetching memory.stat: %w", err)
	}

	return nil
}

// memoryData reads off the the auxiliary memory stats from the memory controller
func memoryData(path, file string) (MemoryData, error) {

	// root cgroups won't have these files.
	// If .high doesn't exist, assume the rest don't either.
	_, err := os.Stat(filepath.Join(path, file+".high"))
	if errors.Is(err, os.ErrNotExist) {
		return MemoryData{}, nil
	}

	data := MemoryData{}
	// High and max can be set to "max", which means "off"
	lowMetric, err := cgcommon.ParseUintFromFile(filepath.Join(path, file+".low"))
	if err != nil {
		return data, fmt.Errorf("error reading %s.low file: %w", file, err)
	}

	highMetric, err := maxOrValue(path, file+".high")
	if err != nil {
		return data, fmt.Errorf("error parsing %s.high file: %w", file, err)
	}

	maxMetric, err := maxOrValue(path, file+".max")
	if err != nil {
		return data, fmt.Errorf("error parsing %s.max file: %w", file, err)
	}

	currentMetric, err := cgcommon.ParseUintFromFile(filepath.Join(path, file+".current"))
	if err != nil {
		return data, fmt.Errorf("error reading %s.current file: %w", file, err)
	}

	data.Low.Bytes = lowMetric
	data.High.Bytes = highMetric
	data.Max.Bytes = maxMetric
	data.Usage.Bytes = currentMetric
	data.Events, err = fetchEventsFile(path, file+".events")
	if err != nil {
		return data, fmt.Errorf("error fetching events file for %s: %w", file, err)
	}

	return data, nil
}

// fetch memory.events contents
func fetchEventsFile(path, file string) (Events, error) {
	evt := Events{}
	toRead := filepath.Join(path, file)
	f, err := os.Open(toRead)
	if err != nil {
		return evt, fmt.Errorf("error reading %s: %w", toRead, err)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		key, val, err := cgcommon.ParseCgroupParamKeyValue(sc.Text())
		if err != nil {
			return evt, fmt.Errorf("error parsing key from events: %w", err)
		}
		switch key {
		case "low":
			evt.Low = opt.UintWith(val)
		case "high":
			evt.High = val
		case "max":
			evt.Max = val
		case "oom":
			evt.OOM = opt.UintWith(val)
		case "oom_kill":
			evt.OOMKill = opt.UintWith(val)
		case "fail":
			evt.Fail = opt.UintWith(val)
		}
	}

	return evt, nil
}

// Some values, such as mem.max and mem.high, can be set to "max," which disables the metric.
func maxOrValue(path, file string) (opt.Uint, error) {
	var finalMetric opt.Uint
	highRaw, err := os.ReadFile(filepath.Join(path, file))
	if err != nil {
		return finalMetric, fmt.Errorf("error reading %s.high file: %w", path, err)
	}

	if strings.TrimSpace(string(highRaw)) == "max" {
		finalMetric = opt.NewUintNone()
	} else {
		highUint, err := cgcommon.ParseUint(highRaw)
		if err != nil {
			return finalMetric, fmt.Errorf("error parsing raw high value: %v: %w", highRaw, err)
		}
		finalMetric = opt.UintWith(highUint)
	}

	return finalMetric, nil
}

// fillStatStruct iteratively fills out the MemoryStat struct
// This works via reflection, and it's a tad ugly, but we also have a lot of fields to fill
// Note that this assumes all the values in the struct are either `uint64`, `opt.Bytes` or `opt.BytesOpt`
func fillStatStruct(path string) (MemoryStat, error) {
	statPath := filepath.Join(path, "memory.stat")
	raw, err := os.ReadFile(statPath)
	if err != nil {
		return MemoryStat{}, fmt.Errorf("error reading memory.stat: %w", err)
	}

	stats := MemoryStat{}
	sc := bufio.NewScanner(bytes.NewReader(raw))
	var parseErrs error
	for sc.Scan() {
		//break apart the lines
		parts := bytes.SplitN(sc.Bytes(), []byte(" "), 2)
		if len(parts) != 2 {
			continue
		}
		val, err := cgcommon.ParseUint(parts[1])
		if err != nil {
			parseErrs = errors.Join(parseErrs, fmt.Errorf("error parsing value %v: %w", parts[1], err))
			continue
		}
		if setter, found := keySetters[string(parts[0])]; found {
			setter(&stats, val)
		}
	}

	return stats, parseErrs
}
