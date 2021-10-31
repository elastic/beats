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

package types

import "time"

// Host is the interface that wraps methods for returning Host stats
type Host interface {
	CPUTimer
	Info() HostInfo
	Memory() (*HostMemoryInfo, error)
}

// NetworkCounters represents network stats from /proc/net
type NetworkCounters interface {
	NetworkCounters() (*NetworkCountersInfo, error)
}

// SNMP represents the data from /proc/net/snmp
// Note that according to RFC 2012,TCP.MaxConn, if present, is a signed value and should be cast to int64
type SNMP struct {
	IP      map[string]uint64 `json:"ip" netstat:"Ip"`
	ICMP    map[string]uint64 `json:"icmp" netstat:"Icmp"`
	ICMPMsg map[string]uint64 `json:"icmp_msg" netstat:"IcmpMsg"`
	TCP     map[string]uint64 `json:"tcp" netstat:"Tcp"`
	UDP     map[string]uint64 `json:"udp" netstat:"Udp"`
	UDPLite map[string]uint64 `json:"udp_lite" netstat:"UdpLite"`
}

// Netstat represents the data from /proc/net/netstat
type Netstat struct {
	TCPExt map[string]uint64 `json:"tcp_ext" netstat:"TcpExt"`
	IPExt  map[string]uint64 `json:"ip_ext" netstat:"IpExt"`
}

// NetworkCountersInfo represents available network counters from /proc/net
type NetworkCountersInfo struct {
	SNMP    SNMP    `json:"snmp"`
	Netstat Netstat `json:"netstat"`
}

// VMStat is the interface wrapper for platforms that support /proc/vmstat.
type VMStat interface {
	VMStat() (*VMStatInfo, error)
}

// HostInfo contains basic host information.
type HostInfo struct {
	Architecture      string    `json:"architecture"`            // Hardware architecture (e.g. x86_64, arm, ppc, mips).
	BootTime          time.Time `json:"boot_time"`               // Host boot time.
	Containerized     *bool     `json:"containerized,omitempty"` // Is the process containerized.
	Hostname          string    `json:"name"`                    // Hostname
	IPs               []string  `json:"ip,omitempty"`            // List of all IPs.
	KernelVersion     string    `json:"kernel_version"`          // Kernel version.
	MACs              []string  `json:"mac"`                     // List of MAC addresses.
	OS                *OSInfo   `json:"os"`                      // OS information.
	Timezone          string    `json:"timezone"`                // System timezone.
	TimezoneOffsetSec int       `json:"timezone_offset_sec"`     // Timezone offset (seconds from UTC).
	UniqueID          string    `json:"id,omitempty"`            // Unique ID of the host (optional).
}

// Uptime returns the system uptime
func (host HostInfo) Uptime() time.Duration {
	return time.Since(host.BootTime)
}

// OSInfo contains basic OS information
type OSInfo struct {
	Type     string `json:"type"`               // OS Type (one of linux, macos, unix, windows).
	Family   string `json:"family"`             // OS Family (e.g. redhat, debian, freebsd, windows).
	Platform string `json:"platform"`           // OS platform (e.g. centos, ubuntu, windows).
	Name     string `json:"name"`               // OS Name (e.g. Mac OS X, CentOS).
	Version  string `json:"version"`            // OS version (e.g. 10.12.6).
	Major    int    `json:"major"`              // Major release version.
	Minor    int    `json:"minor"`              // Minor release version.
	Patch    int    `json:"patch"`              // Patch release version.
	Build    string `json:"build,omitempty"`    // Build (e.g. 16G1114).
	Codename string `json:"codename,omitempty"` // OS codename (e.g. jessie).
}

// LoadAverage is the interface that wraps the LoadAverage method.
// LoadAverage returns load info on the host
type LoadAverage interface {
	LoadAverage() LoadAverageInfo
}

