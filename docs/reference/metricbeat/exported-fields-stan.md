---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-stan.html
---

# Stan fields [exported-fields-stan]

stan Module


## stan [_stan]

`stan` contains statistics that were read from Nats Streaming server (STAN)

**`stan.server.id`**
:   The server ID

type: keyword


**`stan.cluster.id`**
:   The cluster ID

type: keyword



## channels [_channels]

Contains stan / nats streaming/serverz endpoint metrics

**`stan.channels.name`**
:   The name of the STAN streaming channel

type: keyword


**`stan.channels.messages`**
:   The number of STAN streaming messages

type: long


**`stan.channels.bytes`**
:   The number of STAN bytes in the channel

type: long


**`stan.channels.first_seq`**
:   First sequence number stored in the channel. If first_seq > min([seq in subscriptions]) data loss has possibly occurred

type: long


**`stan.channels.last_seq`**
:   Last sequence number stored in the channel

type: long


**`stan.channels.depth`**
:   Queue depth based upon current sequence number and highest reported subscriber sequence number

type: long



## stats [_stats_11]

Contains only high-level stan / nats streaming server related metrics

**`stan.stats.state`**
:   The cluster / streaming configuration state (STANDALONE, CLUSTERED)

type: keyword


**`stan.stats.role`**
:   If clustered, role of this node in the cluster (Leader, Follower, Candidate)

type: keyword


**`stan.stats.clients`**
:   The number of STAN clients

type: integer


**`stan.stats.subscriptions`**
:   The number of STAN streaming subscriptions

type: integer


**`stan.stats.channels`**
:   The number of STAN channels

type: integer


**`stan.stats.messages`**
:   Number of messages across all STAN queues

type: long


**`stan.stats.bytes`**
:   Number of bytes consumed across all STAN queues

type: long



## subscriptions [_subscriptions_2]

Contains stan / nats streaming/serverz endpoint subscription metrics

**`stan.subscriptions.id`**
:   The name of the STAN channel subscription (client_id)

type: keyword


**`stan.subscriptions.channel`**
:   The name of the STAN channel the subscription is associated with

type: keyword


**`stan.subscriptions.queue`**
:   The name of the NATS queue that the STAN channel subscription is associated with, if any

type: keyword


**`stan.subscriptions.last_sent`**
:   Last known sequence number of the subscription that was acked

type: long


**`stan.subscriptions.pending`**
:   Number of pending messages from / to the subscriber

type: long


**`stan.subscriptions.offline`**
:   Is the subscriber marked as offline?

type: boolean


**`stan.subscriptions.stalled`**
:   Is the subscriber known to be stalled?

type: boolean


