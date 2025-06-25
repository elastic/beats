::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


The `service` metricset reports on the status of systemd services.

This metricset is available on:

* Linux


## systemd resource accounting and process metrics [_systemd_resource_accounting_and_process_metrics]

If systemd resource accounting is enabled, this metricset will report any resources tracked by systemd. On most distributions, `tasks` and `memory` are the only resources with accounting enabled by default. For more information, [see the systemd manual pages](https://www.freedesktop.org/software/systemd/man/systemd.resource-control.html).


## Configuration [_configuration_14]

**`service.state_filter`** - A list of service states to filter by. This can be any of the states or sub-states known to systemd. **`service.pattern_filter`** - A list of glob patterns to filter service names by. This is an "or" filter, and will report any systemd unit that matches at least one filter pattern.


## Dashboard [_dashboard_43]

The system service metricset comes with a predefined dashboard. For example:

![metricbeat services host](images/metricbeat-services-host.png)
