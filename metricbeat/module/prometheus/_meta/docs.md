:::::{admonition} Prefer to use {{agent}} for this use case?
Refer to the [Elastic Integrations documentation](integration-docs://reference/prometheus/index.md).

::::{dropdown} Learn more
{{agent}} is a single, unified way to add monitoring for logs, metrics, and other types of data to a host. It can also protect hosts from security threats, query data from operating systems, forward data from remote services or hardware, and more. Refer to the documentation for a detailed [comparison of {{beats}} and {{agent}}](docs-content://reference/fleet/index.md).

::::


:::::


$$$prometheus-module$$$
This module periodically scrapes metrics from [Prometheus exporters](https://prometheus.io/docs/instrumenting/exporters/).


## Dashboard [_dashboard_38]

The Prometheus module comes with a predefined dashboard for Prometheus specific stats. For example:

![metricbeat prometheus overview](images/metricbeat-prometheus-overview.png)
