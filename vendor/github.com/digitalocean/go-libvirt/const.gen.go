// Copyright 2017 The go-libvirt Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

// WARNING: This file has automatically been generated
// by https://git.io/c-for-go. DO NOT EDIT.

package libvirt

const (
	// ExportVar as defined in libvirt/libvirt-common.h:58
	ExportVar = 0
	// TypedParamFieldLength as defined in libvirt/libvirt-common.h:171
	TypedParamFieldLength = 80
	// SecurityLabelBuflen as defined in libvirt/libvirt-host.h:85
	SecurityLabelBuflen = 4097
	// SecurityModelBuflen as defined in libvirt/libvirt-host.h:113
	SecurityModelBuflen = 257
	// SecurityDoiBuflen as defined in libvirt/libvirt-host.h:120
	SecurityDoiBuflen = 257
	// NodeCPUStatsFieldLength as defined in libvirt/libvirt-host.h:177
	NodeCPUStatsFieldLength = 80
	// NodeCPUStatsKernel as defined in libvirt/libvirt-host.h:194
	NodeCPUStatsKernel = "kernel"
	// NodeCPUStatsUser as defined in libvirt/libvirt-host.h:202
	NodeCPUStatsUser = "user"
	// NodeCPUStatsIdle as defined in libvirt/libvirt-host.h:210
	NodeCPUStatsIdle = "idle"
	// NodeCPUStatsIowait as defined in libvirt/libvirt-host.h:218
	NodeCPUStatsIowait = "iowait"
	// NodeCPUStatsIntr as defined in libvirt/libvirt-host.h:226
	NodeCPUStatsIntr = "intr"
	// NodeCPUStatsUtilization as defined in libvirt/libvirt-host.h:235
	NodeCPUStatsUtilization = "utilization"
	// NodeMemoryStatsFieldLength as defined in libvirt/libvirt-host.h:255
	NodeMemoryStatsFieldLength = 80
	// NodeMemoryStatsTotal as defined in libvirt/libvirt-host.h:272
	NodeMemoryStatsTotal = "total"
	// NodeMemoryStatsFree as defined in libvirt/libvirt-host.h:281
	NodeMemoryStatsFree = "free"
	// NodeMemoryStatsBuffers as defined in libvirt/libvirt-host.h:289
	NodeMemoryStatsBuffers = "buffers"
	// NodeMemoryStatsCached as defined in libvirt/libvirt-host.h:297
	NodeMemoryStatsCached = "cached"
	// NodeMemorySharedPagesToScan as defined in libvirt/libvirt-host.h:318
	NodeMemorySharedPagesToScan = "shm_pages_to_scan"
	// NodeMemorySharedSleepMillisecs as defined in libvirt/libvirt-host.h:326
	NodeMemorySharedSleepMillisecs = "shm_sleep_millisecs"
	// NodeMemorySharedPagesShared as defined in libvirt/libvirt-host.h:334
	NodeMemorySharedPagesShared = "shm_pages_shared"
	// NodeMemorySharedPagesSharing as defined in libvirt/libvirt-host.h:342
	NodeMemorySharedPagesSharing = "shm_pages_sharing"
	// NodeMemorySharedPagesUnshared as defined in libvirt/libvirt-host.h:350
	NodeMemorySharedPagesUnshared = "shm_pages_unshared"
	// NodeMemorySharedPagesVolatile as defined in libvirt/libvirt-host.h:358
	NodeMemorySharedPagesVolatile = "shm_pages_volatile"
	// NodeMemorySharedFullScans as defined in libvirt/libvirt-host.h:366
	NodeMemorySharedFullScans = "shm_full_scans"
	// NodeMemorySharedMergeAcrossNodes as defined in libvirt/libvirt-host.h:378
	NodeMemorySharedMergeAcrossNodes = "shm_merge_across_nodes"
	// UUIDBuflen as defined in libvirt/libvirt-host.h:513
	UUIDBuflen = 16
	// UUIDStringBuflen as defined in libvirt/libvirt-host.h:522
	UUIDStringBuflen = 37
	// DomainSchedulerCPUShares as defined in libvirt/libvirt-domain.h:315
	DomainSchedulerCPUShares = "cpu_shares"
	// DomainSchedulerGlobalPeriod as defined in libvirt/libvirt-domain.h:323
	DomainSchedulerGlobalPeriod = "global_period"
	// DomainSchedulerGlobalQuota as defined in libvirt/libvirt-domain.h:331
	DomainSchedulerGlobalQuota = "global_quota"
	// DomainSchedulerVCPUPeriod as defined in libvirt/libvirt-domain.h:339
	DomainSchedulerVCPUPeriod = "vcpu_period"
	// DomainSchedulerVCPUQuota as defined in libvirt/libvirt-domain.h:347
	DomainSchedulerVCPUQuota = "vcpu_quota"
	// DomainSchedulerEmulatorPeriod as defined in libvirt/libvirt-domain.h:356
	DomainSchedulerEmulatorPeriod = "emulator_period"
	// DomainSchedulerEmulatorQuota as defined in libvirt/libvirt-domain.h:365
	DomainSchedulerEmulatorQuota = "emulator_quota"
	// DomainSchedulerIothreadPeriod as defined in libvirt/libvirt-domain.h:373
	DomainSchedulerIothreadPeriod = "iothread_period"
	// DomainSchedulerIothreadQuota as defined in libvirt/libvirt-domain.h:381
	DomainSchedulerIothreadQuota = "iothread_quota"
	// DomainSchedulerWeight as defined in libvirt/libvirt-domain.h:389
	DomainSchedulerWeight = "weight"
	// DomainSchedulerCap as defined in libvirt/libvirt-domain.h:397
	DomainSchedulerCap = "cap"
	// DomainSchedulerReservation as defined in libvirt/libvirt-domain.h:405
	DomainSchedulerReservation = "reservation"
	// DomainSchedulerLimit as defined in libvirt/libvirt-domain.h:413
	DomainSchedulerLimit = "limit"
	// DomainSchedulerShares as defined in libvirt/libvirt-domain.h:421
	DomainSchedulerShares = "shares"
	// DomainBlockStatsFieldLength as defined in libvirt/libvirt-domain.h:479
	DomainBlockStatsFieldLength = 80
	// DomainBlockStatsReadBytes as defined in libvirt/libvirt-domain.h:487
	DomainBlockStatsReadBytes = "rd_bytes"
	// DomainBlockStatsReadReq as defined in libvirt/libvirt-domain.h:495
	DomainBlockStatsReadReq = "rd_operations"
	// DomainBlockStatsReadTotalTimes as defined in libvirt/libvirt-domain.h:503
	DomainBlockStatsReadTotalTimes = "rd_total_times"
	// DomainBlockStatsWriteBytes as defined in libvirt/libvirt-domain.h:511
	DomainBlockStatsWriteBytes = "wr_bytes"
	// DomainBlockStatsWriteReq as defined in libvirt/libvirt-domain.h:519
	DomainBlockStatsWriteReq = "wr_operations"
	// DomainBlockStatsWriteTotalTimes as defined in libvirt/libvirt-domain.h:527
	DomainBlockStatsWriteTotalTimes = "wr_total_times"
	// DomainBlockStatsFlushReq as defined in libvirt/libvirt-domain.h:535
	DomainBlockStatsFlushReq = "flush_operations"
	// DomainBlockStatsFlushTotalTimes as defined in libvirt/libvirt-domain.h:543
	DomainBlockStatsFlushTotalTimes = "flush_total_times"
	// DomainBlockStatsErrs as defined in libvirt/libvirt-domain.h:550
	DomainBlockStatsErrs = "errs"
	// MigrateParamURI as defined in libvirt/libvirt-domain.h:842
	MigrateParamURI = "migrate_uri"
	// MigrateParamDestName as defined in libvirt/libvirt-domain.h:852
	MigrateParamDestName = "destination_name"
	// MigrateParamDestXML as defined in libvirt/libvirt-domain.h:871
	MigrateParamDestXML = "destination_xml"
	// MigrateParamPersistXML as defined in libvirt/libvirt-domain.h:886
	MigrateParamPersistXML = "persistent_xml"
	// MigrateParamBandwidth as defined in libvirt/libvirt-domain.h:896
	MigrateParamBandwidth = "bandwidth"
	// MigrateParamGraphicsURI as defined in libvirt/libvirt-domain.h:917
	MigrateParamGraphicsURI = "graphics_uri"
	// MigrateParamListenAddress as defined in libvirt/libvirt-domain.h:928
	MigrateParamListenAddress = "listen_address"
	// MigrateParamMigrateDisks as defined in libvirt/libvirt-domain.h:937
	MigrateParamMigrateDisks = "migrate_disks"
	// MigrateParamDisksPort as defined in libvirt/libvirt-domain.h:947
	MigrateParamDisksPort = "disks_port"
	// MigrateParamCompression as defined in libvirt/libvirt-domain.h:957
	MigrateParamCompression = "compression"
	// MigrateParamCompressionMtLevel as defined in libvirt/libvirt-domain.h:966
	MigrateParamCompressionMtLevel = "compression.mt.level"
	// MigrateParamCompressionMtThreads as defined in libvirt/libvirt-domain.h:974
	MigrateParamCompressionMtThreads = "compression.mt.threads"
	// MigrateParamCompressionMtDthreads as defined in libvirt/libvirt-domain.h:982
	MigrateParamCompressionMtDthreads = "compression.mt.dthreads"
	// MigrateParamCompressionXbzrleCache as defined in libvirt/libvirt-domain.h:990
	MigrateParamCompressionXbzrleCache = "compression.xbzrle.cache"
	// MigrateParamAutoConvergeInitial as defined in libvirt/libvirt-domain.h:999
	MigrateParamAutoConvergeInitial = "auto_converge.initial"
	// MigrateParamAutoConvergeIncrement as defined in libvirt/libvirt-domain.h:1009
	MigrateParamAutoConvergeIncrement = "auto_converge.increment"
	// DomainCPUStatsCputime as defined in libvirt/libvirt-domain.h:1252
	DomainCPUStatsCputime = "cpu_time"
	// DomainCPUStatsUsertime as defined in libvirt/libvirt-domain.h:1258
	DomainCPUStatsUsertime = "user_time"
	// DomainCPUStatsSystemtime as defined in libvirt/libvirt-domain.h:1264
	DomainCPUStatsSystemtime = "system_time"
	// DomainCPUStatsVcputime as defined in libvirt/libvirt-domain.h:1271
	DomainCPUStatsVcputime = "vcpu_time"
	// DomainBlkioWeight as defined in libvirt/libvirt-domain.h:1300
	DomainBlkioWeight = "weight"
	// DomainBlkioDeviceWeight as defined in libvirt/libvirt-domain.h:1310
	DomainBlkioDeviceWeight = "device_weight"
	// DomainBlkioDeviceReadIops as defined in libvirt/libvirt-domain.h:1321
	DomainBlkioDeviceReadIops = "device_read_iops_sec"
	// DomainBlkioDeviceWriteIops as defined in libvirt/libvirt-domain.h:1332
	DomainBlkioDeviceWriteIops = "device_write_iops_sec"
	// DomainBlkioDeviceReadBps as defined in libvirt/libvirt-domain.h:1343
	DomainBlkioDeviceReadBps = "device_read_bytes_sec"
	// DomainBlkioDeviceWriteBps as defined in libvirt/libvirt-domain.h:1354
	DomainBlkioDeviceWriteBps = "device_write_bytes_sec"
	// DomainMemoryParamUnlimited as defined in libvirt/libvirt-domain.h:1373
	DomainMemoryParamUnlimited = 9007199254740991
	// DomainMemoryHardLimit as defined in libvirt/libvirt-domain.h:1382
	DomainMemoryHardLimit = "hard_limit"
	// DomainMemorySoftLimit as defined in libvirt/libvirt-domain.h:1391
	DomainMemorySoftLimit = "soft_limit"
	// DomainMemoryMinGuarantee as defined in libvirt/libvirt-domain.h:1400
	DomainMemoryMinGuarantee = "min_guarantee"
	// DomainMemorySwapHardLimit as defined in libvirt/libvirt-domain.h:1410
	DomainMemorySwapHardLimit = "swap_hard_limit"
	// DomainNumaNodeset as defined in libvirt/libvirt-domain.h:1455
	DomainNumaNodeset = "numa_nodeset"
	// DomainNumaMode as defined in libvirt/libvirt-domain.h:1463
	DomainNumaMode = "numa_mode"
	// DomainBandwidthInAverage as defined in libvirt/libvirt-domain.h:1575
	DomainBandwidthInAverage = "inbound.average"
	// DomainBandwidthInPeak as defined in libvirt/libvirt-domain.h:1582
	DomainBandwidthInPeak = "inbound.peak"
	// DomainBandwidthInBurst as defined in libvirt/libvirt-domain.h:1589
	DomainBandwidthInBurst = "inbound.burst"
	// DomainBandwidthInFloor as defined in libvirt/libvirt-domain.h:1596
	DomainBandwidthInFloor = "inbound.floor"
	// DomainBandwidthOutAverage as defined in libvirt/libvirt-domain.h:1603
	DomainBandwidthOutAverage = "outbound.average"
	// DomainBandwidthOutPeak as defined in libvirt/libvirt-domain.h:1610
	DomainBandwidthOutPeak = "outbound.peak"
	// DomainBandwidthOutBurst as defined in libvirt/libvirt-domain.h:1617
	DomainBandwidthOutBurst = "outbound.burst"
	// PerfParamCmt as defined in libvirt/libvirt-domain.h:2073
	PerfParamCmt = "cmt"
	// PerfParamMbmt as defined in libvirt/libvirt-domain.h:2084
	PerfParamMbmt = "mbmt"
	// PerfParamMbml as defined in libvirt/libvirt-domain.h:2094
	PerfParamMbml = "mbml"
	// PerfParamCacheMisses as defined in libvirt/libvirt-domain.h:2104
	PerfParamCacheMisses = "cache_misses"
	// PerfParamCacheReferences as defined in libvirt/libvirt-domain.h:2114
	PerfParamCacheReferences = "cache_references"
	// PerfParamInstructions as defined in libvirt/libvirt-domain.h:2124
	PerfParamInstructions = "instructions"
	// PerfParamCPUCycles as defined in libvirt/libvirt-domain.h:2134
	PerfParamCPUCycles = "cpu_cycles"
	// PerfParamBranchInstructions as defined in libvirt/libvirt-domain.h:2144
	PerfParamBranchInstructions = "branch_instructions"
	// PerfParamBranchMisses as defined in libvirt/libvirt-domain.h:2154
	PerfParamBranchMisses = "branch_misses"
	// PerfParamBusCycles as defined in libvirt/libvirt-domain.h:2164
	PerfParamBusCycles = "bus_cycles"
	// PerfParamStalledCyclesFrontend as defined in libvirt/libvirt-domain.h:2175
	PerfParamStalledCyclesFrontend = "stalled_cycles_frontend"
	// PerfParamStalledCyclesBackend as defined in libvirt/libvirt-domain.h:2186
	PerfParamStalledCyclesBackend = "stalled_cycles_backend"
	// PerfParamRefCPUCycles as defined in libvirt/libvirt-domain.h:2197
	PerfParamRefCPUCycles = "ref_cpu_cycles"
	// PerfParamCPUClock as defined in libvirt/libvirt-domain.h:2208
	PerfParamCPUClock = "cpu_clock"
	// PerfParamTaskClock as defined in libvirt/libvirt-domain.h:2219
	PerfParamTaskClock = "task_clock"
	// PerfParamPageFaults as defined in libvirt/libvirt-domain.h:2229
	PerfParamPageFaults = "page_faults"
	// PerfParamContextSwitches as defined in libvirt/libvirt-domain.h:2239
	PerfParamContextSwitches = "context_switches"
	// PerfParamCPUMigrations as defined in libvirt/libvirt-domain.h:2249
	PerfParamCPUMigrations = "cpu_migrations"
	// PerfParamPageFaultsMin as defined in libvirt/libvirt-domain.h:2259
	PerfParamPageFaultsMin = "page_faults_min"
	// PerfParamPageFaultsMaj as defined in libvirt/libvirt-domain.h:2269
	PerfParamPageFaultsMaj = "page_faults_maj"
	// PerfParamAlignmentFaults as defined in libvirt/libvirt-domain.h:2279
	PerfParamAlignmentFaults = "alignment_faults"
	// PerfParamEmulationFaults as defined in libvirt/libvirt-domain.h:2289
	PerfParamEmulationFaults = "emulation_faults"
	// DomainBlockCopyBandwidth as defined in libvirt/libvirt-domain.h:2450
	DomainBlockCopyBandwidth = "bandwidth"
	// DomainBlockCopyGranularity as defined in libvirt/libvirt-domain.h:2461
	DomainBlockCopyGranularity = "granularity"
	// DomainBlockCopyBufSize as defined in libvirt/libvirt-domain.h:2470
	DomainBlockCopyBufSize = "buf-size"
	// DomainBlockIotuneTotalBytesSec as defined in libvirt/libvirt-domain.h:2511
	DomainBlockIotuneTotalBytesSec = "total_bytes_sec"
	// DomainBlockIotuneReadBytesSec as defined in libvirt/libvirt-domain.h:2519
	DomainBlockIotuneReadBytesSec = "read_bytes_sec"
	// DomainBlockIotuneWriteBytesSec as defined in libvirt/libvirt-domain.h:2527
	DomainBlockIotuneWriteBytesSec = "write_bytes_sec"
	// DomainBlockIotuneTotalIopsSec as defined in libvirt/libvirt-domain.h:2535
	DomainBlockIotuneTotalIopsSec = "total_iops_sec"
	// DomainBlockIotuneReadIopsSec as defined in libvirt/libvirt-domain.h:2543
	DomainBlockIotuneReadIopsSec = "read_iops_sec"
	// DomainBlockIotuneWriteIopsSec as defined in libvirt/libvirt-domain.h:2550
	DomainBlockIotuneWriteIopsSec = "write_iops_sec"
	// DomainBlockIotuneTotalBytesSecMax as defined in libvirt/libvirt-domain.h:2558
	DomainBlockIotuneTotalBytesSecMax = "total_bytes_sec_max"
	// DomainBlockIotuneReadBytesSecMax as defined in libvirt/libvirt-domain.h:2566
	DomainBlockIotuneReadBytesSecMax = "read_bytes_sec_max"
	// DomainBlockIotuneWriteBytesSecMax as defined in libvirt/libvirt-domain.h:2574
	DomainBlockIotuneWriteBytesSecMax = "write_bytes_sec_max"
	// DomainBlockIotuneTotalIopsSecMax as defined in libvirt/libvirt-domain.h:2582
	DomainBlockIotuneTotalIopsSecMax = "total_iops_sec_max"
	// DomainBlockIotuneReadIopsSecMax as defined in libvirt/libvirt-domain.h:2590
	DomainBlockIotuneReadIopsSecMax = "read_iops_sec_max"
	// DomainBlockIotuneWriteIopsSecMax as defined in libvirt/libvirt-domain.h:2597
	DomainBlockIotuneWriteIopsSecMax = "write_iops_sec_max"
	// DomainBlockIotuneTotalBytesSecMaxLength as defined in libvirt/libvirt-domain.h:2605
	DomainBlockIotuneTotalBytesSecMaxLength = "total_bytes_sec_max_length"
	// DomainBlockIotuneReadBytesSecMaxLength as defined in libvirt/libvirt-domain.h:2613
	DomainBlockIotuneReadBytesSecMaxLength = "read_bytes_sec_max_length"
	// DomainBlockIotuneWriteBytesSecMaxLength as defined in libvirt/libvirt-domain.h:2621
	DomainBlockIotuneWriteBytesSecMaxLength = "write_bytes_sec_max_length"
	// DomainBlockIotuneTotalIopsSecMaxLength as defined in libvirt/libvirt-domain.h:2629
	DomainBlockIotuneTotalIopsSecMaxLength = "total_iops_sec_max_length"
	// DomainBlockIotuneReadIopsSecMaxLength as defined in libvirt/libvirt-domain.h:2637
	DomainBlockIotuneReadIopsSecMaxLength = "read_iops_sec_max_length"
	// DomainBlockIotuneWriteIopsSecMaxLength as defined in libvirt/libvirt-domain.h:2645
	DomainBlockIotuneWriteIopsSecMaxLength = "write_iops_sec_max_length"
	// DomainBlockIotuneSizeIopsSec as defined in libvirt/libvirt-domain.h:2652
	DomainBlockIotuneSizeIopsSec = "size_iops_sec"
	// DomainBlockIotuneGroupName as defined in libvirt/libvirt-domain.h:2659
	DomainBlockIotuneGroupName = "group_name"
	// DomainSendKeyMaxKeys as defined in libvirt/libvirt-domain.h:2740
	DomainSendKeyMaxKeys = 16
	// DomainJobOperationStr as defined in libvirt/libvirt-domain.h:3143
	DomainJobOperationStr = "operation"
	// DomainJobTimeElapsed as defined in libvirt/libvirt-domain.h:3153
	DomainJobTimeElapsed = "time_elapsed"
	// DomainJobTimeElapsedNet as defined in libvirt/libvirt-domain.h:3163
	DomainJobTimeElapsedNet = "time_elapsed_net"
	// DomainJobTimeRemaining as defined in libvirt/libvirt-domain.h:3173
	DomainJobTimeRemaining = "time_remaining"
	// DomainJobDowntime as defined in libvirt/libvirt-domain.h:3183
	DomainJobDowntime = "downtime"
	// DomainJobDowntimeNet as defined in libvirt/libvirt-domain.h:3192
	DomainJobDowntimeNet = "downtime_net"
	// DomainJobSetupTime as defined in libvirt/libvirt-domain.h:3201
	DomainJobSetupTime = "setup_time"
	// DomainJobDataTotal as defined in libvirt/libvirt-domain.h:3216
	DomainJobDataTotal = "data_total"
	// DomainJobDataProcessed as defined in libvirt/libvirt-domain.h:3226
	DomainJobDataProcessed = "data_processed"
	// DomainJobDataRemaining as defined in libvirt/libvirt-domain.h:3236
	DomainJobDataRemaining = "data_remaining"
	// DomainJobMemoryTotal as defined in libvirt/libvirt-domain.h:3246
	DomainJobMemoryTotal = "memory_total"
	// DomainJobMemoryProcessed as defined in libvirt/libvirt-domain.h:3256
	DomainJobMemoryProcessed = "memory_processed"
	// DomainJobMemoryRemaining as defined in libvirt/libvirt-domain.h:3266
	DomainJobMemoryRemaining = "memory_remaining"
	// DomainJobMemoryConstant as defined in libvirt/libvirt-domain.h:3278
	DomainJobMemoryConstant = "memory_constant"
	// DomainJobMemoryNormal as defined in libvirt/libvirt-domain.h:3288
	DomainJobMemoryNormal = "memory_normal"
	// DomainJobMemoryNormalBytes as defined in libvirt/libvirt-domain.h:3298
	DomainJobMemoryNormalBytes = "memory_normal_bytes"
	// DomainJobMemoryBps as defined in libvirt/libvirt-domain.h:3306
	DomainJobMemoryBps = "memory_bps"
	// DomainJobMemoryDirtyRate as defined in libvirt/libvirt-domain.h:3314
	DomainJobMemoryDirtyRate = "memory_dirty_rate"
	// DomainJobMemoryIteration as defined in libvirt/libvirt-domain.h:3325
	DomainJobMemoryIteration = "memory_iteration"
	// DomainJobDiskTotal as defined in libvirt/libvirt-domain.h:3335
	DomainJobDiskTotal = "disk_total"
	// DomainJobDiskProcessed as defined in libvirt/libvirt-domain.h:3345
	DomainJobDiskProcessed = "disk_processed"
	// DomainJobDiskRemaining as defined in libvirt/libvirt-domain.h:3355
	DomainJobDiskRemaining = "disk_remaining"
	// DomainJobDiskBps as defined in libvirt/libvirt-domain.h:3363
	DomainJobDiskBps = "disk_bps"
	// DomainJobCompressionCache as defined in libvirt/libvirt-domain.h:3372
	DomainJobCompressionCache = "compression_cache"
	// DomainJobCompressionBytes as defined in libvirt/libvirt-domain.h:3380
	DomainJobCompressionBytes = "compression_bytes"
	// DomainJobCompressionPages as defined in libvirt/libvirt-domain.h:3388
	DomainJobCompressionPages = "compression_pages"
	// DomainJobCompressionCacheMisses as defined in libvirt/libvirt-domain.h:3397
	DomainJobCompressionCacheMisses = "compression_cache_misses"
	// DomainJobCompressionOverflow as defined in libvirt/libvirt-domain.h:3407
	DomainJobCompressionOverflow = "compression_overflow"
	// DomainJobAutoConvergeThrottle as defined in libvirt/libvirt-domain.h:3416
	DomainJobAutoConvergeThrottle = "auto_converge_throttle"
	// DomainTunableCPUVcpupin as defined in libvirt/libvirt-domain.h:3969
	DomainTunableCPUVcpupin = "cputune.vcpupin%u"
	// DomainTunableCPUEmulatorpin as defined in libvirt/libvirt-domain.h:3977
	DomainTunableCPUEmulatorpin = "cputune.emulatorpin"
	// DomainTunableCPUIothreadspin as defined in libvirt/libvirt-domain.h:3986
	DomainTunableCPUIothreadspin = "cputune.iothreadpin%u"
	// DomainTunableCPUCpuShares as defined in libvirt/libvirt-domain.h:3994
	DomainTunableCPUCpuShares = "cputune.cpu_shares"
	// DomainTunableCPUGlobalPeriod as defined in libvirt/libvirt-domain.h:4002
	DomainTunableCPUGlobalPeriod = "cputune.global_period"
	// DomainTunableCPUGlobalQuota as defined in libvirt/libvirt-domain.h:4010
	DomainTunableCPUGlobalQuota = "cputune.global_quota"
	// DomainTunableCPUVCPUPeriod as defined in libvirt/libvirt-domain.h:4018
	DomainTunableCPUVCPUPeriod = "cputune.vcpu_period"
	// DomainTunableCPUVCPUQuota as defined in libvirt/libvirt-domain.h:4026
	DomainTunableCPUVCPUQuota = "cputune.vcpu_quota"
	// DomainTunableCPUEmulatorPeriod as defined in libvirt/libvirt-domain.h:4035
	DomainTunableCPUEmulatorPeriod = "cputune.emulator_period"
	// DomainTunableCPUEmulatorQuota as defined in libvirt/libvirt-domain.h:4044
	DomainTunableCPUEmulatorQuota = "cputune.emulator_quota"
	// DomainTunableCPUIothreadPeriod as defined in libvirt/libvirt-domain.h:4052
	DomainTunableCPUIothreadPeriod = "cputune.iothread_period"
	// DomainTunableCPUIothreadQuota as defined in libvirt/libvirt-domain.h:4060
	DomainTunableCPUIothreadQuota = "cputune.iothread_quota"
	// DomainTunableBlkdevDisk as defined in libvirt/libvirt-domain.h:4068
	DomainTunableBlkdevDisk = "blkdeviotune.disk"
	// DomainTunableBlkdevTotalBytesSec as defined in libvirt/libvirt-domain.h:4076
	DomainTunableBlkdevTotalBytesSec = "blkdeviotune.total_bytes_sec"
	// DomainTunableBlkdevReadBytesSec as defined in libvirt/libvirt-domain.h:4084
	DomainTunableBlkdevReadBytesSec = "blkdeviotune.read_bytes_sec"
	// DomainTunableBlkdevWriteBytesSec as defined in libvirt/libvirt-domain.h:4092
	DomainTunableBlkdevWriteBytesSec = "blkdeviotune.write_bytes_sec"
	// DomainTunableBlkdevTotalIopsSec as defined in libvirt/libvirt-domain.h:4100
	DomainTunableBlkdevTotalIopsSec = "blkdeviotune.total_iops_sec"
	// DomainTunableBlkdevReadIopsSec as defined in libvirt/libvirt-domain.h:4108
	DomainTunableBlkdevReadIopsSec = "blkdeviotune.read_iops_sec"
	// DomainTunableBlkdevWriteIopsSec as defined in libvirt/libvirt-domain.h:4116
	DomainTunableBlkdevWriteIopsSec = "blkdeviotune.write_iops_sec"
	// DomainTunableBlkdevTotalBytesSecMax as defined in libvirt/libvirt-domain.h:4124
	DomainTunableBlkdevTotalBytesSecMax = "blkdeviotune.total_bytes_sec_max"
	// DomainTunableBlkdevReadBytesSecMax as defined in libvirt/libvirt-domain.h:4132
	DomainTunableBlkdevReadBytesSecMax = "blkdeviotune.read_bytes_sec_max"
	// DomainTunableBlkdevWriteBytesSecMax as defined in libvirt/libvirt-domain.h:4140
	DomainTunableBlkdevWriteBytesSecMax = "blkdeviotune.write_bytes_sec_max"
	// DomainTunableBlkdevTotalIopsSecMax as defined in libvirt/libvirt-domain.h:4148
	DomainTunableBlkdevTotalIopsSecMax = "blkdeviotune.total_iops_sec_max"
	// DomainTunableBlkdevReadIopsSecMax as defined in libvirt/libvirt-domain.h:4156
	DomainTunableBlkdevReadIopsSecMax = "blkdeviotune.read_iops_sec_max"
	// DomainTunableBlkdevWriteIopsSecMax as defined in libvirt/libvirt-domain.h:4164
	DomainTunableBlkdevWriteIopsSecMax = "blkdeviotune.write_iops_sec_max"
	// DomainTunableBlkdevSizeIopsSec as defined in libvirt/libvirt-domain.h:4172
	DomainTunableBlkdevSizeIopsSec = "blkdeviotune.size_iops_sec"
	// DomainTunableBlkdevGroupName as defined in libvirt/libvirt-domain.h:4180
	DomainTunableBlkdevGroupName = "blkdeviotune.group_name"
	// DomainTunableBlkdevTotalBytesSecMaxLength as defined in libvirt/libvirt-domain.h:4189
	DomainTunableBlkdevTotalBytesSecMaxLength = "blkdeviotune.total_bytes_sec_max_length"
	// DomainTunableBlkdevReadBytesSecMaxLength as defined in libvirt/libvirt-domain.h:4198
	DomainTunableBlkdevReadBytesSecMaxLength = "blkdeviotune.read_bytes_sec_max_length"
	// DomainTunableBlkdevWriteBytesSecMaxLength as defined in libvirt/libvirt-domain.h:4207
	DomainTunableBlkdevWriteBytesSecMaxLength = "blkdeviotune.write_bytes_sec_max_length"
	// DomainTunableBlkdevTotalIopsSecMaxLength as defined in libvirt/libvirt-domain.h:4216
	DomainTunableBlkdevTotalIopsSecMaxLength = "blkdeviotune.total_iops_sec_max_length"
	// DomainTunableBlkdevReadIopsSecMaxLength as defined in libvirt/libvirt-domain.h:4225
	DomainTunableBlkdevReadIopsSecMaxLength = "blkdeviotune.read_iops_sec_max_length"
	// DomainTunableBlkdevWriteIopsSecMaxLength as defined in libvirt/libvirt-domain.h:4234
	DomainTunableBlkdevWriteIopsSecMaxLength = "blkdeviotune.write_iops_sec_max_length"
	// DomainSchedFieldLength as defined in libvirt/libvirt-domain.h:4522
	DomainSchedFieldLength = 80
	// DomainBlkioFieldLength as defined in libvirt/libvirt-domain.h:4566
	DomainBlkioFieldLength = 80
	// DomainMemoryFieldLength as defined in libvirt/libvirt-domain.h:4610
	DomainMemoryFieldLength = 80
)

