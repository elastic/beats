---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-ceph.html
---

# Ceph fields [exported-fields-ceph]

Ceph module


## ceph [_ceph]

`ceph` contains the metrics that were scraped from CEPH.


## cluster_disk [_cluster_disk]

cluster_disk

**`ceph.cluster_disk.available.bytes`**
:   Available bytes of the cluster

type: long

format: bytes


**`ceph.cluster_disk.total.bytes`**
:   Total bytes of the cluster

type: long

format: bytes


**`ceph.cluster_disk.used.bytes`**
:   Used bytes of the cluster

type: long

format: bytes



## cluster_health [_cluster_health]

cluster_health

**`ceph.cluster_health.overall_status`**
:   Overall status of the cluster

type: keyword


**`ceph.cluster_health.timechecks.epoch`**
:   Map version

type: long


**`ceph.cluster_health.timechecks.round.value`**
:   timecheck round

type: long


**`ceph.cluster_health.timechecks.round.status`**
:   Status of the round

type: keyword



## cluster_status [_cluster_status]

cluster_status

**`ceph.cluster_status.version`**
:   Ceph Status version

type: long


**`ceph.cluster_status.traffic.read_bytes`**
:   Cluster read throughput per second

type: long

format: bytes


**`ceph.cluster_status.traffic.write_bytes`**
:   Cluster write throughput per second

type: long

format: bytes


**`ceph.cluster_status.traffic.read_op_per_sec`**
:   Cluster read iops per second

type: long


**`ceph.cluster_status.traffic.write_op_per_sec`**
:   Cluster write iops per second

type: long


**`ceph.cluster_status.misplace.total`**
:   Cluster misplace pg number

type: long


**`ceph.cluster_status.misplace.objects`**
:   Cluster misplace objects number

type: long


**`ceph.cluster_status.misplace.ratio`**
:   Cluster misplace ratio

type: scaled_float

format: percent


**`ceph.cluster_status.degraded.total`**
:   Cluster degraded pg number

type: long


**`ceph.cluster_status.degraded.objects`**
:   Cluster degraded objects number

type: long


**`ceph.cluster_status.degraded.ratio`**
:   Cluster degraded ratio

type: scaled_float

format: percent


**`ceph.cluster_status.pg.data_bytes`**
:   Cluster pg data bytes

type: long

format: bytes


**`ceph.cluster_status.pg.avail_bytes`**
:   Cluster available bytes

type: long

format: bytes


**`ceph.cluster_status.pg.total_bytes`**
:   Cluster total bytes

type: long

format: bytes


**`ceph.cluster_status.pg.used_bytes`**
:   Cluster used bytes

type: long

format: bytes


**`ceph.cluster_status.pg_state.state_name`**
:   Pg state description

type: long


**`ceph.cluster_status.pg_state.count`**
:   Shows how many pgs are in state of pg_state.state_name

type: long


**`ceph.cluster_status.pg_state.version`**
:   Cluster status version

type: long


**`ceph.cluster_status.osd.full`**
:   Is osd full

type: boolean


**`ceph.cluster_status.osd.nearfull`**
:   Is osd near full

type: boolean


**`ceph.cluster_status.osd.num_osds`**
:   Shows how many osds in the cluster

type: long


**`ceph.cluster_status.osd.num_up_osds`**
:   Shows how many osds are on the state of UP

type: long


**`ceph.cluster_status.osd.num_in_osds`**
:   Shows how many osds are on the state of IN

type: long


**`ceph.cluster_status.osd.num_remapped_pgs`**
:   Shows how many osds are on the state of REMAPPED

type: long


**`ceph.cluster_status.osd.epoch`**
:   epoch number

type: long



## mgr_cluster_disk [_mgr_cluster_disk]

see: cluster_disk


## mgr_cluster_health [_mgr_cluster_health]

see: cluster_health


## mgr_osd_perf [_mgr_osd_perf]

OSD performance metrics of Ceph cluster

**`ceph.mgr_osd_perf.id`**
:   OSD ID

type: long


**`ceph.mgr_osd_perf.stats.commit_latency_ms`**
:   Commit latency in ms

type: long


**`ceph.mgr_osd_perf.stats.apply_latency_ms`**
:   Apply latency in ms

type: long


**`ceph.mgr_osd_perf.stats.commit_latency_ns`**
:   Commit latency in ns

type: long


**`ceph.mgr_osd_perf.stats.apply_latency_ns`**
:   Apply latency in ns

