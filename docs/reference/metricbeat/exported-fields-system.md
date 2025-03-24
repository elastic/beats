---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-system.html
---

# System fields [exported-fields-system]

System status metrics, like CPU and memory usage, that are collected from the operating system.


## process [_process_8]

Process metrics.

**`process.state`**
:   The process state. For example: "running".

type: keyword


**`process.cpu.pct`**
:   The percentage of CPU time spent by the process since the last event. This value is normalized by the number of CPU cores and it ranges from 0 to 1.

type: scaled_float

format: percent


**`process.cpu.start_time`**
:   The time when the process was started.

type: date


**`process.memory.pct`**
:   The percentage of memory the process occupied in main memory (RAM).

type: scaled_float

format: percent



## system [_system_4]

`system` contains local system metrics.


## core [_core_2]

`system-core` contains CPU metrics for a single core of a multi-core system.

**`system.core.id`**
:   CPU Core number.

type: long


**`system.core.total.pct`**
:   Total active time spent by the core

type: scaled_float

format: percent


**`system.core.user.pct`**
:   The percentage of CPU time spent in user space.

type: scaled_float

format: percent


**`system.core.user.ticks`**
:   The amount of CPU time spent in user space.

type: long


**`system.core.system.pct`**
:   The percentage of CPU time spent in kernel space.

type: scaled_float

format: percent


**`system.core.system.ticks`**
:   The amount of CPU time spent in kernel space.

type: long


**`system.core.nice.pct`**
:   The percentage of CPU time spent on low-priority processes.

type: scaled_float

format: percent


**`system.core.nice.ticks`**
:   The amount of CPU time spent on low-priority processes.

type: long


**`system.core.idle.pct`**
:   The percentage of CPU time spent idle.

type: scaled_float

format: percent


**`system.core.idle.ticks`**
:   The amount of CPU time spent idle.

type: long


**`system.core.iowait.pct`**
:   The percentage of CPU time spent in wait (on disk).

type: scaled_float

format: percent


**`system.core.iowait.ticks`**
:   The amount of CPU time spent in wait (on disk).

type: long


**`system.core.irq.pct`**
:   The percentage of CPU time spent servicing and handling hardware interrupts.

type: scaled_float

format: percent


**`system.core.irq.ticks`**
:   The amount of CPU time spent servicing and handling hardware interrupts.

type: long


**`system.core.softirq.pct`**
:   The percentage of CPU time spent servicing and handling software interrupts.

type: scaled_float

format: percent


**`system.core.softirq.ticks`**
:   The amount of CPU time spent servicing and handling software interrupts.

type: long


**`system.core.steal.pct`**
:   The percentage of CPU time spent in involuntary wait by the virtual CPU while the hypervisor was servicing another processor. Available only on Unix.

type: scaled_float

format: percent


**`system.core.steal.ticks`**
:   The amount of CPU time spent in involuntary wait by the virtual CPU while the hypervisor was servicing another processor. Available only on Unix.

type: long


**`system.core.model_number`**
:   CPU model number. Only availabe on Linux

type: keyword


**`system.core.model_name`**
:   CPU model name. Only availabe on Linux

type: keyword


**`system.core.mhz`**
:   CPU core current clock. Only availabe on Linux

type: float


**`system.core.core_id`**
:   CPU physical core ID. One core might might execute multiple threads, hence more than one `system.core.id` can share the same `system.core.core_id`. Only availabe on Linux

type: keyword


**`system.core.physical_id`**
:   CPU core physical ID. Only availabe on Linux

type: keyword



## cpu [_cpu_11]

`cpu` contains local CPU stats.

**`system.cpu.cores`**
:   The number of CPU cores present on the host. The non-normalized percentages will have a maximum value of `100% * cores`. The normalized percentages already take this value into account and have a maximum value of 100%.

type: long


**`system.cpu.user.pct`**
:   The percentage of CPU time spent in user space. On multi-core systems, you can have percentages that are greater than 100%. For example, if 3 cores are at 60% use, then the `system.cpu.user.pct` will be 180%.

type: scaled_float

format: percent


**`system.cpu.system.pct`**
:   The percentage of CPU time spent in kernel space.

type: scaled_float

format: percent


**`system.cpu.nice.pct`**
:   The percentage of CPU time spent on low-priority processes.

type: scaled_float

format: percent


**`system.cpu.idle.pct`**
:   The percentage of CPU time spent idle.

type: scaled_float

format: percent


**`system.cpu.iowait.pct`**
:   The percentage of CPU time spent in wait (on disk).

type: scaled_float

format: percent


**`system.cpu.irq.pct`**
:   The percentage of CPU time spent servicing and handling hardware interrupts.

type: scaled_float

format: percent


**`system.cpu.softirq.pct`**
:   The percentage of CPU time spent servicing and handling software interrupts.

type: scaled_float

format: percent


**`system.cpu.steal.pct`**
:   The percentage of CPU time spent in involuntary wait by the virtual CPU while the hypervisor was servicing another processor. Available only on Unix.

type: scaled_float

format: percent


**`system.cpu.total.pct`**
:   The percentage of CPU time spent in states other than Idle and IOWait.

type: scaled_float

format: percent


**`system.cpu.user.norm.pct`**
:   The percentage of CPU time spent in user space.

type: scaled_float

format: percent


**`system.cpu.system.norm.pct`**
:   The percentage of CPU time spent in kernel space.

type: scaled_float

format: percent


**`system.cpu.nice.norm.pct`**
:   The percentage of CPU time spent on low-priority processes.

type: scaled_float

