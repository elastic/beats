---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-tomcat.html
---

# Tomcat fields [exported-fields-tomcat]

Tomcat module


## cache [_cache_5]

Catalina Cache metrics from the WebResourceRoot

**`tomcat.cache.mbean`**
:   Mbean that this event is related to

type: keyword


**`tomcat.cache.hit.total`**
:   The number of requests for resources that were served from the cache

type: long


**`tomcat.cache.size.total.kb`**
:   The current estimate of the cache size in kilobytes

type: long


**`tomcat.cache.size.max.kb`**
:   The maximum permitted size of the cache in kilobytes

type: long


**`tomcat.cache.lookup.total`**
:   The number of requests for resources

type: long


**`tomcat.cache.ttl.ms`**
:   The time-to-live for cache entries in milliseconds

type: long



## memory [_memory_15]

Memory metrics from java.lang JMX

**`tomcat.memory.mbean`**
:   Mbean that this event is related to

type: keyword


**`tomcat.memory.heap.usage.committed`**
:   Committed heap memory usage

type: long


**`tomcat.memory.heap.usage.max`**
:   Max heap memory usage

type: long


**`tomcat.memory.heap.usage.used`**
:   Used heap memory usage

type: long


**`tomcat.memory.heap.usage.init`**
:   Initial heap memory usage

type: long


**`tomcat.memory.other.usage.committed`**
:   Committed non-heap memory usage

type: long


**`tomcat.memory.other.usage.max`**
:   Max non-heap memory usage

type: long


**`tomcat.memory.other.usage.used`**
:   Used non-heap memory usage

type: long


**`tomcat.memory.other.usage.init`**
:   Initial non-heap memory usage

type: long



## requests [_requests_2]

Requests processor metrics from GlobalRequestProcessor JMX

**`tomcat.requests.mbean`**
:   Mbean that this event is related to

type: keyword


**`tomcat.requests.total`**
:   Number of requests processed

type: long


**`tomcat.requests.bytes.received`**
:   Amount of data received, in bytes

type: long


**`tomcat.requests.bytes.sent`**
:   Amount of data sent, in bytes

type: long


**`tomcat.requests.processing.ms`**
:   Total time to process the requests

type: long


**`tomcat.requests.errors.total`**
:   Number of errors

type: long



## threading [_threading]

Threading metrics from the Catalina’s ThreadPool JMX

**`tomcat.threading.busy`**
:   Current busy threads from the ThreadPool

type: long


**`tomcat.threading.max`**
:   Max threads from the ThreadPool

type: long


**`tomcat.threading.current`**
:   Current number of threads, taken from the ThreadPool

type: long


**`tomcat.threading.keep_alive.total`**
:   Total keep alive on the ThreadPool

type: long


**`tomcat.threading.keep_alive.timeout.ms`**
:   Keep alive timeout on the ThreadPool

type: long


**`tomcat.threading.started.total`**
:   Current started threads at JVM level (from java.lang:type=Threading)

type: long


**`tomcat.threading.user.time.ms`**
:   User time in milliseconds (from java.lang:type=Threading)

type: long


**`tomcat.threading.cpu.time.ms`**
:   CPU time in milliseconds (from java.lang:type=Threading)

type: long


**`tomcat.threading.total`**
:   Total threads at the JVM level (from java.lang:type=Threading)

type: long


**`tomcat.threading.peak`**
:   Peak number of threads at JVM level (from java.lang:type=Threading)

type: long


