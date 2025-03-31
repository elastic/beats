---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-module-activemq.html
---

# ActiveMQ module [metricbeat-module-activemq]

:::::{admonition} Prefer to use {{agent}} for this use case?
Refer to the [Elastic Integrations documentation](integration-docs://reference/activemq/index.md).

::::{dropdown} Learn more
{{agent}} is a single, unified way to add monitoring for logs, metrics, and other types of data to a host. It can also protect hosts from security threats, query data from operating systems, forward data from remote services or hardware, and more. Refer to the documentation for a detailed [comparison of {{beats}} and {{agent}}](docs-content://reference/fleet/index.md).

::::


:::::


This module periodically fetches JMX metrics from Apache ActiveMQ.


## Compatibility [_compatibility_4]

The module has been tested with ActiveMQ 5.13.0 and 5.15.9. Other versions are expected to work.


## Usage [_usage]

The ActiveMQ module requires [Jolokia](/reference/metricbeat/metricbeat-module-jolokia.md) to fetch JMX metrics. Refer to the link for instructions about how to use Jolokia.


## Example configuration [_example_configuration]

The ActiveMQ module supports the standard configuration options that are described in [Modules](/reference/metricbeat/configuration-metricbeat.md). Here is an example configuration:

```yaml
metricbeat.modules:
- module: activemq
  metricsets: ['broker', 'queue', 'topic']
  period: 10s
  hosts: ['localhost:8161']
  path: '/api/jolokia/?ignoreErrors=true&canonicalNaming=false'
  username: admin # default username
  password: admin # default password
```


## Metricsets [_metricsets_2]

The following metricsets are available:

* [broker](/reference/metricbeat/metricbeat-metricset-activemq-broker.md)
* [queue](/reference/metricbeat/metricbeat-metricset-activemq-queue.md)
* [topic](/reference/metricbeat/metricbeat-metricset-activemq-topic.md)




