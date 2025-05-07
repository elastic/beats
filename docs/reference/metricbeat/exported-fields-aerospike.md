---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-aerospike.html
---

# Aerospike fields [exported-fields-aerospike]

Aerospike module


## aerospike [_aerospike]


## namespace [_namespace]

namespace


## client [_client]

Client stats.


## delete [_delete]

Client delete transactions stats.

**`aerospike.namespace.client.delete.error`**
:   Number of client delete transactions that failed with an error.

type: long


**`aerospike.namespace.client.delete.not_found`**
:   Number of client delete transactions that resulted in a not found.

type: long


**`aerospike.namespace.client.delete.success`**
:   Number of successful client delete transactions.

type: long


**`aerospike.namespace.client.delete.timeout`**
:   Number of client delete transactions that timed out.

type: long



## read [_read]

Client read transactions stats.

**`aerospike.namespace.client.read.error`**
:   Number of client read transaction errors.

type: long


**`aerospike.namespace.client.read.not_found`**
:   Number of client read transaction that resulted in not found.

type: long


**`aerospike.namespace.client.read.success`**
:   Number of successful client read transactions.

type: long


**`aerospike.namespace.client.read.timeout`**
:   Number of client read transaction that timed out.

type: long



## write [_write]

Client write transactions stats.

**`aerospike.namespace.client.write.error`**
:   Number of client write transactions that failed with an error.

type: long


**`aerospike.namespace.client.write.success`**
:   Number of successful client write transactions.

type: long


**`aerospike.namespace.client.write.timeout`**
:   Number of client write transactions that timed out.

type: long



## device [_device]

Disk storage stats

**`aerospike.namespace.device.available.pct`**
:   Measures the minimum contiguous disk space across all disks in a namespace.

type: scaled_float

format: percent


**`aerospike.namespace.device.free.pct`**
:   Percentage of disk capacity free for this namespace.

type: scaled_float

format: percent


**`aerospike.namespace.device.total.bytes`**
:   Total bytes of disk space allocated to this namespace on this node.

type: long

format: bytes


**`aerospike.namespace.device.used.bytes`**
:   Total bytes of disk space used by this namespace on this node.

type: long

format: bytes


**`aerospike.namespace.hwm_breached`**
:   If true, Aerospike has breached *high-water-[disk|memory]-pct* for this namespace.

type: boolean



## memory [_memory_2]

Memory storage stats.

**`aerospike.namespace.memory.free.pct`**
:   Percentage of memory capacity free for this namespace on this node.

type: scaled_float

format: percent


**`aerospike.namespace.memory.used.data.bytes`**
:   Amount of memory occupied by data for this namespace on this node.

type: long

format: bytes


**`aerospike.namespace.memory.used.index.bytes`**
:   Amount of memory occupied by the index for this namespace on this node.

type: long

format: bytes


**`aerospike.namespace.memory.used.sindex.bytes`**
:   Amount of memory occupied by secondary indexes for this namespace on this node.

type: long

format: bytes


**`aerospike.namespace.memory.used.total.bytes`**
:   Total bytes of memory used by this namespace on this node.

type: long

format: bytes


**`aerospike.namespace.name`**
:   Namespace name

type: keyword


**`aerospike.namespace.node.host`**
:   Node host

type: keyword


**`aerospike.namespace.node.name`**
:   Node name

type: keyword



## objects [_objects]

Records stats.

**`aerospike.namespace.objects.master`**
:   Number of records on this node which are active masters.

type: long


**`aerospike.namespace.objects.total`**
:   Number of records in this namespace for this node.

type: long


**`aerospike.namespace.stop_writes`**
:   If true this namespace is currently not allowing writes.

type: boolean


