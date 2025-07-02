::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


The KSM module reports data from [Kernel Samepage Merging](https://www.kernel.org/doc/html/latest/admin-guide/mm/ksm.html). In order to take advantage of KSM, applications must use the `madvise` system call to mark memory regions for merging. KSM is not enabled on all distros, and KSM status is set with the `CONFIG_KSM` kernel flag.
