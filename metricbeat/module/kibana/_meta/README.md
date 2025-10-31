# Kibana Module

- [About the kibana metricbeat module](https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-module-kibana.html)
- [List of exported fields](https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-kibana.html) stored by this module generated from fields.yml files

## Metricsets

Metricbeat will call the following Kibana API endpoints corresponding to each metricset.

### stats [xpack]

- `/api/stats`
- [api implementation](https://github.com/elastic/kibana/tree/main/src/plugins/usage_collection/server/routes/stats)
- [mb exported fields](https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-kibana.html#_stats_5)

This metricset provides all the data used to drive the Stack Monitoring UI for kibana components. Metricbeat calls the endpoint with `?extended=true` in order to include the elasticsearch cluster uuid as well.

### status

- `/api/status`
- [api implementation](https://github.com/elastic/kibana/blob/main/src/core/server/status/routes/status.ts)
- [mb exported fields](https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-kibana.html#_status_2)

This endpoint provides detailed information about kibana and plugin status. The metricbeat module only indexes overall state and some request/connection metrics.
