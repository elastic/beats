---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-metricset-mysql-galera_status.html
---

# MySQL galera_status metricset [metricbeat-metricset-mysql-galera_status]

::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


This module periodically fetches metrics from [Galera](http://galeracluster.com/)-MySQL cluster servers.


## Module-specific configuration notes [_module_specific_configuration_notes_14]

When configuring the `hosts` option, you must use a MySQL Data Source Name (DSN) of the following format:

```
[username[:password]@][protocol[(address)]]/
```

You can also separately specify the username and password using the respective configuration options. Usernames and passwords specified in the DSN take precedence over those specified in the `username` and `password` config options.

```
- module: mysql
  metricsets: ["status"]
  hosts: ["tcp(127.0.0.1:3306)/"]
  username: root
  password: secret
```


## Compatibility [_compatibility_38]

The galera MetricSets were tested with galera 3.22 (MySQL 5.7.20) and are expected to work with all versions >= 3.0 (MySQL >= 5.7.0)

