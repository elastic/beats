The MySQL `status` metricset collects data from MySQL by running a [`SHOW GLOBAL STATUS;`](http://dev.mysql.com/doc/refman/5.7/en/show-status.md) SQL query. This query returns a large number of metrics.

## raw config option [_raw_config_option]

::::{warning}
This functionality is in technical preview and may be changed or removed in a future release. Elastic will work to fix any issues, but features in technical preview are not subject to the support SLA of official GA features.
::::


The MySQL Status Metricset supports the `raw` config option. When enabled, in addition to the existing data structure, all fields available from the mysql service through `"SHOW /*!50002 GLOBAL */ STATUS;"` will be added to the event.

These fields will be added under the namespace `mysql.status.raw`. The fields can vary from one MySQL instance to an other and no guarantees are provided for the  mapping of the fields as the mapping happens dynamically. This option is intended for advanced use cases.

This is a default metricset. If the host module is unconfigured, this metricset is enabled by default.

