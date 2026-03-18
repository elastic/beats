:::::{admonition} Prefer to use {{agent}} for this use case?
Refer to the [Elastic Integrations documentation](integration-docs://reference/postgresql/index.md).

::::{dropdown} Learn more
{{agent}} is a single, unified way to add monitoring for logs, metrics, and other types of data to a host. It can also protect hosts from security threats, query data from operating systems, forward data from remote services or hardware, and more. Refer to the documentation for a detailed [comparison of {{beats}} and {{agent}}](docs-content://reference/fleet/index.md).

::::


:::::


This module periodically fetches metrics from [PostgreSQL](https://www.postgresql.org/) servers.

Default metricsets are `activity`, `bgwriter` and `database`.


## Dashboard [_dashboard_37]

The PostgreSQL module comes with a predefined dashboard showing databse related metrics. For example:

![metricbeat postgresql overview](images/metricbeat-postgresql-overview.png)


## Module-specific configuration notes [_module_specific_configuration_notes_17]

When configuring the `hosts` option, you must use Postgres URLs of the following format:

```
[postgres://][user:pass@]host[:port][?options]
```

The URL can be as simple as:

```yaml
- module: postgresql
  hosts: ["postgres://localhost"]
```

Or more complex like:

```yaml
- module: postgresql
  hosts: ["postgres://localhost:40001?sslmode=disable", "postgres://otherhost:40001"]
```

You can also separately specify the username and password using the respective configuration options. Usernames and passwords specified in the URL take precedence over those specified in the `username` and `password` config options.

```yaml
- module: postgresql
  metricsets: ["status"]
  hosts: ["postgres://localhost:5432"]
  username: root
  password: test
```


## Compatibility [_compatibility_43]

This module was tested with PostgreSQL 9, 10, 11, 12 and 13. It is expected to work with all versions >= 9.