format: percent


**`system.cpu.idle.norm.pct`**
:   The percentage of CPU time spent idle.

type: scaled_float

format: percent


**`system.cpu.iowait.norm.pct`**
:   The percentage of CPU time spent in wait (on disk).

type: scaled_float

format: percent


**`system.cpu.irq.norm.pct`**
:   The percentage of CPU time spent servicing and handling hardware interrupts.

type: scaled_float

format: percent


**`system.cpu.softirq.norm.pct`**
:   The percentage of CPU time spent servicing and handling software interrupts.

type: scaled_float

format: percent


**`system.cpu.steal.norm.pct`**
:   The percentage of CPU time spent in involuntary wait by the virtual CPU while the hypervisor was servicing another processor. Available only on Unix.

type: scaled_float

format: percent


**`system.cpu.total.norm.pct`**
:   The percentage of CPU time in states other than Idle and IOWait, normalised by the number of cores.

type: scaled_float

format: percent


**`system.cpu.user.ticks`**
:   The amount of CPU time spent in user space.

type: long


**`system.cpu.system.ticks`**
:   The amount of CPU time spent in kernel space.

type: long


**`system.cpu.nice.ticks`**
:   The amount of CPU time spent on low-priority processes.

type: long


**`system.cpu.idle.ticks`**
:   The amount of CPU time spent idle.

type: long


**`system.cpu.iowait.ticks`**
:   The amount of CPU time spent in wait (on disk).

type: long


**`system.cpu.irq.ticks`**
:   The amount of CPU time spent servicing and handling hardware interrupts.

type: long


**`system.cpu.softirq.ticks`**
:   The amount of CPU time spent servicing and handling software interrupts.

type: long


**`system.cpu.steal.ticks`**
:   The amount of CPU time spent in involuntary wait by the virtual CPU while the hypervisor was servicing another processor. Available only on Unix.

type: long



## diskio [_diskio_4]

`disk` contains disk IO metrics collected from the operating system.

**`system.diskio.name`**
:   The disk name.

type: keyword

example: sda1


**`system.diskio.serial_number`**
:   The disk’s serial number. This may not be provided by all operating systems.

type: keyword


**`system.diskio.read.count`**
:   The total number of reads completed successfully.

type: long


**`system.diskio.write.count`**
:   The total number of writes completed successfully.

type: long


**`system.diskio.read.bytes`**
:   The total number of bytes read successfully. On Linux this is the number of sectors read multiplied by an assumed sector size of 512.

type: long

format: bytes


**`system.diskio.write.bytes`**
:   The total number of bytes written successfully. On Linux this is the number of sectors written multiplied by an assumed sector size of 512.

type: long

format: bytes


**`system.diskio.read.time`**
:   The total number of milliseconds spent by all reads.

type: long


**`system.diskio.write.time`**
:   The total number of milliseconds spent by all writes.

type: long


**`system.diskio.io.time`**
:   The total number of of milliseconds spent doing I/Os.

type: long


**`system.diskio.io.ops`**
:   The total number of I/Os in progress.

type: long



## entropy [_entropy_2]

Available system entropy

**`system.entropy.available_bits`**
:   The available bits of entropy

type: long


**`system.entropy.pct`**
:   The percentage of available entropy, relative to the pool size of 4096

type: scaled_float

format: percent



## filesystem [_filesystem_3]

`filesystem` contains local filesystem stats.

**`system.filesystem.available`**
:   The disk space available to an unprivileged user in bytes.

type: long

format: bytes


**`system.filesystem.device_name`**
:   The disk name. For example: `/dev/disk1`

type: keyword


**`system.filesystem.type`**
:   The disk type. For example: `ext4`. In some case for Windows OS the value will be `unavailable` as access to this information is not allowed (ex. external disks).

type: keyword


**`system.filesystem.mount_point`**
:   The mounting point. For example: `/`

type: keyword


**`system.filesystem.files`**
:   Total number of inodes on the system, which will be a combination of files, folders, symlinks, and devices.

type: long


**`system.filesystem.options`**
:   The options present on the filesystem mount.

type: keyword


**`system.filesystem.free`**
:   The disk space available in bytes.

type: long

format: bytes


**`system.filesystem.free_files`**
:   The number of free inodes in the file system.

type: long


**`system.filesystem.total`**
:   The total disk space in bytes.

type: long

format: bytes


**`system.filesystem.used.bytes`**
:   The used disk space in bytes.

type: long

format: bytes


**`system.filesystem.used.pct`**
:   The percentage of used disk space.

type: scaled_float

format: percent



## fsstat [_fsstat_2]

`system.fsstat` contains filesystem metrics aggregated from all mounted filesystems.

**`system.fsstat.count`**
:   Number of file systems found.

type: long


**`system.fsstat.total_files`**
:   Total number of inodes on the system, which will be a combination of files, folders, symlinks, and devices. Not on Windows.

type: long



## total_size [_total_size_2]

Nested file system docs.

**`system.fsstat.total_size.free`**
:   Total free space.

type: long

format: bytes


**`system.fsstat.total_size.used`**
:   Total used space.

type: long

format: bytes


**`system.fsstat.total_size.total`**
:   Total space (used plus free).

type: long

format: bytes



## load [_load_3]

CPU load averages.

**`system.load.1`**
:   Load average for the last minute.

type: scaled_float


**`system.load.5`**
:   Load average for the last 5 minutes.

type: scaled_float


**`system.load.15`**
:   Load average for the last 15 minutes.

type: scaled_float


**`system.load.norm.1`**
:   Load for the last minute divided by the number of cores.