// ConnectCloseReason as declared in libvirt/libvirt-common.h:120
type ConnectCloseReason int32

// ConnectCloseReason enumeration from libvirt/libvirt-common.h:120
const (
	ConnectCloseReasonError     ConnectCloseReason = iota
	ConnectCloseReasonEOF       ConnectCloseReason = 1
	ConnectCloseReasonKeepalive ConnectCloseReason = 2
	ConnectCloseReasonClient    ConnectCloseReason = 3
)

// TypedParameterType as declared in libvirt/libvirt-common.h:139
type TypedParameterType int32

// TypedParameterType enumeration from libvirt/libvirt-common.h:139
const (
	TypedParamInt     TypedParameterType = 1
	TypedParamUint    TypedParameterType = 2
	TypedParamLlong   TypedParameterType = 3
	TypedParamUllong  TypedParameterType = 4
	TypedParamDouble  TypedParameterType = 5
	TypedParamBoolean TypedParameterType = 6
	TypedParamString  TypedParameterType = 7
)

// TypedParameterFlags as declared in libvirt/libvirt-common.h:164
type TypedParameterFlags int32

// TypedParameterFlags enumeration from libvirt/libvirt-common.h:164
const (
	TypedParamStringOkay TypedParameterFlags = 4
)

// NodeSuspendTarget as declared in libvirt/libvirt-host.h:62
type NodeSuspendTarget int32

