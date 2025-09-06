::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


Sync Gateway is the synchronization server in a Couchbase for Mobile and Edge deployment. This metricset allows to monitor a Sync Gateway instance by using its REST API.

Sync Gateway access `[host]:[port]/_expvar` on Sync Gateway nodes to fetch metrics data, ensure that the URL is accessible from the host where Metricbeat is running.
