::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


The Pressure module reports [Pressure Stall Information (PSI)](https://www.kernel.org/doc/Documentation/accounting/psi.txt) collected for the `cpu`, `memory`, and `io` files/resources found in `/proc/pressure`. PSI metrics are included in Linux kernel versions from 4.20. Some distributions might have PSI support, but have disabled the feature via the `CONFIG_PSI_DEFAULT_DISABLED` setting, to enable PSI metrics pass `psi=1` on the kernel command line during boot.
