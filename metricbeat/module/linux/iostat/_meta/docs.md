::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


The iostat module reports per-disk IO statistics that emulate `iostat -x` on linux.

::::{note}
as of now, this data is part of system/diskio on Metricbeat, but can only be found in the Linux integration in Fleet. In the future, this data will be removed from system/memory.
::::

