---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-zookeeper.html
---

# ZooKeeper fields [exported-fields-zookeeper]

ZooKeeper metrics collected by the four-letter monitoring commands.


## zookeeper [_zookeeper]

`zookeeper` contains the metrics reported by ZooKeeper commands.


## connection [_connection_5]

connections

**`zookeeper.connection.interest_ops`**
:   Interest ops

type: long


**`zookeeper.connection.queued`**
:   Queued connections

type: long


**`zookeeper.connection.received`**
:   Received connections

type: long


**`zookeeper.connection.sent`**
:   Connections sent

type: long



## mntr [_mntr]

`mntr` contains the metrics reported by the four-letter `mntr` command.

**`zookeeper.mntr.approximate_data_size`**
:   Approximate size of ZooKeeper data.

type: long


**`zookeeper.mntr.latency.avg`**
:   Average latency between ensemble hosts in milliseconds.

type: long


**`zookeeper.mntr.ephemerals_count`**
:   Number of ephemeral znodes.

type: long


**`zookeeper.mntr.followers`**
:   Number of followers seen by the current host (Up to ZooKeeper 3.5.9).

type: long


**`zookeeper.mntr.max_file_descriptor_count`**
:   Maximum number of file descriptors allowed for the ZooKeeper process.

type: long


**`zookeeper.mntr.latency.max`**
:   Maximum latency in milliseconds.

type: long


**`zookeeper.mntr.latency.min`**
:   Minimum latency in milliseconds.

type: long


**`zookeeper.mntr.num_alive_connections`**
:   Number of connections to ZooKeeper that are currently alive.

type: long


**`zookeeper.mntr.open_file_descriptor_count`**
:   Number of file descriptors open by the ZooKeeper process.

type: long


**`zookeeper.mntr.outstanding_requests`**
:   Number of outstanding requests that need to be processed by the cluster.

type: long


**`zookeeper.mntr.packets.received`**
:   Number of ZooKeeper network packets received.

type: long


**`zookeeper.mntr.packets.sent`**
:   Number of ZooKeeper network packets sent.

type: long


**`zookeeper.mntr.pending_syncs`**
:   Number of pending syncs to carry out to ZooKeeper ensemble followers.

type: long


**`zookeeper.mntr.server_state`**
:   Role in the ZooKeeper ensemble.

type: keyword


**`zookeeper.mntr.synced_followers`**
:   Number of synced followers reported when a node server_state is leader.

type: long


**`zookeeper.mntr.version`**
:   ZooKeeper version and build string reported.

type: alias

alias to: service.version


**`zookeeper.mntr.watch_count`**
:   Number of watches currently set on the local ZooKeeper process.

type: long


**`zookeeper.mntr.znode_count`**
:   Number of znodes reported by the local ZooKeeper process.

type: long


**`zookeeper.mntr.learners`**
:   Number of learners (either followers or observers) seen by the current host (From ZooKeeper 3.6.0)

type: long



## server [_server_10]

server contains the metrics reported by the four-letter `srvr` command.

**`zookeeper.server.connections`**
:   Number of clients currently connected to the server

type: long


**`zookeeper.server.latency.avg`**
:   Average amount of time taken for the server to respond to a client request

type: long


**`zookeeper.server.latency.max`**
:   Maximum amount of time taken for the server to respond to a client request

type: long


**`zookeeper.server.latency.min`**
:   Minimum amount of time taken for the server to respond to a client request

type: long


**`zookeeper.server.mode`**
:   Mode of the server. In an ensemble, this may either be leader or follower. Otherwise, it is standalone

type: keyword


**`zookeeper.server.node_count`**
:   Total number of nodes

type: long


**`zookeeper.server.outstanding`**
:   Number of requests queued at the server. This exceeds zero when the server receives more requests than it is able to process

type: long


**`zookeeper.server.received`**
:   Number of requests received by the server

type: long


**`zookeeper.server.sent`**
:   Number of requests sent by the server

type: long


**`zookeeper.server.version_date`**
:   Date of the Zookeeper release currently in use

type: date


**`zookeeper.server.zxid`**
:   Unique value of the Zookeeper transaction ID. The zxid consists of an epoch and a counter. It is established by the leader and is used to determine the temporal ordering of changes

type: keyword


**`zookeeper.server.count`**
:   Total transactions of the leader in epoch

type: long


**`zookeeper.server.epoch`**
:   Epoch value of the Zookeeper transaction ID. An epoch signifies the period in which a server is a leader

type: long


