---
navigation_title: "Monitor"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/monitoring.html
---

# Monitor Metricbeat [monitoring]


You can use the {{stack}} {{monitor-features}} to gain insight into the health of Metricbeat instances running in your environment.

To monitor Metricbeat, make sure monitoring is enabled on your {{es}} cluster, then configure the method used to collect Metricbeat metrics. You can use one of following methods:

* [Internal collection](/reference/metricbeat/monitoring-internal-collection.md) - Internal collectors send monitoring data directly to your monitoring cluster.
* [{{metricbeat}} collection](/reference/metricbeat/monitoring-metricbeat-collection.md) - {{metricbeat}} collects monitoring data from your Metricbeat instance and sends it directly to your monitoring cluster.

