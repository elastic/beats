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
