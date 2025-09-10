::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


The `network_summary` metricset collects detailed network metrics from the processes associated with a container. These stats come from `/proc/[PID]/net`, and are summed across the different namespaces found across the PIDs.

Because this metricset will try to access network counters from procfs, it is only available on linux.
