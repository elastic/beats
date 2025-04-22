---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-memcached.html
---

# Memcached fields [exported-fields-memcached]

Memcached module


## memcached [_memcached]


## stats [_stats_7]

stats

**`memcached.stats.pid`**
:   Current process ID of the Memcached task.

type: long


**`memcached.stats.uptime.sec`**
:   Memcached server uptime.

type: long


**`memcached.stats.threads`**
:   Number of threads used by the current Memcached server process.

type: long


**`memcached.stats.connections.current`**
:   Number of open connections to this Memcached server, should be the same value on all servers during normal operation.

type: long


**`memcached.stats.connections.total`**
:   Numer of successful connect attempts to this server since it has been started.

type: long


**`memcached.stats.get.hits`**
:   Number of successful "get" commands (cache hits) since startup, divide them by the "cmd_get" value to get the cache hitrate.

type: long


**`memcached.stats.get.misses`**
:   Number of failed "get" requests because nothing was cached for this key or the cached value was too old.

type: long


**`memcached.stats.cmd.get`**
:   Number of "get" commands received since server startup not counting if they were successful or not.

type: long


**`memcached.stats.cmd.set`**
:   Number of "set" commands serviced since startup.

type: long


**`memcached.stats.read.bytes`**
:   Total number of bytes received from the network by this server.

type: long


**`memcached.stats.written.bytes`**
:   Total number of bytes send to the network by this server.

type: long


**`memcached.stats.items.current`**
:   Number of items currently in this server’s cache.

type: long


**`memcached.stats.items.total`**
:   Number of items stored ever stored on this server. This is no "maximum item count" value but a counted increased by every new item stored in the cache.

type: long


**`memcached.stats.evictions`**
:   Number of objects removed from the cache to free up memory for new items because Memcached reached it’s maximum memory setting (limit_maxbytes).

type: long


**`memcached.stats.bytes.current`**
:   Number of bytes currently used for caching items.

type: long


**`memcached.stats.bytes.limit`**
:   Number of bytes this server is allowed to use for storage.

type: long


$$$exported-fields-meraki$$$

**`meraki.device.serial`**
:   type: keyword


