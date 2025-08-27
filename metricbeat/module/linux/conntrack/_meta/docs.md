::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


The conntrack module reports on performance counters for the linux connection tracking component of netfilter. Conntrack uses a [hash table](http://people.netfilter.org/pablo/docs/login.pdf) to track the state of network connections.

This metricset traditionally reads performance data from the now-obsolete file `/proc/net/stat/nf_conntrack`, which requires the nf_conntrack kernel module and the CONFIG_NF_CONNTRACK_PROCFS option enabled. As of mid-2022, this configuration option is disabled by default, and most modern distributions no longer expose these procfs entries.

The preferred method for collecting conntrack metrics is via the Netlink interface. If procfs data is unavailable, the metricset will fall back to collecting metrics via Netlink—but note that this requires running Metricbeat with root privileges.