// LoadAverageInfo contains load statistics
type LoadAverageInfo struct {
	One     float64 `json:"one_min"`
	Five    float64 `json:"five_min"`
	Fifteen float64 `json:"fifteen_min"`
}

// HostMemoryInfo (all values are specified in bytes).
type HostMemoryInfo struct {
	Total        uint64            `json:"total_bytes"`         // Total physical memory.
	Used         uint64            `json:"used_bytes"`          // Total - Free
	Available    uint64            `json:"available_bytes"`     // Amount of memory available without swapping.
	Free         uint64            `json:"free_bytes"`          // Amount of memory not used by the system.
	VirtualTotal uint64            `json:"virtual_total_bytes"` // Total virtual memory.
	VirtualUsed  uint64            `json:"virtual_used_bytes"`  // VirtualTotal - VirtualFree
	VirtualFree  uint64            `json:"virtual_free_bytes"`  // Virtual memory that is not used.
	Metrics      map[string]uint64 `json:"raw,omitempty"`       // Other memory related metrics.
}

// VMStatInfo contains parsed info from /proc/vmstat.
// This procfs file has expanded much over the years
// with different kernel versions. If we don't have a field in vmstat,
// the field in the struct will just be blank. The comments represent kernel versions.
type VMStatInfo struct {
	NrFreePages                uint64 `json:"nr_free_pages"`                 // (since Linux 2.6.31)
	NrAllocBatch               uint64 `json:"nr_alloc_batch"`                // (since Linux 3.12)
	NrInactiveAnon             uint64 `json:"nr_inactive_anon"`              // (since Linux 2.6.28)
	NrActiveAnon               uint64 `json:"nr_active_anon"`                // (since Linux 2.6.28)
	NrInactiveFile             uint64 `json:"nr_inactive_file"`              // (since Linux 2.6.28)
	NrActiveFile               uint64 `json:"nr_active_file"`                // (since Linux 2.6.28)
	NrUnevictable              uint64 `json:"nr_unevictable"`                // (since Linux 2.6.28)
	NrMlock                    uint64 `json:"nr_mlock"`                      // (since Linux 2.6.28)
	NrAnonPages                uint64 `json:"nr_anon_pages"`                 // (since Linux 2.6.18)
	NrMapped                   uint64 `json:"nr_mapped"`                     // (since Linux 2.6.0)
	NrFilePages                uint64 `json:"nr_file_pages"`                 // (since Linux 2.6.18)
	NrDirty                    uint64 `json:"nr_dirty"`                      // (since Linux 2.6.0)
	NrWriteback                uint64 `json:"nr_writeback"`                  // (since Linux 2.6.0)
	NrSlabReclaimable          uint64 `json:"nr_slab_reclaimable"`           // (since Linux 2.6.19)
	NrSlabUnreclaimable        uint64 `json:"nr_slab_unreclaimable"`         // (since Linux 2.6.19)
	NrPageTablePages           uint64 `json:"nr_page_table_pages"`           // (since Linux 2.6.0)
	NrKernelStack              uint64 `json:"nr_kernel_stack"`               // (since Linux 2.6.32)  Amount of memory allocated to kernel stacks.
	NrUnstable                 uint64 `json:"nr_unstable"`                   // (since Linux 2.6.0)
	NrBounce                   uint64 `json:"nr_bounce"`                     // (since Linux 2.6.12)
	NrVmscanWrite              uint64 `json:"nr_vmscan_write"`               // (since Linux 2.6.19)
	NrVmscanImmediateReclaim   uint64 `json:"nr_vmscan_immediate_reclaim"`   // (since Linux 3.2)
	NrWritebackTemp            uint64 `json:"nr_writeback_temp"`             // (since Linux 2.6.26)
	NrIsolatedAnon             uint64 `json:"nr_isolated_anon"`              // (since Linux 2.6.32)
	NrIsolatedFile             uint64 `json:"nr_isolated_file"`              // (since Linux 2.6.32)
	NrShmem                    uint64 `json:"nr_shmem"`                      // (since Linux 2.6.32) Pages used by shmem and tmpfs(5).
	NrDirtied                  uint64 `json:"nr_dirtied"`                    // (since Linux 2.6.37)
	NrWritten                  uint64 `json:"nr_written"`                    // (since Linux 2.6.37)
	NrPagesScanned             uint64 `json:"nr_pages_scanned"`              // (since Linux 3.17)
	NumaHit                    uint64 `json:"numa_hit"`                      // (since Linux 2.6.18)
	NumaMiss                   uint64 `json:"numa_miss"`                     // (since Linux 2.6.18)
	NumaForeign                uint64 `json:"numa_foreign"`                  // (since Linux 2.6.18)
	NumaInterleave             uint64 `json:"numa_interleave"`               // (since Linux 2.6.18)
	NumaLocal                  uint64 `json:"numa_local"`                    // (since Linux 2.6.18)
	NumaOther                  uint64 `json:"numa_other"`                    // (since Linux 2.6.18)
	WorkingsetRefault          uint64 `json:"workingset_refault"`            // (since Linux 3.15)
	WorkingsetActivate         uint64 `json:"workingset_activate"`           // (since Linux 3.15)
	WorkingsetNodereclaim      uint64 `json:"workingset_nodereclaim"`        // (since Linux 3.15)
	NrAnonTransparentHugepages uint64 `json:"nr_anon_transparent_hugepages"` // (since Linux 2.6.38)
	NrFreeCma                  uint64 `json:"nr_free_cma"`                   // (since Linux 3.7)  Number of free CMA (Contiguous Memory Allocator) pages.
	NrDirtyThreshold           uint64 `json:"nr_dirty_threshold"`            // (since Linux 2.6.37)
	NrDirtyBackgroundThreshold uint64 `json:"nr_dirty_background_threshold"` // (since Linux 2.6.37)
	Pgpgin                     uint64 `json:"pgpgin"`                        // (since Linux 2.6.0)
	Pgpgout                    uint64 `json:"pgpgout"`                       // (since Linux 2.6.0)
	Pswpin                     uint64 `json:"pswpin"`                        // (since Linux 2.6.0)
	Pswpout                    uint64 `json:"pswpout"`                       // (since Linux 2.6.0)
	PgallocDma                 uint64 `json:"pgalloc_dma"`                   // (since Linux 2.6.5)
	PgallocDma32               uint64 `json:"pgalloc_dma32"`                 // (since Linux 2.6.16)
	PgallocNormal              uint64 `json:"pgalloc_normal"`                // (since Linux 2.6.5)
	PgallocHigh                uint64 `json:"pgalloc_high"`                  // (since Linux 2.6.5)
	PgallocMovable             uint64 `json:"pgalloc_movable"`               // (since Linux 2.6.23)
	Pgfree                     uint64 `json:"pgfree"`                        // (since Linux 2.6.0)
	Pgactivate                 uint64 `json:"pgactivate"`                    // (since Linux 2.6.0)
	Pgdeactivate               uint64 `json:"pgdeactivate"`                  // (since Linux 2.6.0)
	Pgfault                    uint64 `json:"pgfault"`                       // (since Linux 2.6.0)
	Pgmajfault                 uint64 `json:"pgmajfault"`                    // (since Linux 2.6.0)
	PgrefillDma                uint64 `json:"pgrefill_dma"`                  // (since Linux 2.6.5)
	PgrefillDma32              uint64 `json:"pgrefill_dma32"`                // (since Linux 2.6.16)
	PgrefillNormal             uint64 `json:"pgrefill_normal"`               // (since Linux 2.6.5)
	PgrefillHigh               uint64 `json:"pgrefill_high"`                 // (since Linux 2.6.5)
	PgrefillMovable            uint64 `json:"pgrefill_movable"`              // (since Linux 2.6.23)
	PgstealKswapdDma           uint64 `json:"pgsteal_kswapd_dma"`            // (since Linux 3.4)
	PgstealKswapdDma32         uint64 `json:"pgsteal_kswapd_dma32"`          // (since Linux 3.4)
	PgstealKswapdNormal        uint64 `json:"pgsteal_kswapd_normal"`         // (since Linux 3.4)
	PgstealKswapdHigh          uint64 `json:"pgsteal_kswapd_high"`           // (since Linux 3.4)
	PgstealKswapdMovable       uint64 `json:"pgsteal_kswapd_movable"`        // (since Linux 3.4)
	PgstealDirectDma           uint64 `json:"pgsteal_direct_dma"`
	PgstealDirectDma32         uint64 `json:"pgsteal_direct_dma32"`   // (since Linux 3.4)
	PgstealDirectNormal        uint64 `json:"pgsteal_direct_normal"`  // (since Linux 3.4)
	PgstealDirectHigh          uint64 `json:"pgsteal_direct_high"`    // (since Linux 3.4)
	PgstealDirectMovable       uint64 `json:"pgsteal_direct_movable"` // (since Linux 2.6.23)
	PgscanKswapdDma            uint64 `json:"pgscan_kswapd_dma"`
	PgscanKswapdDma32          uint64 `json:"pgscan_kswapd_dma32"`  // (since Linux 2.6.16)
	PgscanKswapdNormal         uint64 `json:"pgscan_kswapd_normal"` // (since Linux 2.6.5)
	PgscanKswapdHigh           uint64 `json:"pgscan_kswapd_high"`
	PgscanKswapdMovable        uint64 `json:"pgscan_kswapd_movable"` // (since Linux 2.6.23)
	PgscanDirectDma            uint64 `json:"pgscan_direct_dma"`     //
	PgscanDirectDma32          uint64 `json:"pgscan_direct_dma32"`   // (since Linux 2.6.16)
	PgscanDirectNormal         uint64 `json:"pgscan_direct_normal"`
	PgscanDirectHigh           uint64 `json:"pgscan_direct_high"`
	PgscanDirectMovable        uint64 `json:"pgscan_direct_movable"`         // (since Linux 2.6.23)
	PgscanDirectThrottle       uint64 `json:"pgscan_direct_throttle"`        // (since Linux 3.6)
	ZoneReclaimFailed          uint64 `json:"zone_reclaim_failed"`           // (since linux 2.6.31)
	Pginodesteal               uint64 `json:"pginodesteal"`                  // (since linux 2.6.0)
	SlabsScanned               uint64 `json:"slabs_scanned"`                 // (since linux 2.6.5)
	KswapdInodesteal           uint64 `json:"kswapd_inodesteal"`             // (since linux 2.6.0)
	KswapdLowWmarkHitQuickly   uint64 `json:"kswapd_low_wmark_hit_quickly"`  // (since 2.6.33)
	KswapdHighWmarkHitQuickly  uint64 `json:"kswapd_high_wmark_hit_quickly"` // (since 2.6.33)
	Pageoutrun                 uint64 `json:"pageoutrun"`                    // (since Linux 2.6.0)
	Allocstall                 uint64 `json:"allocstall"`                    // (since Linux 2.6.0)
	Pgrotated                  uint64 `json:"pgrotated"`                     // (since Linux 2.6.0)
	DropPagecache              uint64 `json:"drop_pagecache"`                // (since Linux 3.15)
	DropSlab                   uint64 `json:"drop_slab"`                     // (since Linux 3.15)
	NumaPteUpdates             uint64 `json:"numa_pte_updates"`              // (since Linux 3.8)
	NumaHugePteUpdates         uint64 `json:"numa_huge_pte_updates"`         // (since Linux 3.13)
	NumaHintFaults             uint64 `json:"numa_hint_faults"`              // (since Linux 3.8)
	NumaHintFaultsLocal        uint64 `json:"numa_hint_faults_local"`        // (since Linux 3.8)
	NumaPagesMigrated          uint64 `json:"numa_pages_migrated"`           // (since Linux 3.8)
	PgmigrateSuccess           uint64 `json:"pgmigrate_success"`             // (since Linux 3.8)
	PgmigrateFail              uint64 `json:"pgmigrate_fail"`                // (since Linux 3.8)
	CompactMigrateScanned      uint64 `json:"compact_migrate_scanned"`       // (since Linux 3.8)
	CompactFreeScanned         uint64 `json:"compact_free_scanned"`          // (since Linux 3.8)
	CompactIsolated            uint64 `json:"compact_isolated"`              // (since Linux 3.8)
	CompactStall               uint64 `json:"compact_stall"`                 // (since Linux 2.6.35) See the kernel source file Documentation/admin-guide/mm/transhuge.rst.
	CompactFail                uint64 `json:"compact_fail"`                  // (since Linux 2.6.35) See the kernel source file Documentation/admin-guide/mm/transhuge.rst.
	CompactSuccess             uint64 `json:"compact_success"`               // (since Linux 2.6.35) See the kernel source file Documentation/admin-guide/mm/transhuge.rst.
	HtlbBuddyAllocSuccess      uint64 `json:"htlb_buddy_alloc_success"`      // (since Linux 2.6.26)
	HtlbBuddyAllocFail         uint64 `json:"htlb_buddy_alloc_fail"`         // (since Linux 2.6.26)
	UnevictablePgsCulled       uint64 `json:"unevictable_pgs_culled"`        // (since Linux 2.6.28)
	UnevictablePgsScanned      uint64 `json:"unevictable_pgs_scanned"`       // (since Linux 2.6.28)
	UnevictablePgsRescued      uint64 `json:"unevictable_pgs_rescued"`       // (since Linux 2.6.28)
	UnevictablePgsMlocked      uint64 `json:"unevictable_pgs_mlocked"`       // (since Linux 2.6.28)
	UnevictablePgsMunlocked    uint64 `json:"unevictable_pgs_munlocked"`     // (since Linux 2.6.28)
	UnevictablePgsCleared      uint64 `json:"unevictable_pgs_cleared"`       // (since Linux 2.6.28)
	UnevictablePgsStranded     uint64 `json:"unevictable_pgs_stranded"`      // (since Linux 2.6.28)
	ThpFaultAlloc              uint64 `json:"thp_fault_alloc"`               // (since Linux 2.6.39) See the kernel source file Documentation/admin-guide/mm/transhuge.rst.
	ThpFaultFallback           uint64 `json:"thp_fault_fallback"`            // (since Linux 2.6.39) See the kernel source file Documentation/admin-guide/mm/transhuge.rst.
	ThpCollapseAlloc           uint64 `json:"thp_collapse_alloc"`            // (since Linux 2.6.39) See the kernel source file Documentation/admin-guide/mm/transhuge.rst.
	ThpCollapseAllocFailed     uint64 `json:"thp_collapse_alloc_failed"`     // (since Linux 2.6.39) See the kernel source file Documentation/admin-guide/mm/transhuge.rst.
	ThpSplit                   uint64 `json:"thp_split"`                     // (since Linux 2.6.39) See the kernel source file Documentation/admin-guide/mm/transhuge.rst.
	ThpZeroPageAlloc           uint64 `json:"thp_zero_page_alloc"`           // (since Linux 3.8) See the kernel source file Documentation/admin-guide/mm/transhuge.rst.
	ThpZeroPageAllocFailed     uint64 `json:"thp_zero_page_alloc_failed"`    // (since Linux 3.8) See the kernel source file Documentation/admin-guide/mm/transhuge.rst.
	BalloonInflate             uint64 `json:"balloon_inflate"`               // (since Linux 3.18)
	BalloonDeflate             uint64 `json:"balloon_deflate"`               // (since Linux 3.18)
	BalloonMigrate             uint64 `json:"balloon_migrate"`               // (since Linux 3.18)
	NrTlbRemoteFlush           uint64 `json:"nr_tlb_remote_flush"`           // (since Linux 3.12)
	NrTlbRemoteFlushReceived   uint64 `json:"nr_tlb_remote_flush_received"`  // (since Linux 3.12)
	NrTlbLocalFlushAll         uint64 `json:"nr_tlb_local_flush_all"`        // (since Linux 3.12)
	NrTlbLocalFlushOne         uint64 `json:"nr_tlb_local_flush_one"`        // (since Linux 3.12)
	VmacacheFindCalls          uint64 `json:"vmacache_find_calls"`           // (since Linux 3.16)
	VmacacheFindHits           uint64 `json:"vmacache_find_hits"`            // (since Linux 3.16)
	VmacacheFullFlushes        uint64 `json:"vmacache_full_flushes"`         // (since Linux 3.19)
	// the following fields are not documented in `man 5 proc` as of 4.15
	NrZoneInactiveAnon          uint64 `json:"nr_zone_inactive_anon"`
	NrZoneActiveAnon            uint64 `json:"nr_zone_active_anon"`
	NrZoneInactiveFile          uint64 `json:"nr_zone_inactive_file"`
	NrZoneActiveFile            uint64 `json:"nr_zone_active_file"`
	NrZoneUnevictable           uint64 `json:"nr_zone_unevictable"`
	NrZoneWritePending          uint64 `json:"nr_zone_write_pending"`
	NrZspages                   uint64 `json:"nr_zspages"`
	NrShmemHugepages            uint64 `json:"nr_shmem_hugepages"`
	NrShmemPmdmapped            uint64 `json:"nr_shmem_pmdmapped"`
	AllocstallDma               uint64 `json:"allocstall_dma"`
	AllocstallDma32             uint64 `json:"allocstall_dma32"`
	AllocstallNormal            uint64 `json:"allocstall_normal"`
	AllocstallMovable           uint64 `json:"allocstall_movable"`
	PgskipDma                   uint64 `json:"pgskip_dma"`
	PgskipDma32                 uint64 `json:"pgskip_dma32"`
	PgskipNormal                uint64 `json:"pgskip_normal"`
	PgskipMovable               uint64 `json:"pgskip_movable"`
	Pglazyfree                  uint64 `json:"pglazyfree"`
	Pglazyfreed                 uint64 `json:"pglazyfreed"`
	Pgrefill                    uint64 `json:"pgrefill"`
	PgstealKswapd               uint64 `json:"pgsteal_kswapd"`
	PgstealDirect               uint64 `json:"pgsteal_direct"`
	PgscanKswapd                uint64 `json:"pgscan_kswapd"`
	PgscanDirect                uint64 `json:"pgscan_direct"`
	OomKill                     uint64 `json:"oom_kill"`
	CompactDaemonWake           uint64 `json:"compact_daemon_wake"`
	CompactDaemonMigrateScanned uint64 `json:"compact_daemon_migrate_scanned"`
	CompactDaemonFreeScanned    uint64 `json:"compact_daemon_free_scanned"`
	ThpFileAlloc                uint64 `json:"thp_file_alloc"`
	ThpFileMapped               uint64 `json:"thp_file_mapped"`
	ThpSplitPage                uint64 `json:"thp_split_page"`
	ThpSplitPageFailed          uint64 `json:"thp_split_page_failed"`
	ThpDeferredSplitPage        uint64 `json:"thp_deferred_split_page"`
	ThpSplitPmd                 uint64 `json:"thp_split_pmd"`
	ThpSplitPud                 uint64 `json:"thp_split_pud"`
	ThpSwpout                   uint64 `json:"thp_swpout"`
	ThpSwpoutFallback           uint64 `json:"thp_swpout_fallback"`
	SwapRa                      uint64 `json:"swap_ra"`
	SwapRaHit                   uint64 `json:"swap_ra_hit"`
}
