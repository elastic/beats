---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-module-syncgateway.html
---

# SyncGateway module [metricbeat-module-syncgateway]

::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


Sync Gateway is the synchronization server in a Couchbase for Mobile and Edge deployment. This metricset allows to monitor a Sync Gateway instance by using its REST API.

Sync Gateway access `[host]:[port]/_expvar` on Sync Gateway nodes to fetch metrics data, ensure that the URL is accessible from the host where Metricbeat is running.


## Example configuration [_example_configuration_62]

The SyncGateway module supports the standard configuration options that are described in [Modules](/reference/metricbeat/configuration-metricbeat.md). Here is an example configuration:

```yaml
metricbeat.modules:
- module: syncgateway
  metricsets:
    - db
#    - memory
#    - replication
#    - resources
  period: 10s

  # SyncGateway hosts
  hosts: ["127.0.0.1:4985"]
```


## Metricsets [_metricsets_72]

The following metricsets are available:

* [db](/reference/metricbeat/metricbeat-metricset-syncgateway-db.md)
* [memory](/reference/metricbeat/metricbeat-metricset-syncgateway-memory.md)
* [replication](/reference/metricbeat/metricbeat-metricset-syncgateway-replication.md)
* [resources](/reference/metricbeat/metricbeat-metricset-syncgateway-resources.md)





