::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


This is the Istio module. When using versions prior to `1.5` then the `mesh`, `mixer`, `pilot`, `galley`, `citadel` metricsets should be used.

In such case, the Istio module collects metrics from the pre v1.5 Istio [prometheus exporters endpoints](https://istio.io/v1.4/docs/tasks/observability/metrics/querying-metrics/#about-the-prometheus-add-on).

For versions after `1.5`, the `istiod` and `proxy` metricsets should be used. In such case, the `istiod` endpoint collects metrics directly from the Istio Daemon while the `proxy` endpoint collects from each of the proxy sidecars. The metrics exposed by Istio after version `1.5` are documented on [Istio Documentation > Reference > Configuration > Istio Standard Metrics](https://istio.io/latest/docs/reference/config/metrics/).


## Compatibility [_compatibility_24]

The Istio module is tested with Istio `1.4` for `mesh`, `mixer`, `pilot`, `galley`, `citadel`. The Istio module is tested with Istio `1.7` for `istiod` and `proxy`.


## Dashboard [_dashboard_30]

The Istio module includes predefined dashboards:

1. Overview information about Istio Daemon.
2. Traffic information collected from istio-proxies.

These dashboards are only compatible with versions of Istio after `1.5` which should be monitored with `istiod` and `proxy` metricsets.

![metricbeat istio overview](images/metricbeat-istio-overview.png)

![metricbeat istio traffic](images/metricbeat-istio-traffic.png)
