---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-iis.html
---

# IIS fields [exported-fields-iis]

iis module


## iis [_iis]


## application_pool [_application_pool_2]

Application pool process stats.

**`iis.application_pool.name`**
:   application pool name

type: keyword



## process [_process_4]

Worker process overview.

**`iis.application_pool.process.handle_count`**
:   The number of handles.

type: long


**`iis.application_pool.process.io_read_operations_per_sec`**
:   IO read operations per sec.

type: float


**`iis.application_pool.process.io_write_operations_per_sec`**
:   IO write operations per sec.

type: float


**`iis.application_pool.process.virtual_bytes`**
:   Memory virtual bytes.

type: float


**`iis.application_pool.process.cpu_usage_perc`**
:   The CPU usage percentage.

type: float


**`iis.application_pool.process.thread_count`**
:   The number of threats.

type: long


**`iis.application_pool.process.working_set`**
:   Memory working set.

type: float


**`iis.application_pool.process.private_bytes`**
:   Memory private bytes.

type: float


**`iis.application_pool.process.page_faults_per_sec`**
:   Memory page faults.

type: float



## net_clr [_net_clr]

Common Language Runtime overview.

**`iis.application_pool.net_clr.finallys_per_sec`**
:   The number of finallys per sec.

type: float


**`iis.application_pool.net_clr.throw_to_catch_depth_per_sec`**
:   Throw to catch depth count per sec.

type: float


**`iis.application_pool.net_clr.total_exceptions_thrown`**
:   Total number of exceptions thrown.

type: long


**`iis.application_pool.net_clr.filters_per_sec`**
:   Number of filters per sec.

type: float


**`iis.application_pool.net_clr.exceptions_thrown_per_sec`**
:   Number of Exceptions Thrown / sec.

type: float



## memory [_memory_7]

Memory overview.

**`iis.application_pool.net_clr.memory.bytes_in_all_heaps`**
:   Number of bytes in all heaps.

type: float


**`iis.application_pool.net_clr.memory.gen_0_collections`**
:   Number of Gen 0 Collections.

type: float


**`iis.application_pool.net_clr.memory.gen_1_collections`**
:   Number of Gen 1 Collections.

type: float


**`iis.application_pool.net_clr.memory.gen_2_collections`**
:   Number of Gen 2 Collections.

type: float


**`iis.application_pool.net_clr.memory.total_committed_bytes`**
:   Number of total committed bytes.

type: float


**`iis.application_pool.net_clr.memory.allocated_bytes_per_sec`**
:   Allocated Bytes/sec.

type: float


**`iis.application_pool.net_clr.memory.gen_0_heap_size`**
:   Gen 0 heap size.

type: float


**`iis.application_pool.net_clr.memory.gen_1_heap_size`**
:   Gen 1 heap size.

type: float


**`iis.application_pool.net_clr.memory.gen_2_heap_size`**
:   Gen 2 heap size.

type: float


**`iis.application_pool.net_clr.memory.large_object_heap_size`**
:   Large Object Heap size.

type: float


**`iis.application_pool.net_clr.memory.time_in_gc_perc`**
:   % Time in GC.

type: float



## locks_and_threads [_locks_and_threads]

LocksAndThreads overview.

**`iis.application_pool.net_clr.locks_and_threads.contention_rate_per_sec`**
:   Contention Rate / sec.

type: float


**`iis.application_pool.net_clr.locks_and_threads.current_queue_length`**
:   Current Queue Length.

type: float



## webserver [_webserver_2]

Webserver related metrics.


## process [_process_5]

The process related stats.

**`iis.webserver.process.cpu_usage_perc`**
:   The CPU usage percentage.

type: float


**`iis.webserver.process.handle_count`**
:   The number of handles.

type: float


**`iis.webserver.process.virtual_bytes`**
:   Memory virtual bytes.

type: float


**`iis.webserver.process.thread_count`**
:   The number of threads.

type: long


**`iis.webserver.process.working_set`**
:   Memory working set.

type: float


**`iis.webserver.process.private_bytes`**
:   Memory private bytes.

type: float


**`iis.webserver.process.worker_process_count`**
:   Number of worker processes running.

type: float


**`iis.webserver.process.page_faults_per_sec`**
:   Memory page faults.

type: float


**`iis.webserver.process.io_read_operations_per_sec`**
:   IO read operations per sec.

type: float


**`iis.webserver.process.io_write_operations_per_sec`**
:   IO write operations per sec.

type: float



## asp_net [_asp_net]

Common Language Runtime overview.

**`iis.webserver.asp_net.application_restarts`**
:   Number of applications restarts.

type: float


**`iis.webserver.asp_net.request_wait_time`**
:   Request wait time.

type: long



## asp_net_application [_asp_net_application]

ASP.NET application overview.

**`iis.webserver.asp_net_application.errors_total_per_sec`**
:   Total number of errors per sec.

type: float


**`iis.webserver.asp_net_application.pipeline_instance_count`**
:   The pipeline instance count.

type: float


**`iis.webserver.asp_net_application.requests_per_sec`**
:   Number of requests per sec.

type: float


