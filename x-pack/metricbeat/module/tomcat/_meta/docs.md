::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


:::::{admonition} Prefer to use {{agent}} for this use case?
Refer to the [Elastic Integrations documentation](integration-docs://reference/tomcat/index.md).

::::{dropdown} Learn more
{{agent}} is a single, unified way to add monitoring for logs, metrics, and other types of data to a host. It can also protect hosts from security threats, query data from operating systems, forward data from remote services or hardware, and more. Refer to the documentation for a detailed [comparison of {{beats}} and {{agent}}](docs-content://reference/fleet/index.md).

::::


:::::


This module periodically fetches JMX metrics from Apache Tomcat.


### Compatibility [_compatibility_48]

The module has been tested with Tomcat 7.0.24 and 9.0.24. Other versions are expected to work.


## Dashboard [_dashboard_44]

An overview dashboard for Kibana is already included:

![metricbeat tomcat overview](images/metricbeat-tomcat-overview.png)


### Usage [_usage_9]

The Tomcat module requires [Jolokia](/reference/metricbeat/metricbeat-module-jolokia.md)to fetch JMX metrics. Refer to the link for instructions about how to use Jolokia.
