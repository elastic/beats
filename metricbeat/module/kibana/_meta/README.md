# Kibana Module

- [About the kibana metricbeat module](https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-module-kibana.html)
- [List of exported fields](https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-kibana.html) stored by this module generated from fields.yml files

## Metricsets

Metricbeat will call the following Kibana API endpoints corresponding to each metricset.

### settings

- `/api/settings`
- [mb exported fields](https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-kibana.html#_settings_2)

This endpoint provides some basic information about the Kibana instance and how it's configured (uuids, local settings, status).

The endpoint was removed from kibana in 8.0.0-beta1 ([changelog](https://www.elastic.co/guide/en/kibana/master/release-notes-8.0.0-beta1.html#rest-api-breaking-changes-8.0.0-beta1)) but the metricset can still be used on older kibana versions.

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
