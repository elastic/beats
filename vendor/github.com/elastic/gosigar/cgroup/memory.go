package cgroup

import (
	"bufio"
	"os"
	"path/filepath"
)

// MemorySubsystem contains the metrics and limits from the "memory" subsystem.
type MemorySubsystem struct {
	Metadata
	Mem       MemoryData `json:"mem"`      // Memory usage by tasks in this cgroup.
	MemSwap   MemoryData `json:"memsw"`    // Memory plus swap usage by tasks in this cgroup.
	Kernel    MemoryData `json:"kmem"`     // Kernel memory used by tasks in this cgroup.
	KernelTCP MemoryData `json:"kmem_tcp"` // Kernel TCP buffer memory used by tasks in this cgroup.
	Stats     MemoryStat `json:"stats"`    // A wide range of memory statistics.
}

// MemoryData groups related memory usage metrics and limits.
type MemoryData struct {
	Usage     uint64 `json:"usage"`         // Usage in bytes.
	MaxUsage  uint64 `json:"max_usage"`     // Max usage in bytes.
	Limit     uint64 `json:"limit"`         // Limit in bytes.
	FailCount uint64 `json:"failure_count"` // Number of times the memory limit has been reached.
}

// MemoryStat contains various memory statistics and accounting information
// associated with a cgroup.
type MemoryStat struct {
	// Page cache, including tmpfs (shmem), in bytes.
	Cache uint64 `json:"cache"`
	// Anonymous and swap cache, not including tmpfs (shmem), in bytes.
	RSS uint64 `json:"rss"`
	// Anonymous transparent hugepages in bytes.
	RSSHuge uint64 `json:"rss_huge"`
	// Size of memory-mapped mapped files, including tmpfs (shmem), in bytes.
	MappedFile uint64 `json:"mapped_file"`
	// Number of pages paged into memory.
	PagesIn uint64 `json:"pgpgin"`
	// Number of pages paged out of memory.
	PagesOut uint64 `json:"pgpgout"`
	// Number of times a task in the cgroup triggered a page fault.
	PageFaults uint64 `json:"pgfault"`
	// Number of times a task in the cgroup triggered a major page fault.
	MajorPageFaults uint64 `json:"pgmajfault"`
	// Swap usage in bytes.
	Swap uint64 `json:"swap"`
	// Anonymous and swap cache on active least-recently-used (LRU) list, including tmpfs (shmem), in bytes.
	ActiveAnon uint64 `json:"active_anon"`
	// Anonymous and swap cache on inactive LRU list, including tmpfs (shmem), in bytes.
	InactiveAnon uint64 `json:"inactive_anon"`
	// File-backed memory on active LRU list, in bytes.
	ActiveFile uint64 `json:"active_file"`
	// File-backed memory on inactive LRU list, in bytes.
	InactiveFile uint64 `json:"inactive_file"`
	// Memory that cannot be reclaimed, in bytes.
	Unevictable uint64 `json:"unevictable"`
	// Memory limit for the hierarchy that contains the memory cgroup, in bytes.
	HierarchicalMemoryLimit uint64 `json:"hierarchical_memory_limit"`
	// Memory plus swap limit for the hierarchy that contains the memory cgroup, in bytes.
	HierarchicalMemswLimit uint64 `json:"hierarchical_memsw_limit"`
}

// get reads metrics from the "memory" subsystem. path is the filepath to the
// cgroup hierarchy to read.
func (mem *MemorySubsystem) get(path string) error {
	if err := memoryData(path, "memory", &mem.Mem); err != nil {
		return err
	}

	if err := memoryData(path, "memory.memsw", &mem.MemSwap); err != nil {
		return err
	}

	if err := memoryData(path, "memory.kmem", &mem.Kernel); err != nil {
		return err
	}

	if err := memoryData(path, "memory.kmem.tcp", &mem.KernelTCP); err != nil {
		return err
	}

	if err := memoryStats(path, mem); err != nil {
		return err
	}

	return nil
}

func memoryData(path, prefix string, data *MemoryData) error {
	var err error
	data.Usage, err = parseUintFromFile(path, prefix+".usage_in_bytes")
	if err != nil {
		return err
	}

	data.MaxUsage, err = parseUintFromFile(path, prefix+".max_usage_in_bytes")
	if err != nil {
		return err
	}

	data.Limit, err = parseUintFromFile(path, prefix+".limit_in_bytes")
	if err != nil {
		return err
	}

	data.FailCount, err = parseUintFromFile(path, prefix+".failcnt")
	if err != nil {
		return err
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
		t, v, err := parseCgroupParamKeyValue(sc.Text())
		if err != nil {
			return err
		}
		switch t {
		case "cache":
			mem.Stats.Cache = v
		case "rss":
			mem.Stats.RSS = v
		case "rss_huge":
			mem.Stats.RSSHuge = v
		case "mapped_file":
			mem.Stats.MappedFile = v
		case "pgpgin":
			mem.Stats.PagesIn = v
		case "pgpgout":
			mem.Stats.PagesOut = v
		case "pgfault":
			mem.Stats.PageFaults = v
		case "pgmajfault":
			mem.Stats.MajorPageFaults = v
		case "swap":
			mem.Stats.Swap = v
		case "active_anon":
			mem.Stats.ActiveAnon = v
		case "inactive_anon":
			mem.Stats.InactiveAnon = v
		case "active_file":
			mem.Stats.ActiveFile = v
		case "inactive_file":
			mem.Stats.InactiveFile = v
		case "unevictable":
			mem.Stats.Unevictable = v
		case "hierarchical_memory_limit":
			mem.Stats.HierarchicalMemoryLimit = v
		case "hierarchical_memsw_limit":
			mem.Stats.HierarchicalMemswLimit = v
		}
	}

	return sc.Err()
}
