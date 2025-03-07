---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-couchbase.html
---

# Couchbase fields [exported-fields-couchbase]

Metrics collected from Couchbase servers.


## couchbase [_couchbase]

`couchbase` contains the metrics that were scraped from Couchbase.


## bucket [_bucket]

Couchbase bucket metrics.

**`couchbase.bucket.name`**
:   Name of the bucket.

type: keyword


**`couchbase.bucket.type`**
:   Type of the bucket.

type: keyword


**`couchbase.bucket.data.used.bytes`**
:   Size of user data within buckets of the specified state that are resident in RAM.

type: long

format: bytes


**`couchbase.bucket.disk.fetches`**
:   Number of disk fetches.

type: double


**`couchbase.bucket.disk.used.bytes`**
:   Amount of disk used (bytes).

type: long

format: bytes


**`couchbase.bucket.memory.used.bytes`**
:   Amount of memory used by the bucket (bytes).

type: long

format: bytes


**`couchbase.bucket.quota.ram.bytes`**
:   Amount of RAM used by the bucket (bytes).

type: long

format: bytes


**`couchbase.bucket.quota.use.pct`**
:   Percentage of RAM used (for active objects) against the configured bucket size (%).

type: scaled_float

format: percent


**`couchbase.bucket.ops_per_sec`**
:   Number of operations per second.

type: double


**`couchbase.bucket.item_count`**
:   Number of items associated with the bucket.

type: long



## cluster [_cluster]

Couchbase cluster metrics.

**`couchbase.cluster.hdd.free.bytes`**
:   Free hard drive space in the cluster (bytes).

type: long

format: bytes


**`couchbase.cluster.hdd.quota.total.bytes`**
:   Hard drive quota total for the cluster (bytes).

type: long

format: bytes


**`couchbase.cluster.hdd.total.bytes`**
:   Total hard drive space available to the cluster (bytes).

type: long

format: bytes


**`couchbase.cluster.hdd.used.value.bytes`**
:   Hard drive space used by the cluster (bytes).

type: long

format: bytes


**`couchbase.cluster.hdd.used.by_data.bytes`**
:   Hard drive space used by the data in the cluster (bytes).

type: long

format: bytes


**`couchbase.cluster.max_bucket_count`**
:   Max bucket count setting.

type: long


**`couchbase.cluster.quota.index_memory.mb`**
:   Memory quota setting for the Index service (Mbyte).

type: double


**`couchbase.cluster.quota.memory.mb`**
:   Memory quota setting for the cluster (Mbyte).

type: double


**`couchbase.cluster.ram.quota.total.value.bytes`**
:   RAM quota total for the cluster (bytes).

type: long

format: bytes


**`couchbase.cluster.ram.quota.total.per_node.bytes`**
:   RAM quota used by the current node in the cluster (bytes).

type: long

format: bytes


**`couchbase.cluster.ram.quota.used.value.bytes`**
:   RAM quota used by the cluster (bytes).

type: long

format: bytes


**`couchbase.cluster.ram.quota.used.per_node.bytes`**
:   Ram quota used by the current node in the cluster (bytes)

type: long

format: bytes


**`couchbase.cluster.ram.total.bytes`**
:   Total RAM available to cluster (bytes).

type: long

format: bytes


**`couchbase.cluster.ram.used.value.bytes`**
:   RAM used by the cluster (bytes).

type: long

format: bytes


**`couchbase.cluster.ram.used.by_data.bytes`**
:   RAM used by the data in the cluster (bytes).

type: long

format: bytes



## node [_node]

Couchbase node metrics.

**`couchbase.node.cmd_get`**
:   Number of get commands

type: double


**`couchbase.node.couch.docs.disk_size.bytes`**
:   Amount of disk space used by Couch docs (bytes).

type: long

format: bytes


**`couchbase.node.couch.docs.data_size.bytes`**
:   Data size of Couch docs associated with a node (bytes).

type: long

format: bytes


**`couchbase.node.couch.spatial.data_size.bytes`**
:   Size of object data for spatial views (bytes).

type: long


**`couchbase.node.couch.spatial.disk_size.bytes`**
:   Amount of disk space used by spatial views (bytes).

type: long


**`couchbase.node.couch.views.disk_size.bytes`**
:   Amount of disk space used by Couch views (bytes).

type: long


**`couchbase.node.couch.views.data_size.bytes`**
:   Size of object data for Couch views (bytes).

type: long


**`couchbase.node.cpu_utilization_rate.pct`**
:   The CPU utilization rate (%).

type: scaled_float


**`couchbase.node.current_items.value`**
:   Number of current items.

type: long


**`couchbase.node.current_items.total`**
:   Total number of items associated with the node.

type: long


**`couchbase.node.ep_bg_fetched`**
:   Number of disk fetches performed since the server was started.

type: long


**`couchbase.node.get_hits`**
:   Number of get hits.

type: double


**`couchbase.node.hostname`**
:   The hostname of the node.

type: keyword


**`couchbase.node.mcd_memory.allocated.bytes`**
:   Amount of memcached memory allocated (bytes).

type: long

format: bytes


**`couchbase.node.mcd_memory.reserved.bytes`**
:   Amount of memcached memory reserved (bytes).

type: long


**`couchbase.node.memory.free.bytes`**
:   Amount of memory free for the node (bytes).

type: long


**`couchbase.node.memory.total.bytes`**
:   Total memory available to the node (bytes).

type: long


**`couchbase.node.memory.used.bytes`**
:   Memory used by the node (bytes).

type: long


**`couchbase.node.ops`**
:   Number of operations performed on Couchbase.

type: double


**`couchbase.node.swap.total.bytes`**
:   Total swap size allocated (bytes).

type: long


**`couchbase.node.swap.used.bytes`**
:   Amount of swap space used (bytes).

type: long


**`couchbase.node.uptime.sec`**
:   Time during which the node was in operation (sec).

type: long


**`couchbase.node.vb_replica_curr_items`**
:   Number of items/documents that are replicas.

type: long


