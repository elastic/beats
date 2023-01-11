# Logstash Module

- [About the logstash metricbeat module](https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-module-logstash.html)
- [List of exported fields](https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-logstash.html) stored by this module generated from fields.yml files

## Metricsets

Metricbeat will call the following Logstash API endpoints corresponding to each metricset.

The module stores a subset of the response in fields which can be found [here](https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-logstash.html). The Metricbeat exported fields generally use similar keys to logstash API response, but also have a fair bit of processing related to cluster association and pipeline handling.

See the `eventMapping` function in the code of each metricset for more details about the translation.

### node

- `/_node`
- [api reference](https://www.elastic.co/guide/en/logstash/current/node-info-api.html)

The fields produced by this metricset are mostly found under `logstash.node.state` prefixes.

It also calls a node pipelines API which produces a graph representation of any pipelines running within logstash.

- `/_node/pipelines?graph=true`
- [api reference](https://www.elastic.co/guide/en/logstash/current/node-info-api.html#node-pipeline-info)

This graph structure is used to power the [pipeline viewer](https://github.com/elastic/kibana/tree/main/x-pack/plugins/monitoring/public/components/logstash/pipeline_viewer) in the Stack Monitoring UI.

### node_stats

- `/_node/stats`
- [api reference](https://www.elastic.co/guide/en/logstash/current/node-stats-api.html)

The fields produced by this metricset are mostly found under `logtash.node.stats` prefixes.