// NodeSuspendTarget enumeration from libvirt/libvirt-host.h:62
const (
	NodeSuspendTargetMem    NodeSuspendTarget = iota
	NodeSuspendTargetDisk   NodeSuspendTarget = 1
	NodeSuspendTargetHybrid NodeSuspendTarget = 2
)

// NodeGetCPUStatsAllCPUs as declared in libvirt/libvirt-host.h:186
type NodeGetCPUStatsAllCPUs int32

// NodeGetCPUStatsAllCPUs enumeration from libvirt/libvirt-host.h:186
const (
	NodeCPUStatsAllCpus NodeGetCPUStatsAllCPUs = -1
)

// NodeGetMemoryStatsAllCells as declared in libvirt/libvirt-host.h:264
type NodeGetMemoryStatsAllCells int32

// NodeGetMemoryStatsAllCells enumeration from libvirt/libvirt-host.h:264
const (
	NodeMemoryStatsAllCells NodeGetMemoryStatsAllCells = -1
)

// ConnectFlags as declared in libvirt/libvirt-host.h:443
type ConnectFlags int32

// ConnectFlags enumeration from libvirt/libvirt-host.h:443
const (
	ConnectRo        ConnectFlags = 1
	ConnectNoAliases ConnectFlags = 2
)

// ConnectCredentialType as declared in libvirt/libvirt-host.h:460
type ConnectCredentialType int32

// ConnectCredentialType enumeration from libvirt/libvirt-host.h:460
const (
	CredUsername     ConnectCredentialType = 1
	CredAuthname     ConnectCredentialType = 2
	CredLanguage     ConnectCredentialType = 3
	CredCnonce       ConnectCredentialType = 4
	CredPassphrase   ConnectCredentialType = 5
	CredEchoprompt   ConnectCredentialType = 6
	CredNoechoprompt ConnectCredentialType = 7
	CredRealm        ConnectCredentialType = 8
	CredExternal     ConnectCredentialType = 9
)

// CPUCompareResult as declared in libvirt/libvirt-host.h:633
type CPUCompareResult int32

// CPUCompareResult enumeration from libvirt/libvirt-host.h:633
const (
	CPUCompareError        CPUCompareResult = -1
	CPUCompareIncompatible CPUCompareResult = 0
	CPUCompareIdentical    CPUCompareResult = 1
	CPUCompareSuperset     CPUCompareResult = 2
)

// ConnectCompareCPUFlags as declared in libvirt/libvirt-host.h:638
type ConnectCompareCPUFlags int32

// ConnectCompareCPUFlags enumeration from libvirt/libvirt-host.h:638
const (
	ConnectCompareCPUFailIncompatible ConnectCompareCPUFlags = 1
)

// ConnectBaselineCPUFlags as declared in libvirt/libvirt-host.h:657
type ConnectBaselineCPUFlags int32

// ConnectBaselineCPUFlags enumeration from libvirt/libvirt-host.h:657
const (
	ConnectBaselineCPUExpandFeatures ConnectBaselineCPUFlags = 1
	ConnectBaselineCPUMigratable     ConnectBaselineCPUFlags = 2
)

// NodeAllocPagesFlags as declared in libvirt/libvirt-host.h:679
type NodeAllocPagesFlags int32

// NodeAllocPagesFlags enumeration from libvirt/libvirt-host.h:679
const (
	NodeAllocPagesAdd NodeAllocPagesFlags = iota
	NodeAllocPagesSet NodeAllocPagesFlags = 1
)

// DomainState as declared in libvirt/libvirt-domain.h:71
type DomainState int32

// DomainState enumeration from libvirt/libvirt-domain.h:71
const (
	DomainNostate     DomainState = iota
	DomainRunning     DomainState = 1
	DomainBlocked     DomainState = 2
	DomainPaused      DomainState = 3
	DomainShutdown    DomainState = 4
	DomainShutoff     DomainState = 5
	DomainCrashed     DomainState = 6
	DomainPmsuspended DomainState = 7
)

// DomainNostateReason as declared in libvirt/libvirt-domain.h:79
type DomainNostateReason int32

// DomainNostateReason enumeration from libvirt/libvirt-domain.h:79
const (
	DomainNostateUnknown DomainNostateReason = iota
)

// DomainRunningReason as declared in libvirt/libvirt-domain.h:98
type DomainRunningReason int32

// DomainRunningReason enumeration from libvirt/libvirt-domain.h:98
const (
	DomainRunningUnknown           DomainRunningReason = iota
	DomainRunningBooted            DomainRunningReason = 1
	DomainRunningMigrated          DomainRunningReason = 2
	DomainRunningRestored          DomainRunningReason = 3
	DomainRunningFromSnapshot      DomainRunningReason = 4
	DomainRunningUnpaused          DomainRunningReason = 5
	DomainRunningMigrationCanceled DomainRunningReason = 6
	DomainRunningSaveCanceled      DomainRunningReason = 7
	DomainRunningWakeup            DomainRunningReason = 8
	DomainRunningCrashed           DomainRunningReason = 9
	DomainRunningPostcopy          DomainRunningReason = 10
)

// DomainBlockedReason as declared in libvirt/libvirt-domain.h:106
type DomainBlockedReason int32

// DomainBlockedReason enumeration from libvirt/libvirt-domain.h:106
const (
	DomainBlockedUnknown DomainBlockedReason = iota
)

// DomainPausedReason as declared in libvirt/libvirt-domain.h:127
type DomainPausedReason int32

// DomainPausedReason enumeration from libvirt/libvirt-domain.h:127
const (
	DomainPausedUnknown        DomainPausedReason = iota
	DomainPausedUser           DomainPausedReason = 1
	DomainPausedMigration      DomainPausedReason = 2
	DomainPausedSave           DomainPausedReason = 3
	DomainPausedDump           DomainPausedReason = 4
	DomainPausedIoerror        DomainPausedReason = 5
	DomainPausedWatchdog       DomainPausedReason = 6
	DomainPausedFromSnapshot   DomainPausedReason = 7
	DomainPausedShuttingDown   DomainPausedReason = 8
	DomainPausedSnapshot       DomainPausedReason = 9
	DomainPausedCrashed        DomainPausedReason = 10
	DomainPausedStartingUp     DomainPausedReason = 11
	DomainPausedPostcopy       DomainPausedReason = 12
	DomainPausedPostcopyFailed DomainPausedReason = 13
)

