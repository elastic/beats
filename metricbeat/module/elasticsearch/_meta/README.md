# Elasticsearch Module

- [About the elasticsearch metricbeat module](https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-module-elasticsearch.html)
- [List of exported fields](https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-elasticsearch.html) stored by this module generated from fields.yml files

## Metricsets

Metricbeat will call the following Elasticsearch API endpoints corresponding to each metricset.  The module stores a subset of the response in fields which can be found [here](https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-elasticsearch.html).  The Metricbeat exported fields generally follow the same paths as the elasticsearch API response, but may differ somewhat according to the [schema in each metricset](https://github.com/elastic/beats/blob/main/metricbeat/module/elasticsearch/node/data.go#L36).  They are [namespaced with `{module}.{metricset}`](https://github.com/elastic/beats/blob/main/metricbeat/module/elasticsearch/cluster_stats/data.go#L39).
### ccr
- `/_ccr/stats`
- [api reference](https://www.elastic.co/guide/en/elasticsearch/reference/current/ccr-get-stats.html)
- [mb exported fields](https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-elasticsearch.html#_ccr)

### cluster_stats
- `/_cluster/stats`
- [api reference](https://www.elastic.co/guide/en/elasticsearch/reference/current/cluster-stats.html)
- [mb exported fields](https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-elasticsearch.html#_cluster_stats)

### enrich
- `/_enrich/_stats`
- [api reference](https://www.elastic.co/guide/en/elasticsearch/reference/current/enrich-stats-api.html)
- [mb exported fields](https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-elasticsearch.html#_enrich)

### index
-  `/_stats/docs,fielddata,indexing,merge,search,segments,store,refresh,query_cache,request_cache?filter_path=indices&expandWildcards=open`
- [api reference](https://www.elastic.co/guide/en/elasticsearch/reference/current/indices-stats.html)
- [mb exported fields](https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-elasticsearch.html#_index_3)

### index_recovery
- `/_recovery?active_only=true`
- `active_only` value from [`index_recovery.active_only` setting](https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-metricset-elasticsearch-index_recovery.html)
- [api reference](https://www.elastic.co/guide/en/elasticsearch/reference/current/indices-recovery.html)
- [mb exported fields](https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-elasticsearch.html#_index_recovery)

### index_summary
- `/_stats`
- [api reference](https://www.elastic.co/guide/en/elasticsearch/reference/current/indices-stats.html)
- [mb exported fields](https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-elasticsearch.html#_index_summary)

### ml_job
- `/_ml/anomaly_detectors/_all/_stats`
- [api reference](https://www.elastic.co/guide/en/elasticsearch/reference/current/ml-get-job.html)
- [mb exported fields](https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-elasticsearch.html#_ml_job)

### node
- `/_nodes/_local`
- [api reference](https://www.elastic.co/guide/en/elasticsearch/reference/current/cluster-nodes-info.html)
- [mb exported fields](https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-elasticsearch.html#_node_2)

### node_stats
- `/_nodes/{_local|_all}/stats`
- `_local` | `_all` from [`scope`](https://www.elastic.co/guide/en/elasticsearch/reference/current/configuring-metricbeat.html#CO490-2) setting
- [api reference](https://www.elastic.co/guide/en/elasticsearch/reference/current/cluster-nodes-info.html)
- [mb exported fields](https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-elasticsearch.html#_node_stats)
### pending_tasks
- `/_cluster/pending_tasks`
- [api reference](https://www.elastic.co/guide/en/elasticsearch/reference/current/cluster-pending.html)
- [mb exported fields](https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-elasticsearch.html#_cluster_pending_task)
### shard
-  `/_cluster/state/version,nodes,master_node,routing_table`
- [api reference](https://www.elastic.co/guide/en/elasticsearch/reference/current/cluster-state.html)
- [mb exported fields](https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-elasticsearch.html#_shard)