type: long



## mgr_osd_pool_stats [_mgr_osd_pool_stats]

OSD pool stats of Ceph cluster

**`ceph.mgr_osd_pool_stats.pool_name`**
:   Pool name

type: keyword


**`ceph.mgr_osd_pool_stats.pool_id`**
:   Pool ID

type: long


**`ceph.mgr_osd_pool_stats.client_io_rate`**
:   Client I/O rates

type: object



## mgr_osd_tree [_mgr_osd_tree]

see: osd_tree


## mgr_pool_disk [_mgr_pool_disk]

see: pool_disk


## monitor_health [_monitor_health]

monitor_health stats data

**`ceph.monitor_health.available.pct`**
:   Available percent of the MON

type: long


**`ceph.monitor_health.health`**
:   Health of the MON

type: keyword


**`ceph.monitor_health.available.kb`**
:   Available KB of the MON

type: long


**`ceph.monitor_health.total.kb`**
:   Total KB of the MON

type: long


**`ceph.monitor_health.used.kb`**
:   Used KB of the MON

type: long


**`ceph.monitor_health.last_updated`**
:   Time when was updated

type: date


**`ceph.monitor_health.name`**
:   Name of the MON

type: keyword


**`ceph.monitor_health.store_stats.log.bytes`**
:   Log bytes of MON

type: long

format: bytes


**`ceph.monitor_health.store_stats.misc.bytes`**
:   Misc bytes of MON

type: long

format: bytes


**`ceph.monitor_health.store_stats.sst.bytes`**
:   SST bytes of MON

type: long

format: bytes


**`ceph.monitor_health.store_stats.total.bytes`**
:   Total bytes of MON

type: long

format: bytes


**`ceph.monitor_health.store_stats.last_updated`**
:   Last updated

type: long



## osd_df [_osd_df]

ceph osd disk usage information

**`ceph.osd_df.id`**
:   osd node id

type: long


**`ceph.osd_df.name`**
:   osd node name

type: keyword


**`ceph.osd_df.device_class`**
:   osd node type, illegal type include hdd, ssd etc.

type: keyword


**`ceph.osd_df.total.byte`**
:   osd disk total volume

type: long

format: bytes


**`ceph.osd_df.used.byte`**
:   osd disk usage volume

type: long

format: bytes


**`ceph.osd_df.available.bytes`**
:   osd disk available volume

type: long

format: bytes


**`ceph.osd_df.pg_num`**
:   shows how many pg located on this osd

type: long


**`ceph.osd_df.used.pct`**
:   osd disk usage percentage

type: scaled_float

format: percent



## osd_tree [_osd_tree]

ceph osd tree info

**`ceph.osd_tree.id`**
:   osd or bucket node id

type: long


**`ceph.osd_tree.name`**
:   osd or bucket node name

type: keyword


**`ceph.osd_tree.type`**
:   osd or bucket node type, illegal type include osd, host, root etc.

type: keyword


**`ceph.osd_tree.type_id`**
:   osd or bucket node typeID

type: long


**`ceph.osd_tree.children`**
:   bucket children list, separated by comma.

type: keyword


**`ceph.osd_tree.crush_weight`**
:   osd node crush weight

type: float


**`ceph.osd_tree.depth`**
:   node depth

type: long


**`ceph.osd_tree.exists`**
:   is node still exist or not(1-yes, 0-no)

type: boolean


**`ceph.osd_tree.primary_affinity`**
:   the weight of reading data from primary osd

type: float


**`ceph.osd_tree.reweight`**
:   the reweight of osd

type: long


**`ceph.osd_tree.status`**
:   status of osd, it should be up or down

type: keyword


**`ceph.osd_tree.device_class`**
:   the device class of osd, like hdd, ssd etc.

type: keyword


**`ceph.osd_tree.father`**
:   the parent node of this osd or bucket node

type: keyword



## pool_disk [_pool_disk]

pool_disk

**`ceph.pool_disk.id`**
:   Id of the pool

type: long


**`ceph.pool_disk.name`**
:   Name of the pool

type: keyword


**`ceph.pool_disk.stats.available.bytes`**
:   Available bytes of the pool

type: long

format: bytes


**`ceph.pool_disk.stats.objects`**
:   Number of objects of the pool

type: long


**`ceph.pool_disk.stats.used.bytes`**
:   Used bytes of the pool

type: long

format: bytes


**`ceph.pool_disk.stats.used.kb`**
:   Used kb of the pool

type: long


