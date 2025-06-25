The Ceph module collects metrics by submitting HTTP GET requests to the [ceph-rest-api](https://docs.ceph.com/docs/jewel/man/8/ceph-rest-api/). The default metricsets are `cluster_disk`, `cluster_health`, `monitor_health`, `pool_disk`, `osd_tree`.

Metricsets connecting to the Ceph REST API uses by default the service exposed on port 5000. Metricsets using the Ceph Manager Daemon communicate with the API exposed by default on port 8003 (SSL encryption).


## Compatibility [_compatibility_9]

The Ceph module is tested with Ceph Jewel (10.2.10) and Ceph Nautilus (14.2.7).

Metricsets with the `mgr_` prefix are compatible with Ceph releases using the Ceph Manager Daemon.


## Dashboard [_dashboard_20]

The Ceph module comes with a predefined dashboard showing Ceph cluster related metrics. For example:

![ceph overview dashboard](images/ceph-overview-dashboard.png)