type: scaled_float


**`system.load.norm.5`**
:   Load for the last 5 minutes divided by the number of cores.

type: scaled_float


**`system.load.norm.15`**
:   Load for the last 15 minutes divided by the number of cores.

type: scaled_float


**`system.load.cores`**
:   The number of CPU cores present on the host.

type: long



## memory [_memory_12]

`memory` contains local memory stats.

**`system.memory.total`**
:   Total memory.

type: long

format: bytes


**`system.memory.used.bytes`**
:   Used memory.

type: long

format: bytes


**`system.memory.free`**
:   The total amount of free memory in bytes. This value does not include memory consumed by system caches and buffers (see system.memory.actual.free).

type: long

format: bytes


**`system.memory.cached`**
:   Total Cached memory on system.

type: long

format: bytes


**`system.memory.used.pct`**
:   The percentage of used memory.

type: scaled_float

format: percent



## actual [_actual]

Actual memory used and free.

**`system.memory.actual.used.bytes`**
:   Actual used memory in bytes. It represents the difference between the total and the available memory. The available memory depends on the OS. For more details, please check `system.actual.free`.

type: long

format: bytes


**`system.memory.actual.free`**
:   Actual free memory in bytes. It is calculated based on the OS. On Linux this value will be MemAvailable from /proc/meminfo,  or calculated from free memory plus caches and buffers if /proc/meminfo is not available. On OSX it is a sum of free memory and the inactive memory. On Windows, it is equal to `system.memory.free`.

type: long

format: bytes


**`system.memory.actual.used.pct`**
:   The percentage of actual used memory.

type: scaled_float

format: percent



## swap [_swap_3]

This group contains statistics related to the swap memory usage on the system.

**`system.memory.swap.total`**
:   Total swap memory.

type: long

format: bytes


**`system.memory.swap.used.bytes`**
:   Used swap memory.

type: long

format: bytes


**`system.memory.swap.free`**
:   Available swap memory.

type: long

format: bytes


**`system.memory.swap.used.pct`**
:   The percentage of used swap memory.

type: scaled_float

format: percent



## network [_network_10]

`network` contains network IO metrics for a single network interface.

**`system.network.name`**
:   The network interface name.

type: keyword

example: eth0


**`system.network.out.bytes`**
:   The number of bytes sent.

type: long

format: bytes


**`system.network.in.bytes`**
:   The number of bytes received.

type: long

format: bytes


**`system.network.out.packets`**
:   The number of packets sent.

type: long


**`system.network.in.packets`**
:   The number or packets received.

type: long


**`system.network.in.errors`**
:   The number of errors while receiving.

type: long


**`system.network.out.errors`**
:   The number of errors while sending.

type: long


**`system.network.in.dropped`**
:   The number of incoming packets that were dropped.

type: long


**`system.network.out.dropped`**
:   The number of outgoing packets that were dropped. This value is always 0 on Darwin and BSD because it is not reported by the operating system.

type: long



## network_summary [_network_summary_2]

Metrics relating to global network activity

**`system.network_summary.ip.*`**
:   IP counters

type: object


**`system.network_summary.tcp.*`**
:   TCP counters

type: object


**`system.network_summary.udp.*`**
:   UDP counters

type: object


**`system.network_summary.udp_lite.*`**
:   UDP Lite counters

type: object


**`system.network_summary.icmp.*`**
:   ICMP counters

type: object



## process [_process_9]

`process` contains process metadata, CPU metrics, and memory metrics.

**`system.process.name`**
:   type: alias

alias to: process.name


**`system.process.state`**
:   The process state. For example: "running".

type: keyword


**`system.process.pid`**
:   type: alias

alias to: process.pid


**`system.process.ppid`**
:   type: alias

alias to: process.parent.pid


**`system.process.pgid`**
:   type: alias

alias to: process.pgid


**`system.process.num_threads`**
:   Number of threads in the process

type: integer


**`system.process.cmdline`**
:   The full command-line used to start the process, including the arguments separated by space.

type: keyword


**`system.process.username`**
:   type: alias

alias to: user.name


**`system.process.cwd`**
:   type: alias

alias to: process.working_directory


**`system.process.env`**
:   The environment variables used to start the process. The data is available on FreeBSD, Linux, and OS X.

type: object



## cpu [_cpu_12]

CPU-specific statistics per process.

**`system.process.cpu.user.ticks`**
:   The amount of CPU time the process spent in user space.

type: long


**`system.process.cpu.total.value`**
:   The value of CPU usage since starting the process.

type: long


**`system.process.cpu.total.pct`**
:   The percentage of CPU time spent by the process since the last update. Its value is similar to the %CPU value of the process displayed by the top command on Unix systems.

type: scaled_float

format: percent


**`system.process.cpu.total.norm.pct`**
:   The percentage of CPU time spent by the process since the last event. This value is normalized by the number of CPU cores and it ranges from 0 to 100%.

type: scaled_float

format: percent


**`system.process.cpu.system.ticks`**
:   The amount of CPU time the process spent in kernel space.

type: long


**`system.process.cpu.total.ticks`**
:   The total CPU time spent by the process.

type: long


**`system.process.cpu.start_time`**
:   The time when the process was started.

type: date



## memory [_memory_13]

Memory-specific statistics per process.

**`system.process.memory.size`**
:   The total virtual memory the process has. On Windows this represents the Commit Charge (the total amount of memory that the memory manager has committed for a running process) value in bytes for this process.

type: long

format: bytes


