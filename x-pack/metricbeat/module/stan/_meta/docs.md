:::::{admonition} Prefer to use {{agent}} for this use case?
Refer to the [Elastic Integrations documentation](integration-docs://reference/stan/index.md).

::::{dropdown} Learn more
{{agent}} is a single, unified way to add monitoring for logs, metrics, and other types of data to a host. It can also protect hosts from security threats, query data from operating systems, forward data from remote services or hardware, and more. Refer to the documentation for a detailed [comparison of {{beats}} and {{agent}}](docs-content://reference/fleet/index.md).

::::


:::::


The STAN module uses [STAN monitoring server APIs](https://github.com/nats-io/nats-streaming-server/blob/master/server/monitor.go) to collect metrics.

The default metricsets are `channels`, `stats` and `subscriptions`.


### Compatibility [_compatibility_47]

The STAN module is tested with STAN 0.15.1.


## Dashboard [_dashboard_41]

Dashboards for topic message count and queue depth are included:

![metricbeat stan overview](images/metricbeat-stan-overview.png)
