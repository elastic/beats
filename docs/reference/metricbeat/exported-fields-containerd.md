---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-containerd.html
---

# Containerd fields [exported-fields-containerd]

Containerd stats collected from containerd


## containerd [_containerd]

Information and statistics about containerd’s running containers.

**`containerd.namespace`**
:   Containerd namespace

type: keyword



## blkio [_blkio]

Block I/O metrics.

**`containerd.blkio.device`**
:   Name of block device

type: keyword



## read [_read_4]

Accumulated reads during the life of the container

**`containerd.blkio.read.ops`**
:   Number of reads during the life of the container

type: long


**`containerd.blkio.read.bytes`**
:   Bytes read during the life of the container

type: long

format: bytes



## write [_write_4]

Accumulated writes during the life of the container

**`containerd.blkio.write.ops`**
:   Number of writes during the life of the container

type: long


**`containerd.blkio.write.bytes`**
:   Bytes written during the life of the container

type: long

format: bytes



## summary [_summary_2]

Accumulated reads and writes during the life of the container

**`containerd.blkio.summary.ops`**
:   Number of I/O operations during the life of the container

type: long


**`containerd.blkio.summary.bytes`**
:   Bytes read and written during the life of the container

type: long

format: bytes



## cpu [_cpu_4]

Containerd Runtime CPU metrics.

**`containerd.cpu.system.total`**
:   Total user and system CPU time spent in seconds.

type: double


**`containerd.cpu.usage.kernel.ns`**
:   CPU Kernel usage nanoseconds

type: double


**`containerd.cpu.usage.user.ns`**
:   CPU User usage nanoseconds

type: double


**`containerd.cpu.usage.total.ns`**
:   CPU total usage nanoseconds

type: double


**`containerd.cpu.usage.total.pct`**
:   Percentage of total CPU time normalized by the number of CPU cores, expressed as a value between 0 and 1.

type: scaled_float

format: percent


**`containerd.cpu.usage.kernel.pct`**
:   Percentage of time in kernel space normalized by the number of CPU cores, expressed as a value between 0 and 1.

type: scaled_float

format: percent


**`containerd.cpu.usage.user.pct`**
:   Percentage of time in user space normalized by the number of CPU cores, expressed as a value between 0 and 1.

type: scaled_float

format: percent


**`containerd.cpu.usage.cpu.*.ns`**
:   CPU usage nanoseconds in this cpu.

type: object



## memory [_memory_4]

memory

**`containerd.memory.workingset.pct`**
:   Memory working set percentage.

type: scaled_float

format: percent


**`containerd.memory.rss`**
:   Total memory resident set size.

type: long

format: bytes


**`containerd.memory.activeFiles`**
:   Total active file bytes.

type: long

format: bytes


**`containerd.memory.cache`**
:   Total cache bytes.

type: long

format: bytes


**`containerd.memory.inactiveFiles`**
:   Total inactive file bytes.

type: long

format: bytes



## usage [_usage_12]

Usage memory stats.

**`containerd.memory.usage.max`**
:   Max memory usage.

type: long

format: bytes


**`containerd.memory.usage.pct`**
:   Total allocated memory percentage.

type: scaled_float

format: percent


**`containerd.memory.usage.total`**
:   Total memory usage.

type: long

format: bytes


**`containerd.memory.usage.fail.count`**
:   Fail counter.

type: scaled_float


**`containerd.memory.usage.limit`**
:   Memory usage limit.

type: long

format: bytes



## kernel [_kernel]

Kernel memory stats.

**`containerd.memory.kernel.max`**
:   Kernel max memory usage.

type: long

format: bytes


**`containerd.memory.kernel.total`**
:   Kernel total memory usage.

type: long

format: bytes


**`containerd.memory.kernel.fail.count`**
:   Kernel fail counter.

type: scaled_float


**`containerd.memory.kernel.limit`**
:   Kernel memory limit.

type: long

format: bytes



## swap [_swap]

Swap memory stats.

**`containerd.memory.swap.max`**
:   Swap max memory usage.

type: long

format: bytes


**`containerd.memory.swap.total`**
:   Swap total memory usage.

type: long

format: bytes


**`containerd.memory.swap.fail.count`**
:   Swap fail counter.

type: scaled_float


**`containerd.memory.swap.limit`**
:   Swap memory limit.

type: long

format: bytes