**`system.process.memory.rss.bytes`**
:   The Resident Set Size. The amount of memory the process occupied in main memory (RAM). On Windows this represents the current working set size, in bytes.

type: long

format: bytes


**`system.process.memory.rss.pct`**
:   The percentage of memory the process occupied in main memory (RAM).

type: scaled_float

format: percent


**`system.process.memory.share`**
:   The shared memory the process uses.

type: long

format: bytes



## io [_io]

Disk I/O Metrics, as forwarded from /proc/[PID]/io. Available on Linux only.

**`system.process.io.cancelled_write_bytes`**
:   The number of bytes this process cancelled, or caused not to be written.

type: long


**`system.process.io.read_bytes`**
:   The number of bytes fetched from the storage layer.

type: long


**`system.process.io.write_bytes`**
:   The number of bytes written to the storage layer.

type: long


**`system.process.io.read_char`**
:   The number of bytes read from read(2) and similar syscalls.

type: long


**`system.process.io.write_char`**
:   The number of bytes sent to syscalls for writing.

type: long


**`system.process.io.read_ops`**
:   The count of read-related syscalls.

type: long


**`system.process.io.write_ops`**
:   The count of write-related syscalls.

type: long



## fd [_fd]

File descriptor usage metrics. This set of metrics is available for Linux and FreeBSD.

**`system.process.fd.open`**
:   The number of file descriptors open by the process.

type: long


**`system.process.fd.limit.soft`**
:   The soft limit on the number of file descriptors opened by the process. The soft limit can be changed by the process at any time.

type: long


**`system.process.fd.limit.hard`**
:   The hard limit on the number of file descriptors opened by the process. The hard limit can only be raised by root.

type: long



## cgroup [_cgroup]

Metrics and limits from the cgroup of which the task is a member. cgroup metrics are reported when the process has membership in a non-root cgroup. These metrics are only available on Linux.

**`system.process.cgroup.id`**
:   The ID common to all cgroups associated with this task. If there isn’t a common ID used by all cgroups this field will be absent.

type: keyword


**`system.process.cgroup.path`**
:   The path to the cgroup relative to the cgroup subsystem’s mountpoint. If there isn’t a common path used by all cgroups this field will be absent.

type: keyword


**`system.process.cgroup.cgroups_version`**
:   The version of cgroups reported for the process

type: long



## cpu [_cpu_13]

The cpu subsystem schedules CPU access for tasks in the cgroup. Access can be controlled by two separate schedulers, CFS and RT. CFS stands for completely fair scheduler which proportionally divides the CPU time between cgroups based on weight. RT stands for real time scheduler which sets a maximum amount of CPU time that processes in the cgroup can consume during a given period. In CPU under cgroups V2, the cgroup is merged with many of the metrics from cpuacct. In addition, per-scheduler metrics are gone in V2.

**`system.process.cgroup.cpu.id`**
:   ID of the cgroup.

type: keyword


**`system.process.cgroup.cpu.path`**
:   Path to the cgroup relative to the cgroup subsystem’s mountpoint.

type: keyword



## stats [_stats_12]

cgroupv2 stats

**`system.process.cgroup.cpu.stats.usage.ns`**
:   cgroups v2 usage in nanoseconds

type: long


**`system.process.cgroup.cpu.stats.usage.pct`**
:   cgroups v2 usage

type: float


**`system.process.cgroup.cpu.stats.usage.norm.pct`**
:   cgroups v2 normalized usage

type: float


**`system.process.cgroup.cpu.stats.user.ns`**
:   cgroups v2 cpu user time in nanoseconds

type: long


**`system.process.cgroup.cpu.stats.user.pct`**
:   cgroups v2 cpu user time

type: float


**`system.process.cgroup.cpu.stats.user.norm.pct`**
:   cgroups v2 normalized cpu user time

type: float


**`system.process.cgroup.cpu.stats.system.ns`**
:   cgroups v2 system time in nanoseconds

type: long


**`system.process.cgroup.cpu.stats.system.pct`**
:   cgroups v2 system time

type: float


**`system.process.cgroup.cpu.stats.system.norm.pct`**
:   cgroups v2 normalized system time

type: float


**`system.process.cgroup.cpu.cfs.period.us`**
:   Period of time in microseconds for how regularly a cgroup’s access to CPU resources should be reallocated.

type: long


**`system.process.cgroup.cpu.cfs.quota.us`**
:   Total amount of time in microseconds for which all tasks in a cgroup can run during one period (as defined by cfs.period.us).

type: long


**`system.process.cgroup.cpu.cfs.shares`**
:   An integer value that specifies a relative share of CPU time available to the tasks in a cgroup. The value specified in the cpu.shares file must be 2 or higher.

type: long


**`system.process.cgroup.cpu.rt.period.us`**
:   Period of time in microseconds for how regularly a cgroup’s access to CPU resources is reallocated.

type: long


**`system.process.cgroup.cpu.rt.runtime.us`**
:   Period of time in microseconds for the longest continuous period in which the tasks in a cgroup have access to CPU resources.

type: long


**`system.process.cgroup.cpu.stats.periods`**
:   Number of period intervals (as specified in cpu.cfs.period.us) that have elapsed.

type: long


**`system.process.cgroup.cpu.stats.throttled.periods`**
:   Number of times tasks in a cgroup have been throttled (that is, not allowed to run because they have exhausted all of the available time as specified by their quota).

type: long


**`system.process.cgroup.cpu.stats.throttled.us`**
:   The total time duration (in microseconds) for which tasks in a cgroup have been throttled, as reported by cgroupsv2

type: long


