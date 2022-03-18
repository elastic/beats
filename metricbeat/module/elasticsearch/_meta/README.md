# Elasticsearch Module

- [About the elasticsearch metricbeat module](https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-module-elasticsearch.html)
- [List of exported fields](https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-elasticsearch.html) stored by this module generated from fields.yml files

## Metricsets

Metricbeat will call the following Elasticsearch API endpoints corresponding to each metricset.  The module stores all or a subset of the response in fields which can be found [here](https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-elasticsearch.html).
### ccr
- `/_ccr/stats`
- [api reference](https://www.elastic.co/guide/en/elasticsearch/reference/current/ccr-get-stats.html)

### cluster_stats
- `/_cluster/stats`
- [api reference](https://www.elastic.co/guide/en/elasticsearch/reference/current/cluster-stats.html)

### enrich
- `/_enrich/_stats`
- [api reference](https://www.elastic.co/guide/en/elasticsearch/reference/current/enrich-stats-api.html)

### index
-  `/_stats/docs,fielddata,indexing,merge,search,segments,store,refresh,query_cache,request_cache?filter_path=indices&expandWildcards=open`
- [api reference](https://www.elastic.co/guide/en/elasticsearch/reference/current/indices-stats.html)

### index_recovery
- `/_recovery?active_only=`
- [api reference](https://www.elastic.co/guide/en/elasticsearch/reference/current/indices-recovery.html)

### index_summary
- `/_stats`
- [api reference](https://www.elastic.co/guide/en/elasticsearch/reference/current/indices-stats.html)

### ml_job
- `/_ml/anomaly_detectors/_all/_stats`
- [api reference](https://www.elastic.co/guide/en/elasticsearch/reference/current/ml-get-job.html)

### node
- `/_nodes/_local`
- [api reference](https://www.elastic.co/guide/en/elasticsearch/reference/current/cluster-nodes-info.html)

### node_stats
- `/_nodes/_local/stats` or `/_nodes/_all/stats` depending on [`scope`](https://www.elastic.co/guide/en/elasticsearch/reference/current/configuring-metricbeat.html#CO490-2) setting
- [api reference](https://www.elastic.co/guide/en/elasticsearch/reference/current/cluster-nodes-info.html)

### pending_tasks
- `/_cluster/pending_tasks`
- [api reference](https://www.elastic.co/guide/en/elasticsearch/reference/current/cluster-pending.html)

### shard
-  `/_cluster/state/version,nodes,master_node,routing_table`
- [api reference](https://www.elastic.co/guide/en/elasticsearch/reference/current/cluster-state.html)