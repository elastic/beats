---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-linux.html
---

# Linux fields [exported-fields-linux]

linux module


## linux [_linux]

linux system metrics


## conntrack [_conntrack]

conntrack


## summary [_summary_5]

summary of nf_conntrack statistics, summed across CPU cores

**`linux.conntrack.summary.drop`**
:   packets dropped due to conntrack failiure

type: long


**`linux.conntrack.summary.early_drop`**
:   conntrack entries dropped to make room for new ones

type: long


**`linux.conntrack.summary.entries`**
:   entries in the conntrack table

type: long


**`linux.conntrack.summary.found`**
:   successfully searched entries

type: long


**`linux.conntrack.summary.ignore`**
:   packets seen already connected to a conntrack entry

type: long


**`linux.conntrack.summary.insert_failed`**
:   Number of entries where list insert insert failed

type: long


**`linux.conntrack.summary.invalid`**
:   packets seen that cannot be tracked

type: long


**`linux.conntrack.summary.search_restart`**
:   table lookups which had to be restarted due to table resizes

type: long



## iostat [_iostat]

iostat

**`linux.iostat.read.request.merges_per_sec`**
:   The number of read requests merged per second that were queued to the device.

type: float


**`linux.iostat.write.request.merges_per_sec`**
:   The number of write requests merged per second that were queued to the device.

type: float


**`linux.iostat.read.request.per_sec`**
:   The number of read requests that were issued to the device per second

type: float


**`linux.iostat.write.request.per_sec`**
:   The number of write requests that were issued to the device per second

type: float


**`linux.iostat.read.per_sec.bytes`**
:   The number of Bytes read from the device per second.

type: float

format: bytes


**`linux.iostat.read.await`**
:   The average time spent for read requests issued to the device to be served.

type: float


**`linux.iostat.write.per_sec.bytes`**
:   The number of Bytes write from the device per second.

type: float

format: bytes


**`linux.iostat.write.await`**
:   The average time spent for write requests issued to the device to be served.

type: float


**`linux.iostat.request.avg_size`**
:   The average size (in bytes) of the requests that were issued to the device.

type: float


**`linux.iostat.queue.avg_size`**
:   The average queue length of the requests that were issued to the device.

type: float


**`linux.iostat.await`**
:   The average time spent for requests issued to the device to be served.

type: float


**`linux.iostat.service_time`**
:   The average service time (in milliseconds) for I/O requests that were issued to the device.

type: float


**`linux.iostat.busy`**
:   Percentage of CPU time during which I/O requests were issued to the device (bandwidth utilization for the device). Device saturation occurs when this value is close to 100%.

type: float



## ksm [_ksm]

ksm


## stats [_stats_6]

KSM statistics

**`linux.ksm.stats.pages_shared`**
:   Shared pages in use.

type: long


**`linux.ksm.stats.pages_sharing`**
:   Sites sharing pages.

type: long


**`linux.ksm.stats.pages_unshared`**
:   Unique pages.

type: long


**`linux.ksm.stats.full_scans`**
:   Number of times mergable pages have been scanned.

type: long


**`linux.ksm.stats.stable_node_chains`**
:   Pages that have reached max_page_sharing.

type: long


**`linux.ksm.stats.stable_node_dups`**
:   Number of duplicated KSM pages.

type: long



## memory [_memory_8]

Linux memory data


## page_stats [_page_stats]

memory page statistics

**`linux.memory.page_stats.pgscan_kswapd.pages`**
:   pages scanned by kswapd

type: long

format: number


**`linux.memory.page_stats.pgscan_direct.pages`**
:   pages scanned directly

type: long

format: number


**`linux.memory.page_stats.pgfree.pages`**
:   pages freed by the system

type: long

format: number


**`linux.memory.page_stats.pgsteal_kswapd.pages`**
:   number of pages reclaimed by kswapd

type: long

format: number


**`linux.memory.page_stats.pgsteal_direct.pages`**
:   number of pages reclaimed directly

type: long

format: number


