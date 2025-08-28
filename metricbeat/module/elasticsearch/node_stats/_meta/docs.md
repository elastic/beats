The `node_stats` metricset interrogates the [Cluster API endpoint](https://www.elastic.co/docs/api/doc/elasticsearch/operation/operation-nodes-stats) of Elasticsearch to get the cluster nodes statistics. The data received is only for the local node so this Metricbeat has to be run on each Elasticsearch node.

::::{note}
The indices stats are node-specific. That means for example the total number of docs reported by all nodes together is not the total number of documents in all indices as there can also be replicas.
::::

