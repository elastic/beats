::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


The `agent` metricset fetches information from a Hashicorp Consul agent in *Client* mode. It fetches information about the health of the autopilot, runtime metrics, and raft data.

* **agent.autopilot.healthy**: Tracks the overall health of the local server cluster. If all servers are considered healthy by Autopilot, this will be set to 1. If any are unhealthy, this will be 0.
* **agent.raft.apply**: This metric gives the number of logs committed since the last interval.
* **agent.raft.commit_time.ms**: This tracks the average time in milliseconds it takes to commit a new entry to the transaction log of the leader
* **agent.runtime.alloc.bytes**: This measures the number of bytes allocated by the Consul process.
* **agent.runtime.garbage_collector.pause.current.ns**: Garbage collector pause time in nanoseconds
* **agent.runtime.garbage_collector.pause.total.ns**: Number of nanoseconds consumed by stop-the-world garbage collection pauses since Consul started.
* **agent.runtime.garbage_collector.runs**: Garbage collector total executions
* **agent.runtime.goroutines**: Number of running goroutines and is a general load pressure indicator. This may burst from time to time but should return to a steady state value.
* **agent.runtime.heap_objects**: This measures the number of objects allocated on the heap and is a general memory pressure indicator. This may burst from time to time but should return to a steady state value.
* **agent.runtime.malloc_count**: Heap objects allocated
* **agent.runtime.sys.bytes**: Total number of bytes of memory obtained from the OS.
