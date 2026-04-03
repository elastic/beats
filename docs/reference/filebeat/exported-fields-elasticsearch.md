---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-elasticsearch.html
applies_to:
  stack: ga
  serverless: ga
---

% This file is generated! See dev-tools/mage/generate_fields_docs.go

# Elasticsearch fields [exported-fields-elasticsearch]

elasticsearch Module

## elasticsearch [_elasticsearch]



**`elasticsearch.component`**
:   Elasticsearch component from where the log event originated

    type: keyword

    example: o.e.c.m.MetaDataCreateIndexService


**`elasticsearch.cluster.uuid`**
:   UUID of the cluster

    type: keyword

    example: GmvrbHlNTiSVYiPf8kxg9g


**`elasticsearch.cluster.name`**
:   Name of the cluster

    type: keyword

    example: docker-cluster


**`elasticsearch.node.id`**
:   ID of the node

    type: keyword

    example: DSiWcTyeThWtUXLB9J0BMw


**`elasticsearch.node.name`**
:   Name of the node

    type: keyword

    example: vWNJsZ3


**`elasticsearch.task.id`**
:   Task id for the query task (query log)

    type: keyword


**`elasticsearch.parent.task.id`**
:   Parent task id when the task has a parent (query log)

    type: keyword


**`elasticsearch.parent.node.id`**
:   Parent node id, present together with parent task id (query log)

    type: keyword


**`elasticsearch.index.name`**
:   Index name

    type: keyword

    example: filebeat-test-input


**`elasticsearch.index.id`**
:   Index id

    type: keyword

    example: aOGgDwbURfCV57AScqbCgw


**`elasticsearch.shard.id`**
:   Id of the shard

    type: keyword

    example: 0


**`elasticsearch.elastic_product_origin`**
:   Used by Elastic stack to identify which component of the stack sent the request

    type: keyword

    example: kibana


**`elasticsearch.http.request.x_opaque_id`**
:   Used by Elasticsearch to throttle and deduplicate deprecation warnings

    type: keyword

    example: v7app


**`elasticsearch.event.category`**
:   Category of the deprecation event

    type: keyword

    example: compatible_api


**`elasticsearch.audit.layer`**
:   The layer from which this event originated: rest, transport or ip_filter

    type: keyword

    example: rest


**`elasticsearch.audit.event_type`**
:   The type of event that occurred: anonymous_access_denied, authentication_failed, access_denied, access_granted, connection_granted, connection_denied, tampered_request, run_as_granted, run_as_denied

    type: keyword

    example: access_granted


**`elasticsearch.audit.origin.type`**
:   Where the request originated: rest (request originated from a REST API request), transport (request was received on the transport channel), local_node (the local node issued the request)

    type: keyword

    example: local_node


**`elasticsearch.audit.realm`**
:   The authentication realm the authentication was validated against

    type: keyword


**`elasticsearch.audit.user.realm`**
:   The user's authentication realm, if authenticated

    type: keyword


**`elasticsearch.audit.user.roles`**
:   Roles to which the principal belongs

    type: keyword

    example: ['kibana_admin', 'beats_admin']


**`elasticsearch.audit.user.run_as.name`**
:   type: keyword


**`elasticsearch.audit.user.run_as.realm`**
:   type: keyword


**`elasticsearch.audit.component`**
:   type: keyword


**`elasticsearch.audit.action`**
:   The name of the action that was executed

    type: keyword

    example: cluster:monitor/main


**`elasticsearch.audit.url.params`**
:   REST URI parameters

    example: {username=jacknich2}


**`elasticsearch.audit.indices`**
:   Indices accessed by action

    type: keyword

    example: ['foo-2019.01.04', 'foo-2019.01.03', 'foo-2019.01.06']


**`elasticsearch.audit.request.id`**
:   Unique ID of request

    type: keyword

    example: WzL_kb6VSvOhAq0twPvHOQ


**`elasticsearch.audit.request.name`**
:   The type of request that was executed

    type: keyword

    example: ClearScrollRequest


**`elasticsearch.audit.request_body`**
:   type: alias

    alias to: http.request.body.content


**`elasticsearch.audit.origin_address`**
:   type: alias

    alias to: source.ip


**`elasticsearch.audit.uri`**
:   type: alias

    alias to: url.original


**`elasticsearch.audit.principal`**
:   type: alias

    alias to: user.name


**`elasticsearch.audit.message`**
:   type: text


**`elasticsearch.audit.invalidate.apikeys.owned_by_authenticated_user`**
:   type: boolean


**`elasticsearch.audit.authentication.type`**
:   type: keyword


**`elasticsearch.audit.opaque_id`**
:   type: text


## deprecation [_deprecation]



## gc [_gc]