// DomainShutdownReason as declared in libvirt/libvirt-domain.h:136
type DomainShutdownReason int32

// DomainShutdownReason enumeration from libvirt/libvirt-domain.h:136
const (
	DomainShutdownUnknown DomainShutdownReason = iota
	DomainShutdownUser    DomainShutdownReason = 1
)

// DomainShutoffReason as declared in libvirt/libvirt-domain.h:151
type DomainShutoffReason int32

// DomainShutoffReason enumeration from libvirt/libvirt-domain.h:151
const (
	DomainShutoffUnknown      DomainShutoffReason = iota
	DomainShutoffShutdown     DomainShutoffReason = 1
	DomainShutoffDestroyed    DomainShutoffReason = 2
	DomainShutoffCrashed      DomainShutoffReason = 3
	DomainShutoffMigrated     DomainShutoffReason = 4
	DomainShutoffSaved        DomainShutoffReason = 5
	DomainShutoffFailed       DomainShutoffReason = 6
	DomainShutoffFromSnapshot DomainShutoffReason = 7
)

// DomainCrashedReason as declared in libvirt/libvirt-domain.h:160
type DomainCrashedReason int32

// DomainCrashedReason enumeration from libvirt/libvirt-domain.h:160
const (
	DomainCrashedUnknown  DomainCrashedReason = iota
	DomainCrashedPanicked DomainCrashedReason = 1
)

// DomainPMSuspendedReason as declared in libvirt/libvirt-domain.h:168
type DomainPMSuspendedReason int32

// DomainPMSuspendedReason enumeration from libvirt/libvirt-domain.h:168
const (
	DomainPmsuspendedUnknown DomainPMSuspendedReason = iota
)

// DomainPMSuspendedDiskReason as declared in libvirt/libvirt-domain.h:176
type DomainPMSuspendedDiskReason int32

// DomainPMSuspendedDiskReason enumeration from libvirt/libvirt-domain.h:176
const (
	DomainPmsuspendedDiskUnknown DomainPMSuspendedDiskReason = iota
)

// DomainControlState as declared in libvirt/libvirt-domain.h:196
type DomainControlState int32

// DomainControlState enumeration from libvirt/libvirt-domain.h:196
const (
	DomainControlOk       DomainControlState = iota
	DomainControlJob      DomainControlState = 1
	DomainControlOccupied DomainControlState = 2
	DomainControlError    DomainControlState = 3
)

// DomainControlErrorReason as declared in libvirt/libvirt-domain.h:216
type DomainControlErrorReason int32

// DomainControlErrorReason enumeration from libvirt/libvirt-domain.h:216
const (
	DomainControlErrorReasonNone     DomainControlErrorReason = iota
	DomainControlErrorReasonUnknown  DomainControlErrorReason = 1
	DomainControlErrorReasonMonitor  DomainControlErrorReason = 2
	DomainControlErrorReasonInternal DomainControlErrorReason = 3
)

// DomainModificationImpact as declared in libvirt/libvirt-domain.h:264
type DomainModificationImpact int32

// DomainModificationImpact enumeration from libvirt/libvirt-domain.h:264
const (
	DomainAffectCurrent DomainModificationImpact = iota
	DomainAffectLive    DomainModificationImpact = 1
	DomainAffectConfig  DomainModificationImpact = 2
)

// DomainCreateFlags as declared in libvirt/libvirt-domain.h:304
type DomainCreateFlags int32

// DomainCreateFlags enumeration from libvirt/libvirt-domain.h:304
const (
	DomainNone             DomainCreateFlags = iota
	DomainStartPaused      DomainCreateFlags = 1
	DomainStartAutodestroy DomainCreateFlags = 2
	DomainStartBypassCache DomainCreateFlags = 4
	DomainStartForceBoot   DomainCreateFlags = 8
	DomainStartValidate    DomainCreateFlags = 16
)

// DomainMemoryStatTags as declared in libvirt/libvirt-domain.h:640
type DomainMemoryStatTags int32

// DomainMemoryStatTags enumeration from libvirt/libvirt-domain.h:640
const (
	DomainMemoryStatSwapIn        DomainMemoryStatTags = iota
	DomainMemoryStatSwapOut       DomainMemoryStatTags = 1
	DomainMemoryStatMajorFault    DomainMemoryStatTags = 2
	DomainMemoryStatMinorFault    DomainMemoryStatTags = 3
	DomainMemoryStatUnused        DomainMemoryStatTags = 4
	DomainMemoryStatAvailable     DomainMemoryStatTags = 5
	DomainMemoryStatActualBalloon DomainMemoryStatTags = 6
	DomainMemoryStatRss           DomainMemoryStatTags = 7
	DomainMemoryStatUsable        DomainMemoryStatTags = 8
	DomainMemoryStatLastUpdate    DomainMemoryStatTags = 9
	DomainMemoryStatNr            DomainMemoryStatTags = 10
)

// DomainCoreDumpFlags as declared in libvirt/libvirt-domain.h:659
type DomainCoreDumpFlags int32

// DomainCoreDumpFlags enumeration from libvirt/libvirt-domain.h:659
const (
	DumpCrash       DomainCoreDumpFlags = 1
	DumpLive        DomainCoreDumpFlags = 2
	DumpBypassCache DomainCoreDumpFlags = 4
	DumpReset       DomainCoreDumpFlags = 8
	DumpMemoryOnly  DomainCoreDumpFlags = 16
)

// DomainCoreDumpFormat as declared in libvirt/libvirt-domain.h:682
type DomainCoreDumpFormat int32

// DomainCoreDumpFormat enumeration from libvirt/libvirt-domain.h:682
const (
	DomainCoreDumpFormatRaw         DomainCoreDumpFormat = iota
	DomainCoreDumpFormatKdumpZlib   DomainCoreDumpFormat = 1
	DomainCoreDumpFormatKdumpLzo    DomainCoreDumpFormat = 2
	DomainCoreDumpFormatKdumpSnappy DomainCoreDumpFormat = 3
)

// DomainMigrateFlags as declared in libvirt/libvirt-domain.h:826
type DomainMigrateFlags int32

// DomainMigrateFlags enumeration from libvirt/libvirt-domain.h:826
const (
	MigrateLive             DomainMigrateFlags = 1
	MigratePeer2peer        DomainMigrateFlags = 2
	MigrateTunnelled        DomainMigrateFlags = 4
	MigratePersistDest      DomainMigrateFlags = 8
	MigrateUndefineSource   DomainMigrateFlags = 16
	MigratePaused           DomainMigrateFlags = 32
	MigrateNonSharedDisk    DomainMigrateFlags = 64
	MigrateNonSharedInc     DomainMigrateFlags = 128
	MigrateChangeProtection DomainMigrateFlags = 256
	MigrateUnsafe           DomainMigrateFlags = 512
	MigrateOffline          DomainMigrateFlags = 1024
	MigrateCompressed       DomainMigrateFlags = 2048
	MigrateAbortOnError     DomainMigrateFlags = 4096
	MigrateAutoConverge     DomainMigrateFlags = 8192
	MigrateRdmaPinAll       DomainMigrateFlags = 16384
	MigratePostcopy         DomainMigrateFlags = 32768
	MigrateTLS              DomainMigrateFlags = 65536
)

// DomainShutdownFlagValues as declared in libvirt/libvirt-domain.h:1117
type DomainShutdownFlagValues int32

// DomainShutdownFlagValues enumeration from libvirt/libvirt-domain.h:1117
const (
	DomainShutdownDefault      DomainShutdownFlagValues = iota
	DomainShutdownAcpiPowerBtn DomainShutdownFlagValues = 1
	DomainShutdownGuestAgent   DomainShutdownFlagValues = 2
	DomainShutdownInitctl      DomainShutdownFlagValues = 4
	DomainShutdownSignal       DomainShutdownFlagValues = 8
	DomainShutdownParavirt     DomainShutdownFlagValues = 16
)

// DomainRebootFlagValues as declared in libvirt/libvirt-domain.h:1130
type DomainRebootFlagValues int32

// DomainRebootFlagValues enumeration from libvirt/libvirt-domain.h:1130
const (
	DomainRebootDefault      DomainRebootFlagValues = iota
	DomainRebootAcpiPowerBtn DomainRebootFlagValues = 1
	DomainRebootGuestAgent   DomainRebootFlagValues = 2
	DomainRebootInitctl      DomainRebootFlagValues = 4
	DomainRebootSignal       DomainRebootFlagValues = 8
	DomainRebootParavirt     DomainRebootFlagValues = 16
)

// DomainDestroyFlagsValues as declared in libvirt/libvirt-domain.h:1148
type DomainDestroyFlagsValues int32

// DomainDestroyFlagsValues enumeration from libvirt/libvirt-domain.h:1148
const (
	DomainDestroyDefault  DomainDestroyFlagsValues = iota
	DomainDestroyGraceful DomainDestroyFlagsValues = 1
)

// DomainSaveRestoreFlags as declared in libvirt/libvirt-domain.h:1180
type DomainSaveRestoreFlags int32

// DomainSaveRestoreFlags enumeration from libvirt/libvirt-domain.h:1180
const (
	DomainSaveBypassCache DomainSaveRestoreFlags = 1
	DomainSaveRunning     DomainSaveRestoreFlags = 2
	DomainSavePaused      DomainSaveRestoreFlags = 4
)

// DomainMemoryModFlags as declared in libvirt/libvirt-domain.h:1429
type DomainMemoryModFlags int32

// DomainMemoryModFlags enumeration from libvirt/libvirt-domain.h:1429
const (
	DomainMemCurrent DomainMemoryModFlags = iota
	DomainMemLive    DomainMemoryModFlags = 1
	DomainMemConfig  DomainMemoryModFlags = 2
	DomainMemMaximum DomainMemoryModFlags = 4
)

// DomainNumatuneMemMode as declared in libvirt/libvirt-domain.h:1447
type DomainNumatuneMemMode int32

// DomainNumatuneMemMode enumeration from libvirt/libvirt-domain.h:1447
const (
	DomainNumatuneMemStrict     DomainNumatuneMemMode = iota
	DomainNumatuneMemPreferred  DomainNumatuneMemMode = 1
	DomainNumatuneMemInterleave DomainNumatuneMemMode = 2
)

// DomainMetadataType as declared in libvirt/libvirt-domain.h:1509
type DomainMetadataType int32

// DomainMetadataType enumeration from libvirt/libvirt-domain.h:1509
const (
	DomainMetadataDescription DomainMetadataType = iota
	DomainMetadataTitle       DomainMetadataType = 1
	DomainMetadataElement     DomainMetadataType = 2
)

// DomainXMLFlags as declared in libvirt/libvirt-domain.h:1539
type DomainXMLFlags int32

// DomainXMLFlags enumeration from libvirt/libvirt-domain.h:1539
const (
	DomainXMLSecure     DomainXMLFlags = 1
	DomainXMLInactive   DomainXMLFlags = 2
	DomainXMLUpdateCPU  DomainXMLFlags = 4
	DomainXMLMigratable DomainXMLFlags = 8
)

// DomainBlockResizeFlags as declared in libvirt/libvirt-domain.h:1644
type DomainBlockResizeFlags int32

// DomainBlockResizeFlags enumeration from libvirt/libvirt-domain.h:1644
const (
	DomainBlockResizeBytes DomainBlockResizeFlags = 1
)

// DomainMemoryFlags as declared in libvirt/libvirt-domain.h:1707
type DomainMemoryFlags int32

// DomainMemoryFlags enumeration from libvirt/libvirt-domain.h:1707
const (
	MemoryVirtual  DomainMemoryFlags = 1
	MemoryPhysical DomainMemoryFlags = 2
)

// DomainDefineFlags as declared in libvirt/libvirt-domain.h:1717
type DomainDefineFlags int32

// DomainDefineFlags enumeration from libvirt/libvirt-domain.h:1717
const (
	DomainDefineValidate DomainDefineFlags = 1
)

// DomainUndefineFlagsValues as declared in libvirt/libvirt-domain.h:1741
type DomainUndefineFlagsValues int32

// DomainUndefineFlagsValues enumeration from libvirt/libvirt-domain.h:1741
const (
	DomainUndefineManagedSave       DomainUndefineFlagsValues = 1
	DomainUndefineSnapshotsMetadata DomainUndefineFlagsValues = 2
	DomainUndefineNvram             DomainUndefineFlagsValues = 4
	DomainUndefineKeepNvram         DomainUndefineFlagsValues = 8
)

// ConnectListAllDomainsFlags as declared in libvirt/libvirt-domain.h:1777
type ConnectListAllDomainsFlags int32

