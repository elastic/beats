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

package cgv1

import (
	"bufio"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v8/libbeat/metric/system/cgroup/cgcommon"
	"github.com/elastic/beats/v8/libbeat/opt"
)

// MemorySubsystem contains the metrics and limits from the "memory" subsystem.
type MemorySubsystem struct {
	ID   string `json:"id,omitempty"`   // ID of the cgroup.
	Path string `json:"path,omitempty"` // Path to the cgroup relative to the cgroup subsystem's mountpoint.

	Mem       MemoryData `json:"mem" struct:"mem"`           // Memory usage by tasks in this cgroup.
	MemSwap   MemoryData `json:"memsw" struct:"memsw"`       // Memory plus swap usage by tasks in this cgroup.
	Kernel    MemoryData `json:"kmem" struct:"kmem"`         // Kernel memory used by tasks in this cgroup.
	KernelTCP MemoryData `json:"kmem_tcp" struct:"kmem_tcp"` // Kernel TCP buffer memory used by tasks in this cgroup.
	Stats     MemoryStat `json:"stats" struct:"stats"`       // A wide range of memory statistics.
}

// MemoryData groups related memory usage metrics and limits.
type MemoryData struct {
	Usage    MemSubsystemUsage `json:"usage" struct:"usage"`       // Usage in bytes.
	Limit    opt.Bytes         `json:"limit" struct:"limit"`       // Limit in bytes.
	Failures uint64            `json:"failures" struct:"failures"` // Number of times the memory limit has been reached.
}

// MemSubsystemUsage groups fields used in memory.SUBSYSTEM.usage
type MemSubsystemUsage struct {
	Bytes uint64    `json:"bytes" struct:"bytes"`
	Max   opt.Bytes `json:"max" struct:"max"`
}

// MemoryStat contains various memory statistics and accounting information
// associated with a cgroup.
type MemoryStat struct {
	// Page cache, including tmpfs (shmem), in bytes.
	Cache opt.Bytes `json:"cache" struct:"cache"`
	// Anonymous and swap cache, not including tmpfs (shmem), in bytes.
	RSS opt.Bytes `json:"rss" struct:"rss"`
	// Anonymous transparent hugepages in bytes.
	RSSHuge opt.Bytes `json:"rss_huge" struct:"rss_huge"`
	// Size of memory-mapped mapped files, including tmpfs (shmem), in bytes.
	MappedFile opt.Bytes `json:"mapped_file" struct:"mapped_file"`
	// Number of pages paged into memory.
	PagesIn uint64 `json:"pages_in" struct:"pages_in"`
	// Number of pages paged out of memory.
	PagesOut uint64 `json:"pages_out" struct:"pages_out"`
	// Number of times a task in the cgroup triggered a page fault.
	PageFaults uint64 `json:"page_faults" struct:"page_faults"`
	// Number of times a task in the cgroup triggered a major page fault.
	MajorPageFaults uint64 `json:"major_page_faults" struct:"major_page_faults"`
	// Swap usage in bytes.
	Swap opt.Bytes `json:"swap"`
	// Anonymous and swap cache on active least-recently-used (LRU) list, including tmpfs (shmem), in bytes.
	ActiveAnon opt.Bytes `json:"active_anon" struct:"active_anon"`
	// Anonymous and swap cache on inactive LRU list, including tmpfs (shmem), in bytes.
	InactiveAnon opt.Bytes `json:"inactive_anon" struct:"inactive_anon"`
	// File-backed memory on active LRU list, in bytes.
	ActiveFile opt.Bytes `json:"active_file" struct:"active_file"`
	// File-backed memory on inactive LRU list, in bytes.
	InactiveFile opt.Bytes `json:"inactive_file" struct:"inactive_file"`
	// Memory that cannot be reclaimed, in bytes.
	Unevictable opt.Bytes `json:"unevictable" struct:"unevictable"`
	// Memory limit for the hierarchy that contains the memory cgroup, in bytes.
	HierarchicalMemoryLimit opt.Bytes `json:"hierarchical_memory_limit" struct:"hierarchical_memory_limit"`
	// Memory plus swap limit for the hierarchy that contains the memory cgroup, in bytes.
	HierarchicalMemswLimit opt.Bytes `json:"hierarchical_memsw_limit" struct:"hierarchical_memsw_limit"`
}