**`system.process.cgroup.cpu.stats.throttled.ns`**
:   The total time duration (in nanoseconds) for which tasks in a cgroup have been throttled.

type: long



## pressure [_pressure_2]

Pressure (resource contention) stats.


## some [_some]

Share of time in which at least some tasks are stalled on a given resource

**`system.process.cgroup.cpu.pressure.some.10.pct`**
:   Pressure over 10 seconds

type: float

format: percent


**`system.process.cgroup.cpu.pressure.some.60.pct`**
:   Pressure over 60 seconds

type: float

format: percent


**`system.process.cgroup.cpu.pressure.some.300.pct`**
:   Pressure over 300 seconds

type: float

format: percent


**`system.process.cgroup.cpu.pressure.some.total`**
:   total Some pressure time

type: long

format: percent



## full [_full]

Share of time in which all non-idle tasks are stalled on a given resource simultaneously

**`system.process.cgroup.cpu.pressure.full.10.pct`**
:   Pressure over 10 seconds

type: float

format: percent


**`system.process.cgroup.cpu.pressure.full.60.pct`**
:   Pressure over 60 seconds

type: float

format: percent


**`system.process.cgroup.cpu.pressure.full.300.pct`**
:   Pressure over 300 seconds

type: float

format: percent


**`system.process.cgroup.cpu.pressure.full.total`**
:   total Full pressure time

type: long



## cpuacct [_cpuacct]

CPU accounting metrics.

**`system.process.cgroup.cpuacct.id`**
:   ID of the cgroup.

type: keyword


**`system.process.cgroup.cpuacct.path`**
:   Path to the cgroup relative to the cgroup subsystem’s mountpoint.

type: keyword


**`system.process.cgroup.cpuacct.total.ns`**
:   Total CPU time in nanoseconds consumed by all tasks in the cgroup.

type: long


**`system.process.cgroup.cpuacct.total.pct`**
:   CPU time of the cgroup as a percentage of overall CPU time.

type: scaled_float


**`system.process.cgroup.cpuacct.total.norm.pct`**
:   CPU time of the cgroup as a percentage of overall CPU time, normalized by CPU count. This is functionally an average of time spent across individual CPUs.

type: scaled_float


**`system.process.cgroup.cpuacct.stats.user.ns`**
:   CPU time consumed by tasks in user mode.

type: long


**`system.process.cgroup.cpuacct.stats.user.pct`**
:   time the cgroup spent in user space, as a percentage of total CPU time

type: scaled_float


**`system.process.cgroup.cpuacct.stats.user.norm.pct`**
:   time the cgroup spent in user space, as a percentage of total CPU time, normalized by CPU count.

type: scaled_float


**`system.process.cgroup.cpuacct.stats.system.ns`**
:   CPU time consumed by tasks in user (kernel) mode.

type: long


**`system.process.cgroup.cpuacct.stats.system.pct`**
:   Time the cgroup spent in kernel space, as a percentage of total CPU time

type: scaled_float


**`system.process.cgroup.cpuacct.stats.system.norm.pct`**
:   Time the cgroup spent in kernel space, as a percentage of total CPU time, normalized by CPU count.

type: scaled_float


**`system.process.cgroup.cpuacct.percpu`**
:   CPU time (in nanoseconds) consumed on each CPU by all tasks in this cgroup.

type: object



## memory [_memory_14]

Memory limits and metrics.

**`system.process.cgroup.memory.id`**
:   ID of the cgroup.

type: keyword


**`system.process.cgroup.memory.path`**
:   Path to the cgroup relative to the cgroup subsystem’s mountpoint.

type: keyword


**`system.process.cgroup.memory.mem.usage.bytes`**
:   Total memory usage by processes in the cgroup (in bytes).

type: long

format: bytes


**`system.process.cgroup.memory.mem.usage.max.bytes`**
:   The maximum memory used by processes in the cgroup (in bytes).

type: long

format: bytes


**`system.process.cgroup.memory.mem.limit.bytes`**
:   The maximum amount of user memory in bytes (including file cache) that tasks in the cgroup are allowed to use.

type: long

format: bytes


**`system.process.cgroup.memory.mem.failures`**
:   The number of times that the memory limit (mem.limit.bytes) was reached.

type: long


**`system.process.cgroup.memory.mem.low.bytes`**
:   memory low threshhold

type: long

format: bytes


**`system.process.cgroup.memory.mem.high.bytes`**
:   memory high threshhold

type: long

format: bytes


**`system.process.cgroup.memory.mem.max.bytes`**
:   memory max threshhold

type: long

format: bytes



## mem.events [_mem_events]

number of times the controller tripped a given usage level

**`system.process.cgroup.memory.mem.events.low`**
:   low threshold

type: long


**`system.process.cgroup.memory.mem.events.high`**
:   high threshold

type: long


**`system.process.cgroup.memory.mem.events.max`**
:   max threshold

type: long


**`system.process.cgroup.memory.mem.events.oom`**
:   oom threshold

type: long


**`system.process.cgroup.memory.mem.events.oom_kill`**
:   oom killer threshold

type: long


**`system.process.cgroup.memory.mem.events.fail`**
:   failed threshold

type: long


**`system.process.cgroup.memory.memsw.usage.bytes`**
:   The sum of current memory usage plus swap space used by processes in the cgroup (in bytes).

type: long

format: bytes


**`system.process.cgroup.memory.memsw.usage.max.bytes`**
:   The maximum amount of memory and swap space used by processes in the cgroup (in bytes).

type: long

format: bytes