// ConnectListAllDomainsFlags enumeration from libvirt/libvirt-domain.h:1777
const (
	ConnectListDomainsActive        ConnectListAllDomainsFlags = 1
	ConnectListDomainsInactive      ConnectListAllDomainsFlags = 2
	ConnectListDomainsPersistent    ConnectListAllDomainsFlags = 4
	ConnectListDomainsTransient     ConnectListAllDomainsFlags = 8
	ConnectListDomainsRunning       ConnectListAllDomainsFlags = 16
	ConnectListDomainsPaused        ConnectListAllDomainsFlags = 32
	ConnectListDomainsShutoff       ConnectListAllDomainsFlags = 64
	ConnectListDomainsOther         ConnectListAllDomainsFlags = 128
	ConnectListDomainsManagedsave   ConnectListAllDomainsFlags = 256
	ConnectListDomainsNoManagedsave ConnectListAllDomainsFlags = 512
	ConnectListDomainsAutostart     ConnectListAllDomainsFlags = 1024
	ConnectListDomainsNoAutostart   ConnectListAllDomainsFlags = 2048
	ConnectListDomainsHasSnapshot   ConnectListAllDomainsFlags = 4096
	ConnectListDomainsNoSnapshot    ConnectListAllDomainsFlags = 8192
)

// VCPUState as declared in libvirt/libvirt-domain.h:1808
type VCPUState int32

// VCPUState enumeration from libvirt/libvirt-domain.h:1808
const (
	VCPUOffline VCPUState = iota
	VCPURunning VCPUState = 1
	VCPUBlocked VCPUState = 2
)

// DomainVCPUFlags as declared in libvirt/libvirt-domain.h:1830
type DomainVCPUFlags int32

// DomainVCPUFlags enumeration from libvirt/libvirt-domain.h:1830
const (
	DomainVCPUCurrent      DomainVCPUFlags = iota
	DomainVCPULive         DomainVCPUFlags = 1
	DomainVCPUConfig       DomainVCPUFlags = 2
	DomainVCPUMaximum      DomainVCPUFlags = 4
	DomainVCPUGuest        DomainVCPUFlags = 8
	DomainVCPUHotpluggable DomainVCPUFlags = 16
)

// DomainDeviceModifyFlags as declared in libvirt/libvirt-domain.h:2003
type DomainDeviceModifyFlags int32

// DomainDeviceModifyFlags enumeration from libvirt/libvirt-domain.h:2003
const (
	DomainDeviceModifyCurrent DomainDeviceModifyFlags = iota
	DomainDeviceModifyLive    DomainDeviceModifyFlags = 1
	DomainDeviceModifyConfig  DomainDeviceModifyFlags = 2
	DomainDeviceModifyForce   DomainDeviceModifyFlags = 4
)

// DomainStatsTypes as declared in libvirt/libvirt-domain.h:2031
type DomainStatsTypes int32

// DomainStatsTypes enumeration from libvirt/libvirt-domain.h:2031
const (
	DomainStatsState     DomainStatsTypes = 1
	DomainStatsCPUTotal  DomainStatsTypes = 2
	DomainStatsBalloon   DomainStatsTypes = 4
	DomainStatsVCPU      DomainStatsTypes = 8
	DomainStatsInterface DomainStatsTypes = 16
	DomainStatsBlock     DomainStatsTypes = 32
	DomainStatsPerf      DomainStatsTypes = 64
)

// ConnectGetAllDomainStatsFlags as declared in libvirt/libvirt-domain.h:2047
type ConnectGetAllDomainStatsFlags int32

// ConnectGetAllDomainStatsFlags enumeration from libvirt/libvirt-domain.h:2047
const (
	ConnectGetAllDomainsStatsActive       ConnectGetAllDomainStatsFlags = 1
	ConnectGetAllDomainsStatsInactive     ConnectGetAllDomainStatsFlags = 2
	ConnectGetAllDomainsStatsPersistent   ConnectGetAllDomainStatsFlags = 4
	ConnectGetAllDomainsStatsTransient    ConnectGetAllDomainStatsFlags = 8
	ConnectGetAllDomainsStatsRunning      ConnectGetAllDomainStatsFlags = 16
	ConnectGetAllDomainsStatsPaused       ConnectGetAllDomainStatsFlags = 32
	ConnectGetAllDomainsStatsShutoff      ConnectGetAllDomainStatsFlags = 64
	ConnectGetAllDomainsStatsOther        ConnectGetAllDomainStatsFlags = 128
	ConnectGetAllDomainsStatsBacking      ConnectGetAllDomainStatsFlags = 1073741824
	ConnectGetAllDomainsStatsEnforceStats ConnectGetAllDomainStatsFlags = -2147483648
)

// DomainBlockJobType as declared in libvirt/libvirt-domain.h:2331
type DomainBlockJobType int32

// DomainBlockJobType enumeration from libvirt/libvirt-domain.h:2331
const (
	DomainBlockJobTypeUnknown      DomainBlockJobType = iota
	DomainBlockJobTypePull         DomainBlockJobType = 1
	DomainBlockJobTypeCopy         DomainBlockJobType = 2
	DomainBlockJobTypeCommit       DomainBlockJobType = 3
	DomainBlockJobTypeActiveCommit DomainBlockJobType = 4
)

// DomainBlockJobAbortFlags as declared in libvirt/libvirt-domain.h:2343
type DomainBlockJobAbortFlags int32

// DomainBlockJobAbortFlags enumeration from libvirt/libvirt-domain.h:2343
const (
	DomainBlockJobAbortAsync DomainBlockJobAbortFlags = 1
	DomainBlockJobAbortPivot DomainBlockJobAbortFlags = 2
)

// DomainBlockJobInfoFlags as declared in libvirt/libvirt-domain.h:2352
type DomainBlockJobInfoFlags int32

// DomainBlockJobInfoFlags enumeration from libvirt/libvirt-domain.h:2352
const (
	DomainBlockJobInfoBandwidthBytes DomainBlockJobInfoFlags = 1
)

// DomainBlockJobSetSpeedFlags as declared in libvirt/libvirt-domain.h:2381
type DomainBlockJobSetSpeedFlags int32

// DomainBlockJobSetSpeedFlags enumeration from libvirt/libvirt-domain.h:2381
const (
	DomainBlockJobSpeedBandwidthBytes DomainBlockJobSetSpeedFlags = 1
)

// DomainBlockPullFlags as declared in libvirt/libvirt-domain.h:2391
type DomainBlockPullFlags int32

// DomainBlockPullFlags enumeration from libvirt/libvirt-domain.h:2391
const (
	DomainBlockPullBandwidthBytes DomainBlockPullFlags = 64
)

// DomainBlockRebaseFlags as declared in libvirt/libvirt-domain.h:2415
type DomainBlockRebaseFlags int32

// DomainBlockRebaseFlags enumeration from libvirt/libvirt-domain.h:2415
const (
	DomainBlockRebaseShallow        DomainBlockRebaseFlags = 1
	DomainBlockRebaseReuseExt       DomainBlockRebaseFlags = 2
	DomainBlockRebaseCopyRaw        DomainBlockRebaseFlags = 4
	DomainBlockRebaseCopy           DomainBlockRebaseFlags = 8
	DomainBlockRebaseRelative       DomainBlockRebaseFlags = 16
	DomainBlockRebaseCopyDev        DomainBlockRebaseFlags = 32
	DomainBlockRebaseBandwidthBytes DomainBlockRebaseFlags = 64
)

// DomainBlockCopyFlags as declared in libvirt/libvirt-domain.h:2431
type DomainBlockCopyFlags int32

// DomainBlockCopyFlags enumeration from libvirt/libvirt-domain.h:2431
const (
	DomainBlockCopyShallow  DomainBlockCopyFlags = 1
	DomainBlockCopyReuseExt DomainBlockCopyFlags = 2
)

// DomainBlockCommitFlags as declared in libvirt/libvirt-domain.h:2496
type DomainBlockCommitFlags int32

// DomainBlockCommitFlags enumeration from libvirt/libvirt-domain.h:2496
const (
	DomainBlockCommitShallow        DomainBlockCommitFlags = 1
	DomainBlockCommitDelete         DomainBlockCommitFlags = 2
	DomainBlockCommitActive         DomainBlockCommitFlags = 4
	DomainBlockCommitRelative       DomainBlockCommitFlags = 8
	DomainBlockCommitBandwidthBytes DomainBlockCommitFlags = 16
)

// DomainDiskErrorCode as declared in libvirt/libvirt-domain.h:2687
type DomainDiskErrorCode int32

// DomainDiskErrorCode enumeration from libvirt/libvirt-domain.h:2687
const (
	DomainDiskErrorNone    DomainDiskErrorCode = iota
	DomainDiskErrorUnspec  DomainDiskErrorCode = 1
	DomainDiskErrorNoSpace DomainDiskErrorCode = 2
)

// KeycodeSet as declared in libvirt/libvirt-domain.h:2733
type KeycodeSet int32

// KeycodeSet enumeration from libvirt/libvirt-domain.h:2733
const (
	KeycodeSetLinux  KeycodeSet = iota
	KeycodeSetXt     KeycodeSet = 1
	KeycodeSetAtset1 KeycodeSet = 2
	KeycodeSetAtset2 KeycodeSet = 3
	KeycodeSetAtset3 KeycodeSet = 4
	KeycodeSetOsx    KeycodeSet = 5
	KeycodeSetXtKbd  KeycodeSet = 6
	KeycodeSetUsb    KeycodeSet = 7
	KeycodeSetWin32  KeycodeSet = 8
	KeycodeSetRfb    KeycodeSet = 9
)

// DomainProcessSignal as declared in libvirt/libvirt-domain.h:2835
type DomainProcessSignal int32

// DomainProcessSignal enumeration from libvirt/libvirt-domain.h:2835
const (
	DomainProcessSignalNop    DomainProcessSignal = iota
	DomainProcessSignalHup    DomainProcessSignal = 1
	DomainProcessSignalInt    DomainProcessSignal = 2
	DomainProcessSignalQuit   DomainProcessSignal = 3
	DomainProcessSignalIll    DomainProcessSignal = 4
	DomainProcessSignalTrap   DomainProcessSignal = 5
	DomainProcessSignalAbrt   DomainProcessSignal = 6
	DomainProcessSignalBus    DomainProcessSignal = 7
	DomainProcessSignalFpe    DomainProcessSignal = 8
	DomainProcessSignalKill   DomainProcessSignal = 9
	DomainProcessSignalUsr1   DomainProcessSignal = 10
	DomainProcessSignalSegv   DomainProcessSignal = 11
	DomainProcessSignalUsr2   DomainProcessSignal = 12
	DomainProcessSignalPipe   DomainProcessSignal = 13
	DomainProcessSignalAlrm   DomainProcessSignal = 14
	DomainProcessSignalTerm   DomainProcessSignal = 15
	DomainProcessSignalStkflt DomainProcessSignal = 16
	DomainProcessSignalChld   DomainProcessSignal = 17
	DomainProcessSignalCont   DomainProcessSignal = 18
	DomainProcessSignalStop   DomainProcessSignal = 19
	DomainProcessSignalTstp   DomainProcessSignal = 20
	DomainProcessSignalTtin   DomainProcessSignal = 21
	DomainProcessSignalTtou   DomainProcessSignal = 22
	DomainProcessSignalUrg    DomainProcessSignal = 23
	DomainProcessSignalXcpu   DomainProcessSignal = 24
	DomainProcessSignalXfsz   DomainProcessSignal = 25
	DomainProcessSignalVtalrm DomainProcessSignal = 26
	DomainProcessSignalProf   DomainProcessSignal = 27
	DomainProcessSignalWinch  DomainProcessSignal = 28
	DomainProcessSignalPoll   DomainProcessSignal = 29
	DomainProcessSignalPwr    DomainProcessSignal = 30
	DomainProcessSignalSys    DomainProcessSignal = 31
	DomainProcessSignalRt0    DomainProcessSignal = 32
	DomainProcessSignalRt1    DomainProcessSignal = 33
	DomainProcessSignalRt2    DomainProcessSignal = 34
	DomainProcessSignalRt3    DomainProcessSignal = 35
	DomainProcessSignalRt4    DomainProcessSignal = 36
	DomainProcessSignalRt5    DomainProcessSignal = 37
	DomainProcessSignalRt6    DomainProcessSignal = 38
	DomainProcessSignalRt7    DomainProcessSignal = 39
	DomainProcessSignalRt8    DomainProcessSignal = 40
	DomainProcessSignalRt9    DomainProcessSignal = 41
	DomainProcessSignalRt10   DomainProcessSignal = 42
	DomainProcessSignalRt11   DomainProcessSignal = 43
	DomainProcessSignalRt12   DomainProcessSignal = 44
	DomainProcessSignalRt13   DomainProcessSignal = 45
	DomainProcessSignalRt14   DomainProcessSignal = 46
	DomainProcessSignalRt15   DomainProcessSignal = 47
	DomainProcessSignalRt16   DomainProcessSignal = 48
	DomainProcessSignalRt17   DomainProcessSignal = 49
	DomainProcessSignalRt18   DomainProcessSignal = 50
	DomainProcessSignalRt19   DomainProcessSignal = 51
	DomainProcessSignalRt20   DomainProcessSignal = 52
	DomainProcessSignalRt21   DomainProcessSignal = 53
	DomainProcessSignalRt22   DomainProcessSignal = 54
	DomainProcessSignalRt23   DomainProcessSignal = 55
	DomainProcessSignalRt24   DomainProcessSignal = 56
	DomainProcessSignalRt25   DomainProcessSignal = 57
	DomainProcessSignalRt26   DomainProcessSignal = 58
	DomainProcessSignalRt27   DomainProcessSignal = 59
	DomainProcessSignalRt28   DomainProcessSignal = 60
	DomainProcessSignalRt29   DomainProcessSignal = 61
	DomainProcessSignalRt30   DomainProcessSignal = 62
	DomainProcessSignalRt31   DomainProcessSignal = 63
	DomainProcessSignalRt32   DomainProcessSignal = 64
)

