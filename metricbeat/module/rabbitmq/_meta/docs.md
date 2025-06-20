:::::{admonition} Prefer to use {{agent}} for this use case?
Refer to the [Elastic Integrations documentation](integration-docs://reference/rabbitmq/index.md).

::::{dropdown} Learn more
{{agent}} is a single, unified way to add monitoring for logs, metrics, and other types of data to a host. It can also protect hosts from security threats, query data from operating systems, forward data from remote services or hardware, and more. Refer to the documentation for a detailed [comparison of {{beats}} and {{agent}}](docs-content://reference/fleet/index.md).

::::


:::::


The RabbitMQ module uses [HTTP API](http://www.rabbitmq.com/management.html) created by the management plugin to collect metrics.

The default metricsets are `connection`, `node`, `queue`, `exchange` and `shovel`.

If `management.path_prefix` is set in RabbitMQ configuration, `management_path_prefix` has to be set to the same value in this module configuration.


## Compatibility [_compatibility_44]

The rabbitmq module is fully tested with RabbitMQ 3.7.4 and it should be compatible with any version supporting the management plugin (which needs to be installed and enabled). Exchange metricset is also tested with 3.6.0, 3.6.5 and 3.7.14
