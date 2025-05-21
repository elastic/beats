::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


The memory metricset extends system/memory and adds linux-specific memory metrics, including Huge Pages and overall paging statistics.

::::{note}
as of now, this data is part of system/memory on Metricbeat, but can only be found in the Linux integration in Fleet. In the future, this data will be removed from system/memory.
::::


This is a default metricset. If the host module is unconfigured, this metricset is enabled by default.
