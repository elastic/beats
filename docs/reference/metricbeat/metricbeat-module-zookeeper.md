---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-module-zookeeper.html
---

# ZooKeeper module [metricbeat-module-zookeeper]

:::::{admonition} Prefer to use {{agent}} for this use case?
Refer to the [Elastic Integrations documentation](integration-docs://reference/zookeeper/index.md).

::::{dropdown} Learn more
{{agent}} is a single, unified way to add monitoring for logs, metrics, and other types of data to a host. It can also protect hosts from security threats, query data from operating systems, forward data from remote services or hardware, and more. Refer to the documentation for a detailed [comparison of {{beats}} and {{agent}}](docs-content://reference/fleet/index.md).

::::


:::::


The ZooKeeper module fetches statistics from the ZooKeeper service. The default metricsets are `mntr` and `server`.


## Compatibility [_compatibility_52]

The ZooKeeper metricsets were tested with ZooKeeper 3.4.8, 3.6.0 and 3.7.0. They are expected to work with all versions >= 3.4.0. Versions prior to 3.4 do not support the `mntr` command.

Note that from ZooKeeper 3.6.0, `mntr`, `stat`, `ruok`, `conf`, `isro`, `cons` command must be explicitly enabled at ZooKeeper side using the `4lw.commands.whitelist` configuration parameter.


## Dashboard [_dashboard_48]

The Zookeeper module comes with a predefined dashboard:

![metricbeat zookeeper](images/metricbeat-zookeeper.png)


## Example configuration [_example_configuration_69]

The ZooKeeper module supports the standard configuration options that are described in [Modules](/reference/metricbeat/configuration-metricbeat.md). Here is an example configuration:

```yaml
metricbeat.modules:
- module: zookeeper
  enabled: true
  metricsets: ["mntr", "server"]
  period: 10s
  hosts: ["localhost:2181"]
```


## Metricsets [_metricsets_79]

The following metricsets are available:

* [connection](/reference/metricbeat/metricbeat-metricset-zookeeper-connection.md)
* [mntr](/reference/metricbeat/metricbeat-metricset-zookeeper-mntr.md)
* [server](/reference/metricbeat/metricbeat-metricset-zookeeper-server.md)