GC fileset fields.

## phase [_phase]

Fields specific to GC phase.

**`elasticsearch.gc.phase.name`**
:   Name of the GC collection phase.

    type: keyword


**`elasticsearch.gc.phase.duration_sec`**
:   Collection phase duration according to the Java virtual machine.

    type: float


**`elasticsearch.gc.phase.scrub_symbol_table_time_sec`**
:   Pause time in seconds cleaning up symbol tables.

    type: float


**`elasticsearch.gc.phase.scrub_string_table_time_sec`**
:   Pause time in seconds cleaning up string tables.

    type: float


**`elasticsearch.gc.phase.weak_refs_processing_time_sec`**
:   Time spent processing weak references in seconds.

    type: float


**`elasticsearch.gc.phase.parallel_rescan_time_sec`**
:   Time spent in seconds marking live objects while application is stopped.

    type: float


**`elasticsearch.gc.phase.class_unload_time_sec`**
:   Time spent unloading unused classes in seconds.

    type: float


## cpu_time [_cpu_time]

Process CPU time spent performing collections.

**`elasticsearch.gc.phase.cpu_time.user_sec`**
:   CPU time spent outside the kernel.

    type: float


**`elasticsearch.gc.phase.cpu_time.sys_sec`**
:   CPU time spent inside the kernel.

    type: float


**`elasticsearch.gc.phase.cpu_time.real_sec`**
:   Total elapsed CPU time spent to complete the collection from start to finish.

    type: float


**`elasticsearch.gc.jvm_runtime_sec`**
:   The time from JVM start up in seconds, as a floating point number.

    type: float


**`elasticsearch.gc.threads_total_stop_time_sec`**
:   Garbage collection threads total stop time seconds.

    type: float


**`elasticsearch.gc.stopping_threads_time_sec`**
:   Time took to stop threads seconds.

    type: float


**`elasticsearch.gc.tags`**
:   GC logging tags.

    type: keyword


## heap [_heap]

Heap allocation and total size.

**`elasticsearch.gc.heap.size_kb`**
:   Total heap size in kilobytes.

    type: integer


**`elasticsearch.gc.heap.used_kb`**
:   Used heap in kilobytes.

    type: integer


## old_gen [_old_gen]

Old generation occupancy and total size.

**`elasticsearch.gc.old_gen.size_kb`**
:   Total size of old generation in kilobytes.

    type: integer


**`elasticsearch.gc.old_gen.used_kb`**
:   Old generation occupancy in kilobytes.

    type: integer


## young_gen [_young_gen]

Young generation occupancy and total size.

**`elasticsearch.gc.young_gen.size_kb`**
:   Total size of young generation in kilobytes.

    type: integer


**`elasticsearch.gc.young_gen.used_kb`**
:   Young generation occupancy in kilobytes.

    type: integer


## querylog [_querylog]

Query log events from Elasticsearch

**`elasticsearch.querylog.indices`**
:   Indices involved (DSL, ES|QL when response present, EQL). Not emitted by SqlLogProducer.

    type: keyword

    example: query_log_test_index


**`elasticsearch.querylog.query`**
:   Query text (DSL JSON, ES|QL, SQL, or EQL)

    type: keyword

    example: {"size":10,"query":{"match_all":{"boost":1.0}}}


**`elasticsearch.querylog.result_count`**
:   Number of rows or hits returned (clamped to int where applicable)

    type: long

    example: 3


**`elasticsearch.querylog.took`**
:   Duration in nanoseconds

    type: long

    example: 1000000


**`elasticsearch.querylog.took_millis`**
:   Duration in milliseconds

    type: long

    example: 1


**`elasticsearch.querylog.type`**
:   Query kind: dsl, esql, sql, or eql

    type: keyword

    example: dsl


**`elasticsearch.querylog.timed_out`**
:   True when the request timed out (e.g. DSL SearchResponse, EQL timeout)

    type: boolean


**`elasticsearch.querylog.shards.successful`**
:   Successful shard count when shardInfo() is present

    type: long

    example: 1


**`elasticsearch.querylog.shards.skipped`**
:   Skipped shards (only emitted if count > 0)

    type: long


**`elasticsearch.querylog.shards.failed`**
:   Failed shards (only emitted if count > 0)

    type: long


**`elasticsearch.querylog.dsl.total_count`**
:   Total hits from TotalHits (DSL only, SearchLogProducer)

    type: long

    example: 3


**`elasticsearch.querylog.dsl.total_count_partial`**
:   True when total hit relation is not EQUAL_TO

    type: boolean


**`elasticsearch.querylog.has_aggregations`**
:   True if the search response has aggregations (DSL)

    type: boolean