// DomainEventType as declared in libvirt/libvirt-domain.h:2873
type DomainEventType int32

// DomainEventType enumeration from libvirt/libvirt-domain.h:2873
const (
	DomainEventDefined     DomainEventType = iota
	DomainEventUndefined   DomainEventType = 1
	DomainEventStarted     DomainEventType = 2
	DomainEventSuspended   DomainEventType = 3
	DomainEventResumed     DomainEventType = 4
	DomainEventStopped     DomainEventType = 5
	DomainEventShutdown    DomainEventType = 6
	DomainEventPmsuspended DomainEventType = 7
	DomainEventCrashed     DomainEventType = 8
)

// DomainEventDefinedDetailType as declared in libvirt/libvirt-domain.h:2889
type DomainEventDefinedDetailType int32

// DomainEventDefinedDetailType enumeration from libvirt/libvirt-domain.h:2889
const (
	DomainEventDefinedAdded        DomainEventDefinedDetailType = iota
	DomainEventDefinedUpdated      DomainEventDefinedDetailType = 1
	DomainEventDefinedRenamed      DomainEventDefinedDetailType = 2
	DomainEventDefinedFromSnapshot DomainEventDefinedDetailType = 3
)

// DomainEventUndefinedDetailType as declared in libvirt/libvirt-domain.h:2903
type DomainEventUndefinedDetailType int32

// DomainEventUndefinedDetailType enumeration from libvirt/libvirt-domain.h:2903
const (
	DomainEventUndefinedRemoved DomainEventUndefinedDetailType = iota
	DomainEventUndefinedRenamed DomainEventUndefinedDetailType = 1
)

// DomainEventStartedDetailType as declared in libvirt/libvirt-domain.h:2920
type DomainEventStartedDetailType int32

// DomainEventStartedDetailType enumeration from libvirt/libvirt-domain.h:2920
const (
	DomainEventStartedBooted       DomainEventStartedDetailType = iota
	DomainEventStartedMigrated     DomainEventStartedDetailType = 1
	DomainEventStartedRestored     DomainEventStartedDetailType = 2
	DomainEventStartedFromSnapshot DomainEventStartedDetailType = 3
	DomainEventStartedWakeup       DomainEventStartedDetailType = 4
)

// DomainEventSuspendedDetailType as declared in libvirt/libvirt-domain.h:2941
type DomainEventSuspendedDetailType int32

// DomainEventSuspendedDetailType enumeration from libvirt/libvirt-domain.h:2941
const (
	DomainEventSuspendedPaused         DomainEventSuspendedDetailType = iota
	DomainEventSuspendedMigrated       DomainEventSuspendedDetailType = 1
	DomainEventSuspendedIoerror        DomainEventSuspendedDetailType = 2
	DomainEventSuspendedWatchdog       DomainEventSuspendedDetailType = 3
	DomainEventSuspendedRestored       DomainEventSuspendedDetailType = 4
	DomainEventSuspendedFromSnapshot   DomainEventSuspendedDetailType = 5
	DomainEventSuspendedAPIError       DomainEventSuspendedDetailType = 6
	DomainEventSuspendedPostcopy       DomainEventSuspendedDetailType = 7
	DomainEventSuspendedPostcopyFailed DomainEventSuspendedDetailType = 8
)

// DomainEventResumedDetailType as declared in libvirt/libvirt-domain.h:2958
type DomainEventResumedDetailType int32

// DomainEventResumedDetailType enumeration from libvirt/libvirt-domain.h:2958
const (
	DomainEventResumedUnpaused     DomainEventResumedDetailType = iota
	DomainEventResumedMigrated     DomainEventResumedDetailType = 1
	DomainEventResumedFromSnapshot DomainEventResumedDetailType = 2
	DomainEventResumedPostcopy     DomainEventResumedDetailType = 3
)

// DomainEventStoppedDetailType as declared in libvirt/libvirt-domain.h:2977
type DomainEventStoppedDetailType int32

// DomainEventStoppedDetailType enumeration from libvirt/libvirt-domain.h:2977
const (
	DomainEventStoppedShutdown     DomainEventStoppedDetailType = iota
	DomainEventStoppedDestroyed    DomainEventStoppedDetailType = 1
	DomainEventStoppedCrashed      DomainEventStoppedDetailType = 2
	DomainEventStoppedMigrated     DomainEventStoppedDetailType = 3
	DomainEventStoppedSaved        DomainEventStoppedDetailType = 4
	DomainEventStoppedFailed       DomainEventStoppedDetailType = 5
	DomainEventStoppedFromSnapshot DomainEventStoppedDetailType = 6
)

// DomainEventShutdownDetailType as declared in libvirt/libvirt-domain.h:2991
type DomainEventShutdownDetailType int32

// DomainEventShutdownDetailType enumeration from libvirt/libvirt-domain.h:2991
const (
	DomainEventShutdownFinished DomainEventShutdownDetailType = iota
)

// DomainEventPMSuspendedDetailType as declared in libvirt/libvirt-domain.h:3005
type DomainEventPMSuspendedDetailType int32

// DomainEventPMSuspendedDetailType enumeration from libvirt/libvirt-domain.h:3005
const (
	DomainEventPmsuspendedMemory DomainEventPMSuspendedDetailType = iota
	DomainEventPmsuspendedDisk   DomainEventPMSuspendedDetailType = 1
)

// DomainEventCrashedDetailType as declared in libvirt/libvirt-domain.h:3018
type DomainEventCrashedDetailType int32

// DomainEventCrashedDetailType enumeration from libvirt/libvirt-domain.h:3018
const (
	DomainEventCrashedPanicked DomainEventCrashedDetailType = iota
)

// DomainJobType as declared in libvirt/libvirt-domain.h:3062
type DomainJobType int32

// DomainJobType enumeration from libvirt/libvirt-domain.h:3062
const (
	DomainJobNone      DomainJobType = iota
	DomainJobBounded   DomainJobType = 1
	DomainJobUnbounded DomainJobType = 2
	DomainJobCompleted DomainJobType = 3
	DomainJobFailed    DomainJobType = 4
	DomainJobCancelled DomainJobType = 5
)

// DomainGetJobStatsFlags as declared in libvirt/libvirt-domain.h:3109
type DomainGetJobStatsFlags int32

// DomainGetJobStatsFlags enumeration from libvirt/libvirt-domain.h:3109
const (
	DomainJobStatsCompleted DomainGetJobStatsFlags = 1
)

// DomainJobOperation as declared in libvirt/libvirt-domain.h:3134
type DomainJobOperation int32

// DomainJobOperation enumeration from libvirt/libvirt-domain.h:3134
const (
	DomainJobOperationStrUnknown        DomainJobOperation = iota
	DomainJobOperationStrStart          DomainJobOperation = 1
	DomainJobOperationStrSave           DomainJobOperation = 2
	DomainJobOperationStrRestore        DomainJobOperation = 3
	DomainJobOperationStrMigrationIn    DomainJobOperation = 4
	DomainJobOperationStrMigrationOut   DomainJobOperation = 5
	DomainJobOperationStrSnapshot       DomainJobOperation = 6
	DomainJobOperationStrSnapshotRevert DomainJobOperation = 7
	DomainJobOperationStrDump           DomainJobOperation = 8
)

// DomainEventWatchdogAction as declared in libvirt/libvirt-domain.h:3467
type DomainEventWatchdogAction int32

// DomainEventWatchdogAction enumeration from libvirt/libvirt-domain.h:3467
const (
	DomainEventWatchdogNone      DomainEventWatchdogAction = iota
	DomainEventWatchdogPause     DomainEventWatchdogAction = 1
	DomainEventWatchdogReset     DomainEventWatchdogAction = 2
	DomainEventWatchdogPoweroff  DomainEventWatchdogAction = 3
	DomainEventWatchdogShutdown  DomainEventWatchdogAction = 4
	DomainEventWatchdogDebug     DomainEventWatchdogAction = 5
	DomainEventWatchdogInjectnmi DomainEventWatchdogAction = 6
)

// DomainEventIOErrorAction as declared in libvirt/libvirt-domain.h:3498
type DomainEventIOErrorAction int32

// DomainEventIOErrorAction enumeration from libvirt/libvirt-domain.h:3498
const (
	DomainEventIoErrorNone   DomainEventIOErrorAction = iota
	DomainEventIoErrorPause  DomainEventIOErrorAction = 1
	DomainEventIoErrorReport DomainEventIOErrorAction = 2
)

// DomainEventGraphicsPhase as declared in libvirt/libvirt-domain.h:3561
type DomainEventGraphicsPhase int32

// DomainEventGraphicsPhase enumeration from libvirt/libvirt-domain.h:3561
const (
	DomainEventGraphicsConnect    DomainEventGraphicsPhase = iota
	DomainEventGraphicsInitialize DomainEventGraphicsPhase = 1
	DomainEventGraphicsDisconnect DomainEventGraphicsPhase = 2
)

// DomainEventGraphicsAddressType as declared in libvirt/libvirt-domain.h:3576
type DomainEventGraphicsAddressType int32

// DomainEventGraphicsAddressType enumeration from libvirt/libvirt-domain.h:3576
const (
	DomainEventGraphicsAddressIpv4 DomainEventGraphicsAddressType = iota
	DomainEventGraphicsAddressIpv6 DomainEventGraphicsAddressType = 1
	DomainEventGraphicsAddressUnix DomainEventGraphicsAddressType = 2
)

// ConnectDomainEventBlockJobStatus as declared in libvirt/libvirt-domain.h:3664
type ConnectDomainEventBlockJobStatus int32

// ConnectDomainEventBlockJobStatus enumeration from libvirt/libvirt-domain.h:3664
const (
	DomainBlockJobCompleted ConnectDomainEventBlockJobStatus = iota
	DomainBlockJobFailed    ConnectDomainEventBlockJobStatus = 1
	DomainBlockJobCanceled  ConnectDomainEventBlockJobStatus = 2
	DomainBlockJobReady     ConnectDomainEventBlockJobStatus = 3
)

// ConnectDomainEventDiskChangeReason as declared in libvirt/libvirt-domain.h:3713
type ConnectDomainEventDiskChangeReason int32

// ConnectDomainEventDiskChangeReason enumeration from libvirt/libvirt-domain.h:3713
const (
	DomainEventDiskChangeMissingOnStart ConnectDomainEventDiskChangeReason = iota
	DomainEventDiskDropMissingOnStart   ConnectDomainEventDiskChangeReason = 1
)

// DomainEventTrayChangeReason as declared in libvirt/libvirt-domain.h:3754
type DomainEventTrayChangeReason int32

// DomainEventTrayChangeReason enumeration from libvirt/libvirt-domain.h:3754
const (
	DomainEventTrayChangeOpen  DomainEventTrayChangeReason = iota
	DomainEventTrayChangeClose DomainEventTrayChangeReason = 1
)

// ConnectDomainEventAgentLifecycleState as declared in libvirt/libvirt-domain.h:4269
type ConnectDomainEventAgentLifecycleState int32

// ConnectDomainEventAgentLifecycleState enumeration from libvirt/libvirt-domain.h:4269
const (
	ConnectDomainEventAgentLifecycleStateConnected    ConnectDomainEventAgentLifecycleState = 1
	ConnectDomainEventAgentLifecycleStateDisconnected ConnectDomainEventAgentLifecycleState = 2
)

// ConnectDomainEventAgentLifecycleReason as declared in libvirt/libvirt-domain.h:4279
type ConnectDomainEventAgentLifecycleReason int32

// ConnectDomainEventAgentLifecycleReason enumeration from libvirt/libvirt-domain.h:4279
const (
	ConnectDomainEventAgentLifecycleReasonUnknown       ConnectDomainEventAgentLifecycleReason = iota
	ConnectDomainEventAgentLifecycleReasonDomainStarted ConnectDomainEventAgentLifecycleReason = 1
	ConnectDomainEventAgentLifecycleReasonChannel       ConnectDomainEventAgentLifecycleReason = 2
)

// DomainEventID as declared in libvirt/libvirt-domain.h:4383
type DomainEventID int32

