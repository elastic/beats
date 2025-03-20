---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-consul.html
---

# Consul fields [exported-fields-consul]

Consul module


## agent [_agent]

Agent Metricset fetches metrics information from a Consul instance running as Agent

**`consul.agent.autopilot.healthy`**
:   Overall health of the local server cluster

type: boolean



## runtime [_runtime]

Runtime related metrics

**`consul.agent.runtime.sys.bytes`**
:   Number of bytes of memory obtained from the OS.

type: long


**`consul.agent.runtime.malloc_count`**
:   Heap objects allocated

type: long


**`consul.agent.runtime.heap_objects`**
:   Objects allocated on the heap and is a general memory pressure indicator. This may burst from time to time but should return to a steady state value.

type: long


**`consul.agent.runtime.goroutines`**
:   Running goroutines and is a general load pressure indicator. This may burst from time to time but should return to a steady state value.

type: long


**`consul.agent.runtime.alloc.bytes`**
:   Bytes allocated by the Consul process.

type: long



## garbage_collector [_garbage_collector]

Garbage collector metrics

**`consul.agent.runtime.garbage_collector.runs`**
:   Garbage collector total executions

type: long



## pause [_pause]

Time that the garbage collector has paused the app

**`consul.agent.runtime.garbage_collector.pause.current.ns`**
:   Garbage collector pause time in nanoseconds

type: long


**`consul.agent.runtime.garbage_collector.pause.total.ns`**
:   Nanoseconds consumed by stop-the-world garbage collection pauses since Consul started.

type: long