**`linux.memory.page_stats.direct_efficiency.pct`**
:   direct reclaim efficiency percentage. A lower percentage indicates the system is struggling to reclaim memory.

type: scaled_float

format: percent


**`linux.memory.page_stats.kswapd_efficiency.pct`**
:   kswapd reclaim efficiency percentage. A lower percentage indicates the system is struggling to reclaim memory.

type: scaled_float

format: percent



## hugepages [_hugepages]

This group contains statistics related to huge pages usage on the system.

**`linux.memory.hugepages.total`**
:   Number of huge pages in the pool.

type: long

format: number


**`linux.memory.hugepages.used.bytes`**
:   Memory used in allocated huge pages.

type: long

format: bytes


**`linux.memory.hugepages.used.pct`**
:   Percentage of huge pages used.

type: long

format: percent


**`linux.memory.hugepages.free`**
:   Number of available huge pages in the pool.

type: long

format: number


**`linux.memory.hugepages.reserved`**
:   Number of reserved but not allocated huge pages in the pool.

type: long

format: number


**`linux.memory.hugepages.surplus`**
:   Number of overcommited huge pages.

type: long

format: number


**`linux.memory.hugepages.default_size`**
:   Default size for huge pages.

type: long

format: bytes



## swap [_swap_2]

This group contains statistics related to the swap memory usage on the system.

**`linux.memory.swap.total`**
:   Total swap memory.

type: long

format: bytes


**`linux.memory.swap.used.bytes`**
:   Used swap memory.

type: long

format: bytes


**`linux.memory.swap.free`**
:   Available swap memory.

type: long

format: bytes


**`linux.memory.swap.used.pct`**
:   The percentage of used swap memory.

type: scaled_float

format: percent


**`linux.memory.swap.in.pages`**
:   Pages swapped in.

type: long


**`linux.memory.swap.out.pages`**
:   Pages swapped out.

type: long


**`linux.memory.swap.readahead.cached`**
:   Swap readahead pages hit from swap_ra_hit.

type: long


**`linux.memory.swap.readahead.pages`**
:   Pages swapped based on readahead predictions.

type: long



## pageinfo [_pageinfo]

pageinfo


## buddy_info [_buddy_info]

Data from /proc/buddyinfo grouping used pages by order


## DMA [_dma]

DMA page Data

**`linux.pageinfo.buddy_info.DMA.0`**
:   free chunks of 2^0*PAGE_SIZE

type: long


**`linux.pageinfo.buddy_info.DMA.1`**
:   free chunks of 2^1*PAGE_SIZE

type: long


**`linux.pageinfo.buddy_info.DMA.2`**
:   free chunks of 2^2*PAGE_SIZE

type: long


**`linux.pageinfo.buddy_info.DMA.3`**
:   free chunks of 2^3*PAGE_SIZE

type: long


**`linux.pageinfo.buddy_info.DMA.4`**
:   free chunks of 2^4*PAGE_SIZE

type: long


**`linux.pageinfo.buddy_info.DMA.5`**
:   free chunks of 2^5*PAGE_SIZE

type: long


**`linux.pageinfo.buddy_info.DMA.6`**
:   free chunks of 2^6*PAGE_SIZE

type: long


**`linux.pageinfo.buddy_info.DMA.7`**
:   free chunks of 2^7*PAGE_SIZE

type: long


**`linux.pageinfo.buddy_info.DMA.8`**
:   free chunks of 2^8*PAGE_SIZE

type: long


**`linux.pageinfo.buddy_info.DMA.9`**
:   free chunks of 2^9*PAGE_SIZE

type: long


**`linux.pageinfo.buddy_info.DMA.10`**
:   free chunks of 2^10*PAGE_SIZE

type: long


**`linux.pageinfo.nodes.*`**
:   Raw allocation info from /proc/pagetypeinfo

type: object



## pressure [_pressure]

Linux pressure stall information metrics for cpu, memory, and io

**`linux.pressure.cpu.some.10.pct`**
:   The average share of time in which at least some tasks were stalled on CPU over a ten second window.

type: float

format: percent


