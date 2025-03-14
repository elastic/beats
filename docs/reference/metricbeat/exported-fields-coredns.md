---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-coredns.html
---

# Coredns fields [exported-fields-coredns]

coredns Module


## coredns [_coredns]

`coredns` contains statistics that were read from coreDNS


## stats [_stats_3]

Contains statistics related to the coreDNS service

**`coredns.stats.panic.count`**
:   Total number of panics

type: long


**`coredns.stats.dns.request.count`**
:   Total query count

type: long


**`coredns.stats.dns.request.duration.ns.bucket.*`**
:   Request duration histogram buckets in nanoseconds

type: object


**`coredns.stats.dns.request.duration.ns.sum`**
:   Requests duration, sum of durations in nanoseconds

type: long

format: duration


**`coredns.stats.dns.request.duration.ns.count`**
:   Requests duration, number of requests

type: long


**`coredns.stats.dns.request.size.bytes.bucket.*`**
:   Request Size histogram buckets

type: object


**`coredns.stats.dns.request.size.bytes.sum`**
:   Request Size histogram sum

type: long


**`coredns.stats.dns.request.size.bytes.count`**
:   Request Size histogram count

type: long


**`coredns.stats.dns.request.do.count`**
:   Number of queries that have the DO bit set

type: long


**`coredns.stats.dns.request.type.count`**
:   Counter of queries per zone and type

type: long


**`coredns.stats.type`**
:   Holds the query type of the request

type: keyword


**`coredns.stats.dns.response.rcode.count`**
:   Counter of responses per zone and rcode

type: long


**`coredns.stats.rcode`**
:   Holds the rcode of the response

type: keyword


**`coredns.stats.family`**
:   The address family of the transport (1 = IP (IP version 4), 2 = IP6 (IP version 6))

type: keyword


**`coredns.stats.dns.response.size.bytes.bucket.*`**
:   Response Size histogram buckets

type: object


**`coredns.stats.dns.response.size.bytes.sum`**
:   Response Size histogram sum

type: long


**`coredns.stats.dns.response.size.bytes.count`**
:   Response Size histogram count

type: long


**`coredns.stats.server`**
:   The server responsible for the request

type: keyword


**`coredns.stats.zone`**
:   The zonename used for the request/response

type: keyword


**`coredns.stats.proto`**
:   The transport of the response ("udp" or "tcp")

type: keyword


**`coredns.stats.dns.cache.hits.count`**
:   Cache hits count for the cache plugin

type: long


**`coredns.stats.dns.cache.misses.count`**
:   Cache misses count for the cache plugin

type: long


