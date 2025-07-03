:::::{admonition} Prefer to use {{agent}} for this use case?
Refer to the [Elastic Integrations documentation](integration-docs://reference/windows/index.md).

::::{dropdown} Learn more
{{agent}} is a single, unified way to add monitoring for logs, metrics, and other types of data to a host. It can also protect hosts from security threats, query data from operating systems, forward data from remote services or hardware, and more. Refer to the documentation for a detailed [comparison of {{beats}} and {{agent}}](docs-content://reference/fleet/index.md).

::::


:::::


This is the `windows` module which collects metrics from Windows systems. The module contains the `service` metricset, which is set up by default when the `windows` module is enabled. The `service` metricset will retrieve status information of the services on the Windows machines. The second `windows` metricset is `perfmon` which collects Windows performance counter values.