**`linux.pressure.cpu.some.60.pct`**
:   The average share of time in which at least some tasks were stalled on CPU over a sixty second window.

type: float

format: percent


**`linux.pressure.cpu.some.300.pct`**
:   The average share of time in which at least some tasks were stalled on CPU over a three hundred second window.

type: float

format: percent


**`linux.pressure.cpu.some.total.time.us`**
:   The total absolute stall time (in microseconds) in which at least some tasks were stalled on CPU.

type: long


**`linux.pressure.memory.some.10.pct`**
:   The average share of time in which at least some tasks were stalled on Memory over a ten second window.

type: float

format: percent


**`linux.pressure.memory.some.60.pct`**
:   The average share of time in which at least some tasks were stalled on Memory over a sixty second window.

type: float

format: percent


**`linux.pressure.memory.some.300.pct`**
:   The average share of time in which at least some tasks were stalled on Memory over a three hundred second window.

type: float

format: percent


**`linux.pressure.memory.some.total.time.us`**
:   The total absolute stall time (in microseconds) in which at least some tasks were stalled on memory.

type: long


**`linux.pressure.memory.full.10.pct`**
:   The average share of time in which in which all non-idle tasks were stalled on memory simultaneously over a ten second window.

type: float

format: percent


**`linux.pressure.memory.full.60.pct`**
:   The average share of time in which in which all non-idle tasks were stalled on memory simultaneously over a sixty second window.

type: float

format: percent


**`linux.pressure.memory.full.300.pct`**
:   The average share of time in which in which all non-idle tasks were stalled on memory simultaneously over a three hundred second window.

type: float

format: percent


**`linux.pressure.memory.full.total.time.us`**
:   The total absolute stall time (in microseconds) in which in which all non-idle tasks were stalled on memory.

type: long


**`linux.pressure.io.some.10.pct`**
:   The average share of time in which at least some tasks were stalled on io over a ten second window.

type: float

format: percent


**`linux.pressure.io.some.60.pct`**
:   The average share of time in which at least some tasks were stalled on io over a sixty second window.

type: float

format: percent


**`linux.pressure.io.some.300.pct`**
:   The average share of time in which at least some tasks were stalled on io over a three hundred second window.

type: float

format: percent


**`linux.pressure.io.some.total.time.us`**
:   The total absolute stall time (in microseconds) in which at least some tasks were stalled on io.

type: long


**`linux.pressure.io.full.10.pct`**
:   The average share of time in which in which all non-idle tasks were stalled on io simultaneously over a ten second window.

type: float

format: percent


**`linux.pressure.io.full.60.pct`**
:   The average share of time in which in which all non-idle tasks were stalled on io simultaneously over a sixty second window.

type: float

format: percent


**`linux.pressure.io.full.300.pct`**
:   The average share of time in which in which all non-idle tasks were stalled on io simultaneously over a three hundred second window.

type: float

format: percent


**`linux.pressure.io.full.total.time.us`**
:   The total absolute stall time (in microseconds) in which in which all non-idle tasks were stalled on io.

type: long



## rapl [_rapl]

Wattage as reported by Intel RAPL

**`linux.rapl.core`**
:   The core where the RAPL request originated from. Only one core is queried per hardware CPU.

type: long


**`linux.rapl.dram.watts`**
:   Power usage in watts on the DRAM RAPL domain

type: float


**`linux.rapl.dram.joules`**
:   Raw power usage counter for the DRAM domain

type: float


**`linux.rapl.pp0.watts`**
:   Power usage in watts on the PP0 RAPL domain

type: float


**`linux.rapl.pp0.joules`**
:   Raw power usage counter for the PP0 domain

type: float


**`linux.rapl.pp1.watts`**
:   Power usage in watts on the PP1 RAPL domain

type: float


**`linux.rapl.pp1.joules`**
:   Raw power usage counter for the PP1 domain

type: float


**`linux.rapl.package.watts`**
:   Power usage in watts on the Package RAPL domain

type: float


**`linux.rapl.package.joules`**
:   Raw power usage counter for the package domain

type: float