**`system.process.cgroup.memory.memsw.limit.bytes`**
:   The maximum amount for the sum of memory and swap usage that tasks in the cgroup are allowed to use.

type: long

format: bytes


**`system.process.cgroup.memory.memsw.low.bytes`**
:   memory low threshhold

type: long

format: bytes


**`system.process.cgroup.memory.memsw.high.bytes`**
:   memory high threshhold

type: long

format: bytes


**`system.process.cgroup.memory.memsw.max.bytes`**
:   memory max threshhold

type: long

format: bytes


**`system.process.cgroup.memory.memsw.failures`**
:   The number of times that the memory plus swap space limit (memsw.limit.bytes) was reached.

type: long



## memsw.events [_memsw_events]

number of times the controller tripped a given usage level

**`system.process.cgroup.memory.memsw.events.low`**
:   low threshold

type: long


**`system.process.cgroup.memory.memsw.events.high`**
:   high threshold

type: long


**`system.process.cgroup.memory.memsw.events.max`**
:   max threshold

type: long


**`system.process.cgroup.memory.memsw.events.oom`**
:   oom threshold

type: long


**`system.process.cgroup.memory.memsw.events.oom_kill`**
:   oom killer threshold

type: long


**`system.process.cgroup.memory.memsw.events.fail`**
:   failed threshold

type: long


**`system.process.cgroup.memory.kmem.usage.bytes`**
:   Total kernel memory usage by processes in the cgroup (in bytes).

type: long

format: bytes


**`system.process.cgroup.memory.kmem.usage.max.bytes`**
:   The maximum kernel memory used by processes in the cgroup (in bytes).

type: long

format: bytes


**`system.process.cgroup.memory.kmem.limit.bytes`**
:   The maximum amount of kernel memory that tasks in the cgroup are allowed to use.

type: long

format: bytes


**`system.process.cgroup.memory.kmem.failures`**
:   The number of times that the memory limit (kmem.limit.bytes) was reached.

type: long


**`system.process.cgroup.memory.kmem_tcp.usage.bytes`**
:   Total memory usage for TCP buffers in bytes.

type: long

format: bytes


**`system.process.cgroup.memory.kmem_tcp.usage.max.bytes`**
:   The maximum memory used for TCP buffers by processes in the cgroup (in bytes).

type: long

format: bytes


**`system.process.cgroup.memory.kmem_tcp.limit.bytes`**
:   The maximum amount of memory for TCP buffers that tasks in the cgroup are allowed to use.

type: long

format: bytes


**`system.process.cgroup.memory.kmem_tcp.failures`**
:   The number of times that the memory limit (kmem_tcp.limit.bytes) was reached.

type: long


**`system.process.cgroup.memory.stats.*`**
:   detailed memory IO stats

type: object


**`system.process.cgroup.memory.stats.*.bytes`**
:   detailed memory IO stats

type: object


**`system.process.cgroup.memory.stats.active_anon.bytes`**
:   Anonymous and swap cache on active least-recently-used (LRU) list, including tmpfs (shmem), in bytes.

type: long

format: bytes


**`system.process.cgroup.memory.stats.active_file.bytes`**
:   File-backed memory on active LRU list, in bytes.

type: long

format: bytes


**`system.process.cgroup.memory.stats.cache.bytes`**
:   Page cache, including tmpfs (shmem), in bytes.

type: long

format: bytes


**`system.process.cgroup.memory.stats.hierarchical_memory_limit.bytes`**
:   Memory limit for the hierarchy that contains the memory cgroup, in bytes.

type: long

format: bytes


**`system.process.cgroup.memory.stats.hierarchical_memsw_limit.bytes`**
:   Memory plus swap limit for the hierarchy that contains the memory cgroup, in bytes.

type: long

format: bytes


**`system.process.cgroup.memory.stats.inactive_anon.bytes`**
:   Anonymous and swap cache on inactive LRU list, including tmpfs (shmem), in bytes

type: long

format: bytes


**`system.process.cgroup.memory.stats.inactive_file.bytes`**
:   File-backed memory on inactive LRU list, in bytes.

type: long

format: bytes


**`system.process.cgroup.memory.stats.mapped_file.bytes`**
:   Size of memory-mapped mapped files, including tmpfs (shmem), in bytes.

type: long

format: bytes


**`system.process.cgroup.memory.stats.page_faults`**
:   Number of times that a process in the cgroup triggered a page fault.

type: long


**`system.process.cgroup.memory.stats.major_page_faults`**
:   Number of times that a process in the cgroup triggered a major fault. "Major" faults happen when the kernel actually has to read the data from disk.

type: long


**`system.process.cgroup.memory.stats.pages_in`**
:   Number of pages paged into memory. This is a counter.

type: long


**`system.process.cgroup.memory.stats.pages_out`**
:   Number of pages paged out of memory. This is a counter.

type: long


**`system.process.cgroup.memory.stats.rss.bytes`**
:   Anonymous and swap cache (includes transparent hugepages), not including tmpfs (shmem), in bytes.

type: long

format: bytes


**`system.process.cgroup.memory.stats.rss_huge.bytes`**
:   Number of bytes of anonymous transparent hugepages.

type: long

format: bytes


**`system.process.cgroup.memory.stats.swap.bytes`**
:   Swap usage, in bytes.

type: long

format: bytes


**`system.process.cgroup.memory.stats.unevictable.bytes`**
:   Memory that cannot be reclaimed, in bytes.

type: long

format: bytes



## blkio [_blkio_2]

Block IO metrics.

**`system.process.cgroup.blkio.id`**
:   ID of the cgroup.

type: keyword


