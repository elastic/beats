::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


The system/users metricset reports logged in users and associated sessions via dbus and logind, which is a systemd component. By default, the metricset will look in `/var/run/dbus/` for a system socket, although a new path can be selected with `DBUS_SYSTEM_BUS_ADDRESS`.

This metricset is available on:

* Linux


## Configuration [_configuration_18]

There are no configuration options for this metricset.
