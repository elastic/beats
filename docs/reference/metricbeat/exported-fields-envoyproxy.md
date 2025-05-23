---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-envoyproxy.html
---

# Envoyproxy fields [exported-fields-envoyproxy]

envoyproxy module


## envoyproxy [_envoyproxy]


## server [_server_4]

Contains envoy proxy server stats

**`envoyproxy.server.cluster_manager.active_clusters`**
:   Number of currently active (warmed) clusters

type: integer


**`envoyproxy.server.cluster_manager.cluster_added`**
:   Total clusters added (either via static config or CDS)

type: integer


**`envoyproxy.server.cluster_manager.cluster_modified`**
:   Total clusters modified (via CDS)

type: integer


**`envoyproxy.server.cluster_manager.cluster_removed`**
:   Total clusters removed (via CDS)

type: integer


**`envoyproxy.server.cluster_manager.warming_clusters`**
:   Number of currently warming (not active) clusters

type: integer


**`envoyproxy.server.cluster_manager.cluster_updated`**
:   Total cluster updates

type: integer


**`envoyproxy.server.cluster_manager.cluster_updated_via_merge`**
:   Total cluster updates applied as merged updates

type: integer


**`envoyproxy.server.cluster_manager.update_merge_cancelled`**
:   Total merged updates that got cancelled and delivered early

type: integer


**`envoyproxy.server.cluster_manager.update_out_of_merge_window`**
:   Total updates which arrived out of a merge window

type: integer


**`envoyproxy.server.filesystem.flushed_by_timer`**
:   Total number of times internal flush buffers are written to a file due to flush timeout

type: integer


**`envoyproxy.server.filesystem.reopen_failed`**
:   Total number of times a file was failed to be opened

type: integer


**`envoyproxy.server.filesystem.write_buffered`**
:   Total number of times file data is moved to Envoys internal flush buffer

type: integer


**`envoyproxy.server.filesystem.write_completed`**
:   Total number of times a file was written

type: integer


**`envoyproxy.server.filesystem.write_total_buffered`**
:   Current total size of internal flush buffer in bytes

type: integer


**`envoyproxy.server.filesystem.write_failed`**
:   Total number of times an error occurred during a file write operation

type: integer


**`envoyproxy.server.runtime.load_error`**
:   Total number of load attempts that resulted in an error in any layer

type: integer


**`envoyproxy.server.runtime.load_success`**
:   Total number of load attempts that were successful at all layers

type: integer


**`envoyproxy.server.runtime.num_keys`**
:   Number of keys currently loaded

type: integer


**`envoyproxy.server.runtime.override_dir_exists`**
:   Total number of loads that did use an override directory

type: integer


**`envoyproxy.server.runtime.override_dir_not_exists`**
:   Total number of loads that did not use an override directory

type: integer


**`envoyproxy.server.runtime.admin_overrides_active`**
:   1 if any admin overrides are active otherwise 0

type: integer


**`envoyproxy.server.runtime.deprecated_feature_use`**
:   Total number of times deprecated features were used.

type: integer


**`envoyproxy.server.runtime.num_layers`**
:   Number of layers currently active (without loading errors)

type: integer


**`envoyproxy.server.listener_manager.listener_added`**
:   Total listeners added (either via static config or LDS)

type: integer


**`envoyproxy.server.listener_manager.listener_create_failure`**
:   Total failed listener object additions to workers

type: integer


**`envoyproxy.server.listener_manager.listener_create_success`**
:   Total listener objects successfully added to workers

type: integer


**`envoyproxy.server.listener_manager.listener_modified`**
:   Total listeners modified (via LDS)

type: integer


**`envoyproxy.server.listener_manager.listener_removed`**
:   Total listeners removed (via LDS)

type: integer


**`envoyproxy.server.listener_manager.total_listeners_active`**
:   Number of currently active listeners

type: integer


**`envoyproxy.server.listener_manager.total_listeners_draining`**
:   Number of currently draining listeners

type: integer


**`envoyproxy.server.listener_manager.total_listeners_warming`**
:   Number of currently warming listeners

type: integer


**`envoyproxy.server.listener_manager.listener_stopped`**
:   Total listeners stopped

type: integer


**`envoyproxy.server.stats.overflow`**
:   Total number of times Envoy cannot allocate a statistic due to a shortage of shared memory

type: integer


**`envoyproxy.server.server.days_until_first_cert_expiring`**
:   Number of days until the next certificate being managed will expire

type: integer


**`envoyproxy.server.server.live`**
:   1 if the server is not currently draining, 0 otherwise

type: integer


**`envoyproxy.server.server.memory_allocated`**
:   Current amount of allocated memory in bytes

type: integer


**`envoyproxy.server.server.memory_heap_size`**
:   Current reserved heap size in bytes

type: integer


**`envoyproxy.server.server.parent_connections`**
:   Total connections of the old Envoy process on hot restart

type: integer


**`envoyproxy.server.server.total_connections`**
:   Total connections of both new and old Envoy processes

type: integer


**`envoyproxy.server.server.uptime`**
:   Current server uptime in seconds

type: integer


**`envoyproxy.server.server.version`**
:   Integer represented version number based on SCM revision

type: integer


**`envoyproxy.server.server.watchdog_mega_miss`**
:   type: integer


**`envoyproxy.server.server.watchdog_miss`**
:   type: integer


**`envoyproxy.server.server.hot_restart_epoch`**
:   Current hot restart epoch

type: integer


**`envoyproxy.server.server.concurrency`**
:   Number of worker threads

type: integer


**`envoyproxy.server.server.debug_assertion_failures`**
:   type: integer


**`envoyproxy.server.server.dynamic_unknown_fields`**
:   Number of messages in dynamic configuration with unknown fields

type: integer


**`envoyproxy.server.server.state`**
:   Current state of the Server

type: integer


**`envoyproxy.server.server.static_unknown_fields`**
:   Number of messages in static configuration with unknown fields

type: integer


**`envoyproxy.server.server.stats_recent_lookups`**
:   type: integer


**`envoyproxy.server.http2.header_overflow`**
:   Total number of connections reset due to the headers being larger than Envoy::Http::Http2::ConnectionImpl::StreamImpl::MAX_HEADER_SIZE (63k)

type: integer


**`envoyproxy.server.http2.headers_cb_no_stream`**
:   Total number of errors where a header callback is called without an associated stream. This tracks an unexpected occurrence due to an as yet undiagnosed bug

type: integer


**`envoyproxy.server.http2.rx_messaging_error`**
:   Total number of invalid received frames that violated section 8 of the HTTP/2 spec. This will result in a tx_reset

type: integer


**`envoyproxy.server.http2.rx_reset`**
:   Total number of reset stream frames received by Envoy

type: integer


**`envoyproxy.server.http2.too_many_header_frames`**
:   Total number of times an HTTP2 connection is reset due to receiving too many headers frames. Envoy currently supports proxying at most one header frame for 100-Continue one non-100 response code header frame and one frame with trailers

type: integer


**`envoyproxy.server.http2.trailers`**
:   Total number of trailers seen on requests coming from downstream

type: integer


**`envoyproxy.server.http2.tx_reset`**
:   Total number of reset stream frames transmitted by Envoy

type: integer