// DomainEventID enumeration from libvirt/libvirt-domain.h:4383
const (
	DomainEventIDLifecycle           DomainEventID = iota
	DomainEventIDReboot              DomainEventID = 1
	DomainEventIDRtcChange           DomainEventID = 2
	DomainEventIDWatchdog            DomainEventID = 3
	DomainEventIDIoError             DomainEventID = 4
	DomainEventIDGraphics            DomainEventID = 5
	DomainEventIDIoErrorReason       DomainEventID = 6
	DomainEventIDControlError        DomainEventID = 7
	DomainEventIDBlockJob            DomainEventID = 8
	DomainEventIDDiskChange          DomainEventID = 9
	DomainEventIDTrayChange          DomainEventID = 10
	DomainEventIDPmwakeup            DomainEventID = 11
	DomainEventIDPmsuspend           DomainEventID = 12
	DomainEventIDBalloonChange       DomainEventID = 13
	DomainEventIDPmsuspendDisk       DomainEventID = 14
	DomainEventIDDeviceRemoved       DomainEventID = 15
	DomainEventIDBlockJob2           DomainEventID = 16
	DomainEventIDTunable             DomainEventID = 17
	DomainEventIDAgentLifecycle      DomainEventID = 18
	DomainEventIDDeviceAdded         DomainEventID = 19
	DomainEventIDMigrationIteration  DomainEventID = 20
	DomainEventIDJobCompleted        DomainEventID = 21
	DomainEventIDDeviceRemovalFailed DomainEventID = 22
	DomainEventIDMetadataChange      DomainEventID = 23
	DomainEventIDBlockThreshold      DomainEventID = 24
)

// DomainConsoleFlags as declared in libvirt/libvirt-domain.h:4410
type DomainConsoleFlags int32

// DomainConsoleFlags enumeration from libvirt/libvirt-domain.h:4410
const (
	DomainConsoleForce DomainConsoleFlags = 1
	DomainConsoleSafe  DomainConsoleFlags = 2
)

// DomainChannelFlags as declared in libvirt/libvirt-domain.h:4426
type DomainChannelFlags int32

// DomainChannelFlags enumeration from libvirt/libvirt-domain.h:4426
const (
	DomainChannelForce DomainChannelFlags = 1
)

// DomainOpenGraphicsFlags as declared in libvirt/libvirt-domain.h:4435
type DomainOpenGraphicsFlags int32

// DomainOpenGraphicsFlags enumeration from libvirt/libvirt-domain.h:4435
const (
	DomainOpenGraphicsSkipauth DomainOpenGraphicsFlags = 1
)

// DomainSetTimeFlags as declared in libvirt/libvirt-domain.h:4492
type DomainSetTimeFlags int32

// DomainSetTimeFlags enumeration from libvirt/libvirt-domain.h:4492
const (
	DomainTimeSync DomainSetTimeFlags = 1
)

// SchedParameterType as declared in libvirt/libvirt-domain.h:4513
type SchedParameterType int32

// SchedParameterType enumeration from libvirt/libvirt-domain.h:4513
const (
	DomainSchedFieldInt     SchedParameterType = 1
	DomainSchedFieldUint    SchedParameterType = 2
	DomainSchedFieldLlong   SchedParameterType = 3
	DomainSchedFieldUllong  SchedParameterType = 4
	DomainSchedFieldDouble  SchedParameterType = 5
	DomainSchedFieldBoolean SchedParameterType = 6
)

// BlkioParameterType as declared in libvirt/libvirt-domain.h:4557
type BlkioParameterType int32

// BlkioParameterType enumeration from libvirt/libvirt-domain.h:4557
const (
	DomainBlkioParamInt     BlkioParameterType = 1
	DomainBlkioParamUint    BlkioParameterType = 2
	DomainBlkioParamLlong   BlkioParameterType = 3
	DomainBlkioParamUllong  BlkioParameterType = 4
	DomainBlkioParamDouble  BlkioParameterType = 5
	DomainBlkioParamBoolean BlkioParameterType = 6
)

// MemoryParameterType as declared in libvirt/libvirt-domain.h:4601
type MemoryParameterType int32

// MemoryParameterType enumeration from libvirt/libvirt-domain.h:4601
const (
	DomainMemoryParamInt     MemoryParameterType = 1
	DomainMemoryParamUint    MemoryParameterType = 2
	DomainMemoryParamLlong   MemoryParameterType = 3
	DomainMemoryParamUllong  MemoryParameterType = 4
	DomainMemoryParamDouble  MemoryParameterType = 5
	DomainMemoryParamBoolean MemoryParameterType = 6
)

// DomainInterfaceAddressesSource as declared in libvirt/libvirt-domain.h:4638
type DomainInterfaceAddressesSource int32

// DomainInterfaceAddressesSource enumeration from libvirt/libvirt-domain.h:4638
const (
	DomainInterfaceAddressesSrcLease DomainInterfaceAddressesSource = iota
	DomainInterfaceAddressesSrcAgent DomainInterfaceAddressesSource = 1
)

// DomainSetUserPasswordFlags as declared in libvirt/libvirt-domain.h:4666
type DomainSetUserPasswordFlags int32

// DomainSetUserPasswordFlags enumeration from libvirt/libvirt-domain.h:4666
const (
	DomainPasswordEncrypted DomainSetUserPasswordFlags = 1
)

// DomainSnapshotCreateFlags as declared in libvirt/libvirt-domain-snapshot.h:73
type DomainSnapshotCreateFlags int32

// DomainSnapshotCreateFlags enumeration from libvirt/libvirt-domain-snapshot.h:73
const (
	DomainSnapshotCreateRedefine   DomainSnapshotCreateFlags = 1
	DomainSnapshotCreateCurrent    DomainSnapshotCreateFlags = 2
	DomainSnapshotCreateNoMetadata DomainSnapshotCreateFlags = 4
	DomainSnapshotCreateHalt       DomainSnapshotCreateFlags = 8
	DomainSnapshotCreateDiskOnly   DomainSnapshotCreateFlags = 16
	DomainSnapshotCreateReuseExt   DomainSnapshotCreateFlags = 32
	DomainSnapshotCreateQuiesce    DomainSnapshotCreateFlags = 64
	DomainSnapshotCreateAtomic     DomainSnapshotCreateFlags = 128
	DomainSnapshotCreateLive       DomainSnapshotCreateFlags = 256
)

// DomainSnapshotListFlags as declared in libvirt/libvirt-domain-snapshot.h:133
type DomainSnapshotListFlags int32

// DomainSnapshotListFlags enumeration from libvirt/libvirt-domain-snapshot.h:133
const (
	DomainSnapshotListRoots       DomainSnapshotListFlags = 1
	DomainSnapshotListDescendants DomainSnapshotListFlags = 1
	DomainSnapshotListLeaves      DomainSnapshotListFlags = 4
	DomainSnapshotListNoLeaves    DomainSnapshotListFlags = 8
	DomainSnapshotListMetadata    DomainSnapshotListFlags = 2
	DomainSnapshotListNoMetadata  DomainSnapshotListFlags = 16
	DomainSnapshotListInactive    DomainSnapshotListFlags = 32
	DomainSnapshotListActive      DomainSnapshotListFlags = 64
	DomainSnapshotListDiskOnly    DomainSnapshotListFlags = 128
	DomainSnapshotListInternal    DomainSnapshotListFlags = 256
	DomainSnapshotListExternal    DomainSnapshotListFlags = 512
)

// DomainSnapshotRevertFlags as declared in libvirt/libvirt-domain-snapshot.h:190
type DomainSnapshotRevertFlags int32

// DomainSnapshotRevertFlags enumeration from libvirt/libvirt-domain-snapshot.h:190
const (
	DomainSnapshotRevertRunning DomainSnapshotRevertFlags = 1
	DomainSnapshotRevertPaused  DomainSnapshotRevertFlags = 2
	DomainSnapshotRevertForce   DomainSnapshotRevertFlags = 4
)

// DomainSnapshotDeleteFlags as declared in libvirt/libvirt-domain-snapshot.h:204
type DomainSnapshotDeleteFlags int32

// DomainSnapshotDeleteFlags enumeration from libvirt/libvirt-domain-snapshot.h:204
const (
	DomainSnapshotDeleteChildren     DomainSnapshotDeleteFlags = 1
	DomainSnapshotDeleteMetadataOnly DomainSnapshotDeleteFlags = 2
	DomainSnapshotDeleteChildrenOnly DomainSnapshotDeleteFlags = 4
)

// EventHandleType as declared in libvirt/libvirt-event.h:44
type EventHandleType int32

// EventHandleType enumeration from libvirt/libvirt-event.h:44
const (
	EventHandleReadable EventHandleType = 1
	EventHandleWritable EventHandleType = 2
	EventHandleError    EventHandleType = 4
	EventHandleHangup   EventHandleType = 8
)

// ConnectListAllInterfacesFlags as declared in libvirt/libvirt-interface.h:65
type ConnectListAllInterfacesFlags int32

// ConnectListAllInterfacesFlags enumeration from libvirt/libvirt-interface.h:65
const (
	ConnectListInterfacesInactive ConnectListAllInterfacesFlags = 1
	ConnectListInterfacesActive   ConnectListAllInterfacesFlags = 2
)

// InterfaceXMLFlags as declared in libvirt/libvirt-interface.h:81
type InterfaceXMLFlags int32

// InterfaceXMLFlags enumeration from libvirt/libvirt-interface.h:81
const (
	InterfaceXMLInactive InterfaceXMLFlags = 1
)

// NetworkXMLFlags as declared in libvirt/libvirt-network.h:33
type NetworkXMLFlags int32

// NetworkXMLFlags enumeration from libvirt/libvirt-network.h:33
const (
	NetworkXMLInactive NetworkXMLFlags = 1
)

// ConnectListAllNetworksFlags as declared in libvirt/libvirt-network.h:85
type ConnectListAllNetworksFlags int32

// ConnectListAllNetworksFlags enumeration from libvirt/libvirt-network.h:85
const (
	ConnectListNetworksInactive    ConnectListAllNetworksFlags = 1
	ConnectListNetworksActive      ConnectListAllNetworksFlags = 2
	ConnectListNetworksPersistent  ConnectListAllNetworksFlags = 4
	ConnectListNetworksTransient   ConnectListAllNetworksFlags = 8
	ConnectListNetworksAutostart   ConnectListAllNetworksFlags = 16
	ConnectListNetworksNoAutostart ConnectListAllNetworksFlags = 32
)

// NetworkUpdateCommand as declared in libvirt/libvirt-network.h:134
type NetworkUpdateCommand int32

// NetworkUpdateCommand enumeration from libvirt/libvirt-network.h:134
const (
	NetworkUpdateCommandNone     NetworkUpdateCommand = iota
	NetworkUpdateCommandModify   NetworkUpdateCommand = 1
	NetworkUpdateCommandDelete   NetworkUpdateCommand = 2
	NetworkUpdateCommandAddLast  NetworkUpdateCommand = 3
	NetworkUpdateCommandAddFirst NetworkUpdateCommand = 4
)

// NetworkUpdateSection as declared in libvirt/libvirt-network.h:160
type NetworkUpdateSection int32

// NetworkUpdateSection enumeration from libvirt/libvirt-network.h:160
const (
	NetworkSectionNone             NetworkUpdateSection = iota
	NetworkSectionBridge           NetworkUpdateSection = 1
	NetworkSectionDomain           NetworkUpdateSection = 2
	NetworkSectionIP               NetworkUpdateSection = 3
	NetworkSectionIPDhcpHost       NetworkUpdateSection = 4
	NetworkSectionIPDhcpRange      NetworkUpdateSection = 5
	NetworkSectionForward          NetworkUpdateSection = 6
	NetworkSectionForwardInterface NetworkUpdateSection = 7
	NetworkSectionForwardPf        NetworkUpdateSection = 8
	NetworkSectionPortgroup        NetworkUpdateSection = 9
	NetworkSectionDNSHost          NetworkUpdateSection = 10
	NetworkSectionDNSTxt           NetworkUpdateSection = 11
	NetworkSectionDNSSrv           NetworkUpdateSection = 12
)

// NetworkUpdateFlags as declared in libvirt/libvirt-network.h:172
type NetworkUpdateFlags int32

// NetworkUpdateFlags enumeration from libvirt/libvirt-network.h:172
const (
	NetworkUpdateAffectCurrent NetworkUpdateFlags = iota
	NetworkUpdateAffectLive    NetworkUpdateFlags = 1
	NetworkUpdateAffectConfig  NetworkUpdateFlags = 2
)

// NetworkEventLifecycleType as declared in libvirt/libvirt-network.h:230
type NetworkEventLifecycleType int32

// NetworkEventLifecycleType enumeration from libvirt/libvirt-network.h:230
const (
	NetworkEventDefined   NetworkEventLifecycleType = iota
	NetworkEventUndefined NetworkEventLifecycleType = 1
	NetworkEventStarted   NetworkEventLifecycleType = 2
	NetworkEventStopped   NetworkEventLifecycleType = 3
)

// NetworkEventID as declared in libvirt/libvirt-network.h:278
type NetworkEventID int32

// NetworkEventID enumeration from libvirt/libvirt-network.h:278
const (
	NetworkEventIDLifecycle NetworkEventID = iota
)

// IPAddrType as declared in libvirt/libvirt-network.h:287
type IPAddrType int32

