
The KSM module reports data from [Kernel Samepage Merging](https://www.kernel.org/doc/html/latest/admin-guide/mm/ksm.html). In order to take advantage of KSM, applications must use the `madvise` system call to mark memory regions for merging. KSM is not enabled on all distros, and KSM status is set with the `CONFIG_KSM` kernel flag.