**`system.process.cgroup.blkio.path`**
:   Path to the cgroup relative to the cgroup subsystems mountpoint.

type: keyword


**`system.process.cgroup.blkio.total.bytes`**
:   Total number of bytes transferred to and from all block devices by processes in the cgroup.

type: long

format: bytes


**`system.process.cgroup.blkio.total.ios`**
:   Total number of I/O operations performed on all devices by processes in the cgroup as seen by the throttling policy.

type: long



## io [_io_2]

cgroup V2 IO Metrics, replacing blkio.

**`system.process.cgroup.io.id`**
:   ID of the cgroup.

type: keyword


**`system.process.cgroup.io.path`**
:   Path to the cgroup relative to the cgroup subsystems mountpoint.

type: keyword


**`system.process.cgroup.io.stats.*`**
:   per-device IO usage stats

type: object


**`system.process.cgroup.io.stats.*.*`**
:   type: object


**`system.process.cgroup.io.stats.*.*.bytes`**
:   per-device IO usage stats

type: object


**`system.process.cgroup.io.stats.*.*.ios`**
:   per-device IO usage stats

type: object



## pressure [_pressure_3]

Pressure (resource contention) stats.


## full [_full_2]

Share of time in which at least some tasks are stalled on a given resource

**`system.process.cgroup.io.pressure.full.10.pct`**
:   Pressure over 10 seconds

type: float

format: percent


**`system.process.cgroup.io.pressure.full.60.pct`**
:   Pressure over 60 seconds

type: float

format: percent


**`system.process.cgroup.io.pressure.full.300.pct`**
:   Pressure over 300 seconds

type: float

format: percent


**`system.process.cgroup.io.pressure.full.total`**
:   total Some pressure time

type: long



## some [_some_2]

Share of time in which all tasks are stalled on a given resource

**`system.process.cgroup.io.pressure.some.10.pct`**
:   Pressure over 10 seconds

type: float

format: percent


**`system.process.cgroup.io.pressure.some.60.pct`**
:   Pressure over 60 seconds

type: float

format: percent


**`system.process.cgroup.io.pressure.some.300.pct`**
:   Pressure over 300 seconds

type: float


**`system.process.cgroup.io.pressure.some.total`**
:   total Some pressure time

type: long



## process.summary [_process_summary_2]

Summary metrics for the processes running on the host.

**`system.process.summary.total`**
:   Total number of processes on this host.

type: long


**`system.process.summary.running`**
:   Number of running processes on this host.

type: long


**`system.process.summary.idle`**
:   Number of idle processes on this host.

type: long


**`system.process.summary.sleeping`**
:   Number of sleeping processes on this host.

type: long


**`system.process.summary.stopped`**
:   Number of stopped processes on this host.

type: long


**`system.process.summary.zombie`**
:   Number of zombie processes on this host.

type: long


**`system.process.summary.dead`**
:   Number of dead processes on this host. It’s very unlikely that it will appear but in some special situations it may happen.

type: long


**`system.process.summary.wakekill`**
:   Number of wakekill-state processes on this host. Only found on older Linux Kernel versions.

type: long


**`system.process.summary.wake`**
:   Number of wake-state processes on this host. Only found on older Linux Kernel versions.

type: long


**`system.process.summary.parked`**
:   Number of parked-state processes on this host. Only found on older Linux Kernel versions, or under certain conditions.

type: long


**`system.process.summary.unknown`**
:   Number of processes for which the state couldn’t be retrieved or is unknown.

type: long



## threads [_threads_3]

Counts of individual threads on a system.

**`system.process.summary.threads.running`**
:   Count of currently running threads.

type: long


**`system.process.summary.threads.blocked`**
:   Count of threads blocked by I/O.

type: long



## raid [_raid_2]

raid

**`system.raid.name`**
:   Name of the device.

type: keyword


**`system.raid.status`**
:   activity-state of the device.

type: keyword


**`system.raid.level`**
:   The raid level of the device

type: keyword


**`system.raid.sync_action`**
:   Current sync action, if the RAID array is redundant

type: keyword


**`system.raid.disks.active`**
:   Number of active disks.

type: long


**`system.raid.disks.total`**
:   Total number of disks the device consists of.

type: long


**`system.raid.disks.spare`**
:   Number of spared disks.

type: long


**`system.raid.disks.failed`**
:   Number of failed disks.

type: long


**`system.raid.disks.states.*`**
:   map of raw disk states

type: object


**`system.raid.blocks.total`**
:   Number of blocks the device holds, in 1024-byte blocks.

type: long


**`system.raid.blocks.synced`**
:   Number of blocks on the device that are in sync, in 1024-byte blocks.

type: long



## service [_service_4]

metrics for system services

**`system.service.name`**
:   The name of the service

type: keyword


**`system.service.load_state`**
:   The load state of the service

type: keyword


**`system.service.state`**
:   The activity state of the service

type: keyword


**`system.service.sub_state`**
:   The sub-state of the service

type: keyword


**`system.service.state_since`**
:   The timestamp of the last state change. If the service is active and running, this is its uptime.

type: date


**`system.service.exec_code`**
:   The SIGCHLD code from the service’s main process

type: keyword


**`system.service.unit_file.state`**
:   The state of the unit file

type: keyword


**`system.service.unit_file.vendor_preset`**
:   The default state of the unit file

type: keyword



## resources [_resources_2]

system metrics associated with the service

**`system.service.resources.cpu.usage.ns`**
:   CPU usage in nanoseconds

type: long


**`system.service.resources.memory.usage.bytes`**
:   memory usage in bytes

