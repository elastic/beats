:::::{admonition} Prefer to use {{agent}} for this use case?
Refer to the [Elastic Integrations documentation](integration-docs://reference/nats/index.md).

::::{dropdown} Learn more
{{agent}} is a single, unified way to add monitoring for logs, metrics, and other types of data to a host. It can also protect hosts from security threats, query data from operating systems, forward data from remote services or hardware, and more. Refer to the documentation for a detailed [comparison of {{beats}} and {{agent}}](docs-content://reference/fleet/index.md).

::::


:::::


The Nats module uses [Nats monitoring server APIs](https://docs.nats.io/running-a-nats-service/nats_admin/monitoring) to collect metrics.

The default metricsets are `stats`, `connections`, `routes` and `subscriptions`. The `connection`, `route`, and `jetstream` metricsets can be enabled to collect additional metrics.

## Compatibility [_compatibility_39]

The NATS module is tested with NATS 2.2.6 and 2.11.x. Versions in between are expected to be compatible as well.


## Dashboard [_dashboard_34]

The Nats module comes with a predefined dashboard. For example:

![metricbeat nats dashboard](images/metricbeat_nats_dashboard.png)
