---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-etcd.html
---

# Etcd fields [exported-fields-etcd]

etcd Module


## etcd [_etcd]

`etcd` contains statistics that were read from Etcd

**`etcd.api_version`**
:   Etcd API version for metrics retrieval

type: keyword



## leader [_leader]

Contains etcd leader statistics.


## follower [_follower]

Contains follower statistics.

**`etcd.leader.follower.id`**
:   ID of follower

type: keyword



## latency [_latency]

latency to each peer in the cluster

**`etcd.leader.follower.latency.ms`**
:   type: scaled_float


**`etcd.leader.follower.success_operations`**
:   successful Raft RPC requests

type: integer


**`etcd.leader.follower.failed_operations`**
:   failed Raft RPC requests

type: integer


**`etcd.leader.follower.leader`**
:   ID of actual leader

type: keyword



## server [_server_5]

Server metrics from the Etcd V3 /metrics endpoint

**`etcd.server.has_leader`**
:   Whether a leader exists in the cluster

type: byte


**`etcd.server.leader_changes.count`**
:   Number of leader changes seen at the cluster

type: long


**`etcd.server.proposals_committed.count`**
:   Number of consensus proposals commited

type: long


**`etcd.server.proposals_pending.count`**
:   Number of consensus proposals pending

type: long


**`etcd.server.proposals_failed.count`**
:   Number of consensus proposals failed

type: long


**`etcd.server.grpc_started.count`**
:   Number of sent gRPC requests

type: long


**`etcd.server.grpc_handled.count`**
:   Number of received gRPC requests

type: long



## disk [_disk]

Disk metrics from the Etcd V3 /metrics endpoint

**`etcd.disk.mvcc_db_total_size.bytes`**
:   Size of stored data at MVCC

type: long

format: bytes


**`etcd.disk.wal_fsync_duration.ns.bucket.*`**
:   Latency for writing ahead logs to disk

type: object


**`etcd.disk.wal_fsync_duration.ns.count`**
:   Write ahead logs count

type: long


**`etcd.disk.wal_fsync_duration.ns.sum`**
:   Write ahead logs latency sum

type: long


**`etcd.disk.backend_commit_duration.ns.bucket.*`**
:   Latency for writing backend changes to disk

type: object


**`etcd.disk.backend_commit_duration.ns.count`**
:   Backend commits count

type: long


**`etcd.disk.backend_commit_duration.ns.sum`**
:   Backend commits latency sum

type: long



## memory [_memory_6]

Memory metrics from the Etcd V3 /metrics endpoint

**`etcd.memory.go_memstats_alloc.bytes`**
:   Memory allocated bytes as of MemStats Go

type: long

format: bytes



## network [_network_5]

Network metrics from the Etcd V3 /metrics endpoint

**`etcd.network.client_grpc_sent.bytes`**
:   gRPC sent bytes total

type: long

format: bytes


**`etcd.network.client_grpc_received.bytes`**
:   gRPC received bytes total

type: long

format: bytes



## self [_self]

Contains etcd self statistics.

**`etcd.self.id`**
:   the unique identifier for the member

type: keyword


**`etcd.self.leaderinfo.leader`**
:   id of the current leader member

type: keyword


**`etcd.self.leaderinfo.starttime`**
:   the time when this node was started

type: keyword


**`etcd.self.leaderinfo.uptime`**
:   amount of time the leader has been leader

type: keyword


**`etcd.self.name`**
:   this member’s name

type: keyword


**`etcd.self.recv.appendrequest.count`**
:   number of append requests this node has processed

type: integer


**`etcd.self.recv.bandwidthrate`**
:   number of bytes per second this node is receiving (follower only)

type: scaled_float


**`etcd.self.recv.pkgrate`**
:   number of requests per second this node is receiving (follower only)

type: scaled_float


**`etcd.self.send.appendrequest.count`**
:   number of requests that this node has sent

type: integer


**`etcd.self.send.bandwidthrate`**
:   number of bytes per second this node is sending (leader only). This value is undefined on single member clusters.

type: scaled_float


**`etcd.self.send.pkgrate`**
:   number of requests per second this node is sending (leader only). This value is undefined on single member clusters.

type: scaled_float


**`etcd.self.starttime`**
:   the time when this node was started

type: keyword


**`etcd.self.state`**
:   either leader or follower

type: keyword



## store [_store]

The store statistics include information about the operations that this node has handled.

**`etcd.store.gets.success`**
:   type: integer


**`etcd.store.gets.fail`**
:   type: integer


**`etcd.store.sets.success`**
:   type: integer


**`etcd.store.sets.fail`**
:   type: integer


**`etcd.store.delete.success`**
:   type: integer


**`etcd.store.delete.fail`**
:   type: integer


**`etcd.store.update.success`**
:   type: integer


**`etcd.store.update.fail`**
:   type: integer


**`etcd.store.create.success`**
:   type: integer


**`etcd.store.create.fail`**
:   type: integer


**`etcd.store.compareandswap.success`**
:   type: integer


**`etcd.store.compareandswap.fail`**
:   type: integer


**`etcd.store.compareanddelete.success`**
:   type: integer


**`etcd.store.compareanddelete.fail`**
:   type: integer


**`etcd.store.expire.count`**
:   type: integer


**`etcd.store.watchers`**
:   type: integer


