---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-golang.html
---

# Golang fields [exported-fields-golang]

Golang module


## golang [_golang]


## expvar [_expvar]

expvar

**`golang.expvar.cmdline`**
:   The cmdline of this Go program start with.

type: keyword



## heap [_heap]

The Go program heap information exposed by expvar.

**`golang.heap.cmdline`**
:   The cmdline of this Go program start with.

type: keyword



## gc [_gc_2]

Garbage collector summary.


## total_pause [_total_pause]

Total GC pause duration over lifetime of process.

**`golang.heap.gc.total_pause.ns`**
:   Duration in Ns.

type: long


**`golang.heap.gc.total_count`**
:   Total number of GC was happened.

type: long


**`golang.heap.gc.next_gc_limit`**
:   Next collection will happen when HeapAlloc > this amount.

type: long

format: bytes


**`golang.heap.gc.cpu_fraction`**
:   Fraction of CPU time used by GC.

type: float



## pause [_pause_2]

Last GC pause durations during the monitoring period.

**`golang.heap.gc.pause.count`**
:   Count of GC pause duration during this collect period.

type: long



## sum [_sum]

Total GC pause duration during this collect period.

**`golang.heap.gc.pause.sum.ns`**
:   Duration in Ns.

type: long



## max [_max]

Max GC pause duration during this collect period.

**`golang.heap.gc.pause.max.ns`**
:   Duration in Ns.

type: long



## avg [_avg]

Average GC pause duration during this collect period.

**`golang.heap.gc.pause.avg.ns`**
:   Duration in Ns.

type: long



## system [_system_2]

Heap summary,which bytes was obtained from system.

**`golang.heap.system.total`**
:   Total bytes obtained from system (sum of XxxSys below).

type: long

format: bytes


**`golang.heap.system.obtained`**
:   Via HeapSys, bytes obtained from system. heap_sys = heap_idle + heap_inuse.

type: long

format: bytes


**`golang.heap.system.stack`**
:   Bytes used by stack allocator, and these bytes was obtained from system.

type: long

format: bytes


**`golang.heap.system.released`**
:   Bytes released to the OS.

type: long

format: bytes



## allocations [_allocations]

Heap allocations summary.

**`golang.heap.allocations.mallocs`**
:   Number of mallocs.

type: long


**`golang.heap.allocations.frees`**
:   Number of frees.

type: long


**`golang.heap.allocations.objects`**
:   Total number of allocated objects.

type: long


**`golang.heap.allocations.total`**
:   Bytes allocated (even if freed) throughout the lifetime.

type: long

format: bytes


**`golang.heap.allocations.allocated`**
:   Bytes allocated and not yet freed (same as Alloc above).

type: long

format: bytes


**`golang.heap.allocations.idle`**
:   Bytes in idle spans.

type: long

format: bytes


**`golang.heap.allocations.active`**
:   Bytes in non-idle span.

type: long

format: bytes