// Get reads metrics from the "memory" subsystem. path is the filepath to the
// cgroup hierarchy to read.
func (mem *MemorySubsystem) Get(path string) error {
	if err := memoryData(path, "memory", &mem.Mem); err != nil {
		return errors.Wrap(err, "error fetching memory stats")
	}

	if err := memoryData(path, "memory.memsw", &mem.MemSwap); err != nil {
		return errors.Wrap(err, "error fetching memsw stats")
	}

	if err := memoryData(path, "memory.kmem", &mem.Kernel); err != nil {
		return errors.Wrap(err, "error fetching kmem stats")
	}

	if err := memoryData(path, "memory.kmem.tcp", &mem.KernelTCP); err != nil {
		return errors.Wrap(err, "error fetching kmem.tcp stats")
	}

	if err := memoryStats(path, mem); err != nil {
		return errors.Wrap(err, "error fetching memory.stat metrics")
	}

	return nil
}

func memoryData(path, prefix string, data *MemoryData) error {
	var err error
	data.Usage.Bytes, err = cgcommon.ParseUintFromFile(path, prefix+".usage_in_bytes")
	if err != nil {
		return errors.Wrap(err, "error fetching usage_in_bytes")
	}

	data.Usage.Max.Bytes, err = cgcommon.ParseUintFromFile(path, prefix+".max_usage_in_bytes")
	if err != nil {
		return errors.Wrap(err, "error fetching max_usage_in_bytes")
	}

	data.Limit.Bytes, err = cgcommon.ParseUintFromFile(path, prefix+".limit_in_bytes")
	if err != nil {
		return errors.Wrap(err, "error fetching limit_in_bytes")
	}

	data.Failures, err = cgcommon.ParseUintFromFile(path, prefix+".failcnt")
	if err != nil {
		return errors.Wrap(err, "error fetching failcnt")
	}

	return nil
}

func memoryStats(path string, mem *MemorySubsystem) error {
	f, err := os.Open(filepath.Join(path, "memory.stat"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		t, v, err := cgcommon.ParseCgroupParamKeyValue(sc.Text())
		if err != nil {
			return err
		}
		switch t {
		case "cache":
			mem.Stats.Cache.Bytes = v
		case "rss":
			mem.Stats.RSS.Bytes = v
		case "rss_huge":
			mem.Stats.RSSHuge.Bytes = v
		case "mapped_file":
			mem.Stats.MappedFile.Bytes = v
		case "pgpgin":
			mem.Stats.PagesIn = v
		case "pgpgout":
			mem.Stats.PagesOut = v
		case "pgfault":
			mem.Stats.PageFaults = v
		case "pgmajfault":
			mem.Stats.MajorPageFaults = v
		case "swap":
			mem.Stats.Swap.Bytes = v
		case "active_anon":
			mem.Stats.ActiveAnon.Bytes = v
		case "inactive_anon":
			mem.Stats.InactiveAnon.Bytes = v
		case "active_file":
			mem.Stats.ActiveFile.Bytes = v
		case "inactive_file":
			mem.Stats.InactiveFile.Bytes = v
		case "unevictable":
			mem.Stats.Unevictable.Bytes = v
		case "hierarchical_memory_limit":
			mem.Stats.HierarchicalMemoryLimit.Bytes = v
		case "hierarchical_memsw_limit":
			mem.Stats.HierarchicalMemswLimit.Bytes = v
		}
	}

	return sc.Err()
}