**`iis.webserver.asp_net_application.requests_executing`**
:   Number of requests executing.

type: float


**`iis.webserver.asp_net_application.requests_in_application_queue`**
:   Number of requests in the application queue.

type: float



## cache [_cache]

The cache overview.

**`iis.webserver.cache.current_file_cache_memory_usage`**
:   The current file cache memory usage size.

type: float


**`iis.webserver.cache.current_files_cached`**
:   The number of current files cached.

type: float


**`iis.webserver.cache.current_uris_cached`**
:   The number of current uris cached.

type: float


**`iis.webserver.cache.file_cache_hits`**
:   The number of file cache hits.

type: float


**`iis.webserver.cache.file_cache_misses`**
:   The number of file cache misses.

type: float


**`iis.webserver.cache.maximum_file_cache_memory_usage`**
:   The max file cache size.

type: float


**`iis.webserver.cache.output_cache_current_items`**
:   The number of output cache current items.

type: float


**`iis.webserver.cache.output_cache_current_memory_usage`**
:   The output cache memory usage size.

type: float


**`iis.webserver.cache.output_cache_total_hits`**
:   The output cache total hits count.

type: float


**`iis.webserver.cache.output_cache_total_misses`**
:   The output cache total misses count.

type: float


**`iis.webserver.cache.total_files_cached`**
:   the total number of files cached.

type: float


**`iis.webserver.cache.total_uris_cached`**
:   The total number of URIs cached.

type: float


**`iis.webserver.cache.uri_cache_hits`**
:   The number of URIs cached hits.

type: float


**`iis.webserver.cache.uri_cache_misses`**
:   The number of URIs cache misses.

type: float



## network [_network_6]

The network related stats.

**`iis.webserver.network.anonymous_users_per_sec`**
:   The number of anonymous users per sec.

type: float


**`iis.webserver.network.bytes_received_per_sec`**
:   The size of bytes received per sec.

type: float


**`iis.webserver.network.bytes_sent_per_sec`**
:   The size of bytes sent per sec.

type: float


**`iis.webserver.network.current_anonymous_users`**
:   The number of current anonymous users.

type: float


**`iis.webserver.network.current_connections`**
:   The number of current connections.

type: float


**`iis.webserver.network.current_non_anonymous_users`**
:   The number of current non anonymous users.

type: float


**`iis.webserver.network.delete_requests_per_sec`**
:   Number of DELETE requests per sec.

type: float


**`iis.webserver.network.get_requests_per_sec`**
:   Number of GET requests per sec.

type: float


**`iis.webserver.network.maximum_connections`**
:   Number of maximum connections.

type: float


**`iis.webserver.network.post_requests_per_sec`**
:   Number of POST requests per sec.

type: float


**`iis.webserver.network.service_uptime`**
:   Service uptime.

type: float


**`iis.webserver.network.total_anonymous_users`**
:   Total number of anonymous users.

type: float


**`iis.webserver.network.total_bytes_received`**
:   Total size of bytes received.

type: float


**`iis.webserver.network.total_bytes_sent`**
:   Total size of bytes sent.

type: float


**`iis.webserver.network.total_connection_attempts`**
:   The total number of connection attempts.

type: float


**`iis.webserver.network.total_delete_requests`**
:   The total number of DELETE requests.

type: float


**`iis.webserver.network.total_get_requests`**
:   The total number of GET requests.

type: float


**`iis.webserver.network.total_non_anonymous_users`**
:   The total number of non anonymous users.

type: float


**`iis.webserver.network.total_post_requests`**
:   The total number of POST requests.

type: float



## website [_website_2]

Website related metrics.

**`iis.website.name`**
:   website name

type: keyword



## network [_network_7]

The network overview.

**`iis.website.network.bytes_received_per_sec`**
:   The bytes received per sec size.

type: float


**`iis.website.network.bytes_sent_per_sec`**
:   The bytes sent per sec size.

type: float


**`iis.website.network.current_connections`**
:   The number of current connections.

type: float


**`iis.website.network.delete_requests_per_sec`**
:   The number of DELETE requests per sec.

type: float


**`iis.website.network.get_requests_per_sec`**
:   The number of GET requests per sec.

type: float


**`iis.website.network.maximum_connections`**
:   The number of maximum connections.

type: float


**`iis.website.network.post_requests_per_sec`**
:   The number of POST requests per sec.

type: float


**`iis.website.network.put_requests_per_sec`**
:   The number of PUT requests per sec.

type: float


**`iis.website.network.service_uptime`**
:   The service uptime.

type: float


**`iis.website.network.total_bytes_received`**
:   The total number of bytes received.

type: float


**`iis.website.network.total_bytes_sent`**
:   The  total number of bytes sent.

type: float


**`iis.website.network.total_connection_attempts`**
:   The total number of connection attempts.

type: float


**`iis.website.network.total_delete_requests`**
:   The total number of DELETE requests.

type: float


**`iis.website.network.total_get_requests`**
:   The total number of GET requests.

type: float


**`iis.website.network.total_post_requests`**
:   The total number of POST requests.

type: float


**`iis.website.network.total_put_requests`**
:   The total number of PUT requests.

type: float