// IPAddrType enumeration from libvirt/libvirt-network.h:287
const (
	IPAddrTypeIpv4 IPAddrType = iota
	IPAddrTypeIpv6 IPAddrType = 1
)

// ConnectListAllNodeDeviceFlags as declared in libvirt/libvirt-nodedev.h:82
type ConnectListAllNodeDeviceFlags int32

// ConnectListAllNodeDeviceFlags enumeration from libvirt/libvirt-nodedev.h:82
const (
	ConnectListNodeDevicesCapSystem       ConnectListAllNodeDeviceFlags = 1
	ConnectListNodeDevicesCapPciDev       ConnectListAllNodeDeviceFlags = 2
	ConnectListNodeDevicesCapUsbDev       ConnectListAllNodeDeviceFlags = 4
	ConnectListNodeDevicesCapUsbInterface ConnectListAllNodeDeviceFlags = 8
	ConnectListNodeDevicesCapNet          ConnectListAllNodeDeviceFlags = 16
	ConnectListNodeDevicesCapScsiHost     ConnectListAllNodeDeviceFlags = 32
	ConnectListNodeDevicesCapScsiTarget   ConnectListAllNodeDeviceFlags = 64
	ConnectListNodeDevicesCapScsi         ConnectListAllNodeDeviceFlags = 128
	ConnectListNodeDevicesCapStorage      ConnectListAllNodeDeviceFlags = 256
	ConnectListNodeDevicesCapFcHost       ConnectListAllNodeDeviceFlags = 512
	ConnectListNodeDevicesCapVports       ConnectListAllNodeDeviceFlags = 1024
	ConnectListNodeDevicesCapScsiGeneric  ConnectListAllNodeDeviceFlags = 2048
	ConnectListNodeDevicesCapDrm          ConnectListAllNodeDeviceFlags = 4096
)

// NodeDeviceEventID as declared in libvirt/libvirt-nodedev.h:152
type NodeDeviceEventID int32

// NodeDeviceEventID enumeration from libvirt/libvirt-nodedev.h:152
const (
	NodeDeviceEventIDLifecycle NodeDeviceEventID = iota
	NodeDeviceEventIDUpdate    NodeDeviceEventID = 1
)

// NodeDeviceEventLifecycleType as declared in libvirt/libvirt-nodedev.h:194
type NodeDeviceEventLifecycleType int32

// NodeDeviceEventLifecycleType enumeration from libvirt/libvirt-nodedev.h:194
const (
	NodeDeviceEventCreated NodeDeviceEventLifecycleType = iota
	NodeDeviceEventDeleted NodeDeviceEventLifecycleType = 1
)

// SecretUsageType as declared in libvirt/libvirt-secret.h:56
type SecretUsageType int32

// SecretUsageType enumeration from libvirt/libvirt-secret.h:56
const (
	SecretUsageTypeNone   SecretUsageType = iota
	SecretUsageTypeVolume SecretUsageType = 1
	SecretUsageTypeCeph   SecretUsageType = 2
	SecretUsageTypeIscsi  SecretUsageType = 3
	SecretUsageTypeTLS    SecretUsageType = 4
)

// ConnectListAllSecretsFlags as declared in libvirt/libvirt-secret.h:79
type ConnectListAllSecretsFlags int32

// ConnectListAllSecretsFlags enumeration from libvirt/libvirt-secret.h:79
const (
	ConnectListSecretsEphemeral   ConnectListAllSecretsFlags = 1
	ConnectListSecretsNoEphemeral ConnectListAllSecretsFlags = 2
	ConnectListSecretsPrivate     ConnectListAllSecretsFlags = 4
	ConnectListSecretsNoPrivate   ConnectListAllSecretsFlags = 8
)

// SecretEventID as declared in libvirt/libvirt-secret.h:140
type SecretEventID int32

// SecretEventID enumeration from libvirt/libvirt-secret.h:140
const (
	SecretEventIDLifecycle    SecretEventID = iota
	SecretEventIDValueChanged SecretEventID = 1
)

// SecretEventLifecycleType as declared in libvirt/libvirt-secret.h:182
type SecretEventLifecycleType int32

// SecretEventLifecycleType enumeration from libvirt/libvirt-secret.h:182
const (
	SecretEventDefined   SecretEventLifecycleType = iota
	SecretEventUndefined SecretEventLifecycleType = 1
)

// StoragePoolState as declared in libvirt/libvirt-storage.h:58
type StoragePoolState int32

// StoragePoolState enumeration from libvirt/libvirt-storage.h:58
const (
	StoragePoolInactive     StoragePoolState = iota
	StoragePoolBuilding     StoragePoolState = 1
	StoragePoolRunning      StoragePoolState = 2
	StoragePoolDegraded     StoragePoolState = 3
	StoragePoolInaccessible StoragePoolState = 4
)

// StoragePoolBuildFlags as declared in libvirt/libvirt-storage.h:66
type StoragePoolBuildFlags int32

// StoragePoolBuildFlags enumeration from libvirt/libvirt-storage.h:66
const (
	StoragePoolBuildNew         StoragePoolBuildFlags = iota
	StoragePoolBuildRepair      StoragePoolBuildFlags = 1
	StoragePoolBuildResize      StoragePoolBuildFlags = 2
	StoragePoolBuildNoOverwrite StoragePoolBuildFlags = 4
	StoragePoolBuildOverwrite   StoragePoolBuildFlags = 8
)

// StoragePoolDeleteFlags as declared in libvirt/libvirt-storage.h:71
type StoragePoolDeleteFlags int32

// StoragePoolDeleteFlags enumeration from libvirt/libvirt-storage.h:71
const (
	StoragePoolDeleteNormal StoragePoolDeleteFlags = iota
	StoragePoolDeleteZeroed StoragePoolDeleteFlags = 1
)

// StoragePoolCreateFlags as declared in libvirt/libvirt-storage.h:88
type StoragePoolCreateFlags int32

// StoragePoolCreateFlags enumeration from libvirt/libvirt-storage.h:88
const (
	StoragePoolCreateNormal               StoragePoolCreateFlags = iota
	StoragePoolCreateWithBuild            StoragePoolCreateFlags = 1
	StoragePoolCreateWithBuildOverwrite   StoragePoolCreateFlags = 2
	StoragePoolCreateWithBuildNoOverwrite StoragePoolCreateFlags = 4
)

// StorageVolType as declared in libvirt/libvirt-storage.h:130
type StorageVolType int32

// StorageVolType enumeration from libvirt/libvirt-storage.h:130
const (
	StorageVolFile    StorageVolType = iota
	StorageVolBlock   StorageVolType = 1
	StorageVolDir     StorageVolType = 2
	StorageVolNetwork StorageVolType = 3
	StorageVolNetdir  StorageVolType = 4
	StorageVolPloop   StorageVolType = 5
)

// StorageVolDeleteFlags as declared in libvirt/libvirt-storage.h:136
type StorageVolDeleteFlags int32

// StorageVolDeleteFlags enumeration from libvirt/libvirt-storage.h:136
const (
	StorageVolDeleteNormal        StorageVolDeleteFlags = iota
	StorageVolDeleteZeroed        StorageVolDeleteFlags = 1
	StorageVolDeleteWithSnapshots StorageVolDeleteFlags = 2
)

// StorageVolWipeAlgorithm as declared in libvirt/libvirt-storage.h:168
type StorageVolWipeAlgorithm int32

// StorageVolWipeAlgorithm enumeration from libvirt/libvirt-storage.h:168
const (
	StorageVolWipeAlgZero       StorageVolWipeAlgorithm = iota
	StorageVolWipeAlgNnsa       StorageVolWipeAlgorithm = 1
	StorageVolWipeAlgDod        StorageVolWipeAlgorithm = 2
	StorageVolWipeAlgBsi        StorageVolWipeAlgorithm = 3
	StorageVolWipeAlgGutmann    StorageVolWipeAlgorithm = 4
	StorageVolWipeAlgSchneier   StorageVolWipeAlgorithm = 5
	StorageVolWipeAlgPfitzner7  StorageVolWipeAlgorithm = 6
	StorageVolWipeAlgPfitzner33 StorageVolWipeAlgorithm = 7
	StorageVolWipeAlgRandom     StorageVolWipeAlgorithm = 8
	StorageVolWipeAlgTrim       StorageVolWipeAlgorithm = 9
)

// StorageVolInfoFlags as declared in libvirt/libvirt-storage.h:176
type StorageVolInfoFlags int32

// StorageVolInfoFlags enumeration from libvirt/libvirt-storage.h:176
const (
	StorageVolUseAllocation StorageVolInfoFlags = iota
	StorageVolGetPhysical   StorageVolInfoFlags = 1
)

// StorageXMLFlags as declared in libvirt/libvirt-storage.h:190
type StorageXMLFlags int32

// StorageXMLFlags enumeration from libvirt/libvirt-storage.h:190
const (
	StorageXMLInactive StorageXMLFlags = 1
)

// ConnectListAllStoragePoolsFlags as declared in libvirt/libvirt-storage.h:244
type ConnectListAllStoragePoolsFlags int32

// ConnectListAllStoragePoolsFlags enumeration from libvirt/libvirt-storage.h:244
const (
	ConnectListStoragePoolsInactive    ConnectListAllStoragePoolsFlags = 1
	ConnectListStoragePoolsActive      ConnectListAllStoragePoolsFlags = 2
	ConnectListStoragePoolsPersistent  ConnectListAllStoragePoolsFlags = 4
	ConnectListStoragePoolsTransient   ConnectListAllStoragePoolsFlags = 8
	ConnectListStoragePoolsAutostart   ConnectListAllStoragePoolsFlags = 16
	ConnectListStoragePoolsNoAutostart ConnectListAllStoragePoolsFlags = 32
	ConnectListStoragePoolsDir         ConnectListAllStoragePoolsFlags = 64
	ConnectListStoragePoolsFs          ConnectListAllStoragePoolsFlags = 128
	ConnectListStoragePoolsNetfs       ConnectListAllStoragePoolsFlags = 256
	ConnectListStoragePoolsLogical     ConnectListAllStoragePoolsFlags = 512
	ConnectListStoragePoolsDisk        ConnectListAllStoragePoolsFlags = 1024
	ConnectListStoragePoolsIscsi       ConnectListAllStoragePoolsFlags = 2048
	ConnectListStoragePoolsScsi        ConnectListAllStoragePoolsFlags = 4096
	ConnectListStoragePoolsMpath       ConnectListAllStoragePoolsFlags = 8192
	ConnectListStoragePoolsRbd         ConnectListAllStoragePoolsFlags = 16384
	ConnectListStoragePoolsSheepdog    ConnectListAllStoragePoolsFlags = 32768
	ConnectListStoragePoolsGluster     ConnectListAllStoragePoolsFlags = 65536
	ConnectListStoragePoolsZfs         ConnectListAllStoragePoolsFlags = 131072
	ConnectListStoragePoolsVstorage    ConnectListAllStoragePoolsFlags = 262144
)

// StorageVolCreateFlags as declared in libvirt/libvirt-storage.h:340
type StorageVolCreateFlags int32

// StorageVolCreateFlags enumeration from libvirt/libvirt-storage.h:340
const (
	StorageVolCreatePreallocMetadata StorageVolCreateFlags = 1
	StorageVolCreateReflink          StorageVolCreateFlags = 2
)

// StorageVolResizeFlags as declared in libvirt/libvirt-storage.h:383
type StorageVolResizeFlags int32

// StorageVolResizeFlags enumeration from libvirt/libvirt-storage.h:383
const (
	StorageVolResizeAllocate StorageVolResizeFlags = 1
	StorageVolResizeDelta    StorageVolResizeFlags = 2
	StorageVolResizeShrink   StorageVolResizeFlags = 4
)

// StoragePoolEventID as declared in libvirt/libvirt-storage.h:419
type StoragePoolEventID int32

// StoragePoolEventID enumeration from libvirt/libvirt-storage.h:419
const (
	StoragePoolEventIDLifecycle StoragePoolEventID = iota
	StoragePoolEventIDRefresh   StoragePoolEventID = 1
)

// StoragePoolEventLifecycleType as declared in libvirt/libvirt-storage.h:463
type StoragePoolEventLifecycleType int32

// StoragePoolEventLifecycleType enumeration from libvirt/libvirt-storage.h:463
const (
	StoragePoolEventDefined   StoragePoolEventLifecycleType = iota
	StoragePoolEventUndefined StoragePoolEventLifecycleType = 1
	StoragePoolEventStarted   StoragePoolEventLifecycleType = 2
	StoragePoolEventStopped   StoragePoolEventLifecycleType = 3
)

// StreamFlags as declared in libvirt/libvirt-stream.h:34
type StreamFlags int32

// StreamFlags enumeration from libvirt/libvirt-stream.h:34
const (
	StreamNonblock StreamFlags = 1
)

// StreamEventType as declared in libvirt/libvirt-stream.h:120
type StreamEventType int32

// StreamEventType enumeration from libvirt/libvirt-stream.h:120
const (
	StreamEventReadable StreamEventType = 1
	StreamEventWritable StreamEventType = 2
	StreamEventError    StreamEventType = 4
	StreamEventHangup   StreamEventType = 8
)