**`elasticsearch.querylog.is_system`**
:   True if all resolved indices match the system-index predicate (QueryLogger); omitted when SQL has no indices

    type: boolean


**`elasticsearch.querylog.is_remote`**
:   True when SearchRequest#getLocalClusterAlias() is set (DSL remote-cluster request)

    type: boolean


**`elasticsearch.querylog.clusters.remotes`**
:   Remote cluster aliases when remotes exist (DSL/ES|QL/EQL); JSON array of keyword strings, not a delimited string

    type: keyword


**`elasticsearch.querylog.clusters.remote_count`**
:   Number of remote clusters or distinct remote aliases

    type: long


**`elasticsearch.querylog.clusters.skipped`**
:   Number of skipped remote clusters when present in cluster statistics

    type: long


**`elasticsearch.querylog.clusters.total`**
:   Total cluster entries in the map (DSL and ES|QL only; not EQL). Additional keys use lowercase status strings (failed, partial, running, skipped, successful) with long counts.

    type: long


**`elasticsearch.querylog.esql.profile.analysis.took`**
:   ES|QL profile phase duration in nanoseconds (phase name from TimeSpanMarker)

    type: long


**`elasticsearch.querylog.esql.profile.planning.took`**
:   ES|QL profile phase duration in nanoseconds

    type: long


**`elasticsearch.querylog.esql.profile.parsing.took`**
:   ES|QL profile phase duration in nanoseconds

    type: long


**`elasticsearch.querylog.esql.profile.preanalysis.took`**
:   ES|QL profile phase duration in nanoseconds

    type: long


**`elasticsearch.querylog.esql.profile.query.took`**
:   ES|QL profile phase duration in nanoseconds

    type: long


**`elasticsearch.querylog.esql.profile.dependency_resolution.took`**
:   ES|QL profile phase duration in nanoseconds

    type: long


## server [_server]

Server log file

**`elasticsearch.server.stacktrace`**
:   Field is not indexed.


## gc [_gc]

GC log

## young [_young]

Young GC

**`elasticsearch.server.gc.young.one`**
:       type: long

    example: 


**`elasticsearch.server.gc.young.two`**
:       type: long

    example: 


**`elasticsearch.server.gc.overhead_seq`**
:   Sequence number

    type: long

    example: 3449992


**`elasticsearch.server.gc.collection_duration.ms`**
:   Time spent in GC, in milliseconds

    type: float

    example: 1600


**`elasticsearch.server.gc.observation_duration.ms`**
:   Total time over which collection was observed, in milliseconds

    type: float

    example: 1800


## slowlog [_slowlog]

Slowlog events from Elasticsearch

**`elasticsearch.slowlog.logger`**
:   Logger name

    type: keyword

    example: index.search.slowlog.fetch


**`elasticsearch.slowlog.took`**
:   Time it took to execute the query

    type: keyword

    example: 300ms


**`elasticsearch.slowlog.types`**
:   Types

    type: keyword

    example: 


**`elasticsearch.slowlog.stats`**
:   Stats groups

    type: keyword

    example: group1


**`elasticsearch.slowlog.search_type`**
:   Search type

    type: keyword

    example: QUERY_THEN_FETCH


**`elasticsearch.slowlog.source_query`**
:   Slow query

    type: keyword

    example: {"query":{"match_all":{"boost":1.0}}}


**`elasticsearch.slowlog.extra_source`**
:   Extra source information

    type: keyword

    example: 


**`elasticsearch.slowlog.total_hits`**
:   Total hits

    type: keyword

    example: 42


**`elasticsearch.slowlog.total_shards`**
:   Total queried shards

    type: keyword

    example: 22


**`elasticsearch.slowlog.routing`**
:   Routing

    type: keyword

    example: s01HZ2QBk9jw4gtgaFtn


**`elasticsearch.slowlog.id`**
:   Id

    type: keyword

    example: 


**`elasticsearch.slowlog.type`**
:   Type

    type: keyword

    example: doc


**`elasticsearch.slowlog.source`**
:   Source of document that was indexed

    type: keyword


**`elasticsearch.slowlog.user.realm`**
:   The authentication realm the user was authenticated against

    type: keyword

    example: default_file


**`elasticsearch.slowlog.user.effective.realm`**
:   The authentication realm the effective user was authenticated against

    type: keyword

    example: default_file


**`elasticsearch.slowlog.auth.type`**
:   The authentication type used to authenticate the user. One of TOKEN | REALM | API_KEY

    type: keyword

    example: REALM


**`elasticsearch.slowlog.apikey.id`**
:   The id of the API key used

    type: keyword

    example: WzL_kb6VSvOhAq0twPvHOQ


**`elasticsearch.slowlog.apikey.name`**
:   The name of the API key used

    type: keyword

    example: my-api-key