type: long


**`system.service.resources.tasks.count`**
:   number of tasks associated with the service

type: long



## network [_network_11]

network resource usage

**`system.service.resources.network.in.bytes`**
:   bytes in

type: long

format: bytes


**`system.service.resources.network.in.packets`**
:   packets in

type: long

format: bytes


**`system.service.resources.network.out.packets`**
:   packets out

type: long


**`system.service.resources.network.out.bytes`**
:   bytes out

type: long



## socket [_socket_2]

TCP sockets that are active.

**`system.socket.direction`**
:   type: alias

alias to: network.direction


**`system.socket.family`**
:   type: alias

alias to: network.type


**`system.socket.local.ip`**
:   Local IP address. This can be an IPv4 or IPv6 address.

type: ip

example: 192.0.2.1 or 2001:0DB8:ABED:8536::1


**`system.socket.local.port`**
:   Local port.

type: long

example: 22


**`system.socket.remote.ip`**
:   Remote IP address. This can be an IPv4 or IPv6 address.

type: ip

example: 192.0.2.1 or 2001:0DB8:ABED:8536::1


**`system.socket.remote.port`**
:   Remote port.

type: long

example: 22


**`system.socket.remote.host`**
:   PTR record associated with the remote IP. It is obtained via reverse IP lookup.

type: keyword

example: 76-211-117-36.nw.example.com.


**`system.socket.remote.etld_plus_one`**
:   The effective top-level domain (eTLD) of the remote host plus one more label. For example, the eTLD+1 for "foo.bar.golang.org." is "golang.org.". The data for determining the eTLD comes from an embedded copy of the data from [http://publicsuffix.org](http://publicsuffix.org).

type: keyword

example: example.com.


**`system.socket.remote.host_error`**
:   Error describing the cause of the reverse lookup failure.

type: keyword


**`system.socket.process.pid`**
:   type: alias

alias to: process.pid


**`system.socket.process.command`**
:   type: alias

alias to: process.name


**`system.socket.process.cmdline`**
:   Full command line

type: keyword


**`system.socket.process.exe`**
:   type: alias

alias to: process.executable


**`system.socket.user.id`**
:   type: alias

alias to: user.id


**`system.socket.user.name`**
:   type: alias

alias to: user.full_name



## socket.summary [_socket_summary_2]

Summary metrics of open sockets in the host system


## all [_all]

All connections

**`system.socket.summary.all.count`**
:   All open connections

type: integer


**`system.socket.summary.all.listening`**
:   All listening ports

type: integer



## tcp [_tcp]

All TCP connections

**`system.socket.summary.tcp.memory`**
:   Memory used by TCP sockets in bytes, based on number of allocated pages and system page size. Corresponds to limits set in /proc/sys/net/ipv4/tcp_mem. Only available on Linux.

type: integer

format: bytes



## all [_all_2]

All TCP connections

**`system.socket.summary.tcp.all.orphan`**
:   A count of all orphaned tcp sockets. Only available on Linux.

type: integer


**`system.socket.summary.tcp.all.count`**
:   All open TCP connections

type: integer


**`system.socket.summary.tcp.all.listening`**
:   All TCP listening ports

type: integer


**`system.socket.summary.tcp.all.established`**
:   Number of established TCP connections

type: integer


**`system.socket.summary.tcp.all.close_wait`**
:   Number of TCP connections in *close_wait* state

type: integer


**`system.socket.summary.tcp.all.time_wait`**
:   Number of TCP connections in *time_wait* state

type: integer


**`system.socket.summary.tcp.all.syn_sent`**
:   Number of TCP connections in *syn_sent* state

type: integer


**`system.socket.summary.tcp.all.syn_recv`**
:   Number of TCP connections in *syn_recv* state

type: integer


**`system.socket.summary.tcp.all.fin_wait1`**
:   Number of TCP connections in *fin_wait1* state

type: integer


**`system.socket.summary.tcp.all.fin_wait2`**
:   Number of TCP connections in *fin_wait2* state

type: integer


**`system.socket.summary.tcp.all.last_ack`**
:   Number of TCP connections in *last_ack* state

type: integer


**`system.socket.summary.tcp.all.closing`**
:   Number of TCP connections in *closing* state

type: integer



## udp [_udp]

All UDP connections

**`system.socket.summary.udp.memory`**
:   Memory used by UDP sockets in bytes, based on number of allocated pages and system page size. Corresponds to limits set in /proc/sys/net/ipv4/udp_mem. Only available on Linux.

type: integer

format: bytes



## all [_all_3]

All UDP connections

**`system.socket.summary.udp.all.count`**
:   All open UDP connections

type: integer



## uptime [_uptime_3]

`uptime` contains the operating system uptime metric.

**`system.uptime.duration.ms`**
:   The OS uptime in milliseconds.

type: long

format: duration



## users [_users]

Logged-in user session data

**`system.users.id`**
:   The ID of the session

type: keyword


**`system.users.seat`**
:   An associated logind seat

type: keyword


**`system.users.path`**
:   The DBus object path of the session

type: keyword


**`system.users.type`**
:   The type of the user session

type: keyword


**`system.users.service`**
:   A session associated with the service

type: keyword


**`system.users.remote`**
:   A bool indicating a remote session

type: boolean


**`system.users.state`**
:   The current state of the session

type: keyword


**`system.users.scope`**
:   The associated systemd scope

type: keyword


**`system.users.leader`**
:   The root PID of the session

type: long


**`system.users.remote_host`**
:   A remote host address for the session

type: keyword


