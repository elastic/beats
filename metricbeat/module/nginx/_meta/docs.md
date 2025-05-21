:::::{admonition} Prefer to use {{agent}} for this use case?
Refer to the [Elastic Integrations documentation](integration-docs://reference/nginx/index.md).

::::{dropdown} Learn more
{{agent}} is a single, unified way to add monitoring for logs, metrics, and other types of data to a host. It can also protect hosts from security threats, query data from operating systems, forward data from remote services or hardware, and more. Refer to the documentation for a detailed [comparison of {{beats}} and {{agent}}](docs-content://reference/fleet/index.md).

::::


:::::


This module periodically fetches metrics from [Nginx](https://nginx.org/) servers.

The default metricset is `stubstatus`.


## Compatibility [_compatibility_40]

The Nginx metricsets were tested with Nginx 1.23.2 and are expected to work with all version >= 1.9.


## Dashboard [_dashboard_35]

The nginx module comes with a predefined dashboard. For example:

![metricbeat nginx](images/metricbeat-nginx.png)
