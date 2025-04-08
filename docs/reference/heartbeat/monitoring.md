---
navigation_title: "Monitor"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/heartbeat/current/monitoring.html
---

# Monitor Heartbeat [monitoring]


You can use the {{stack}} {{monitor-features}} to gain insight into the health of Heartbeat instances running in your environment.

To monitor Heartbeat, make sure monitoring is enabled on your {{es}} cluster, then configure the method used to collect Heartbeat metrics. You can use one of following methods:

* [Internal collection](/reference/heartbeat/monitoring-internal-collection.md) - Internal collectors send monitoring data directly to your monitoring cluster.
* [{{metricbeat}} collection](/reference/heartbeat/monitoring-metricbeat-collection.md) - {{metricbeat}} collects monitoring data from your Heartbeat instance and sends it directly to your monitoring cluster.

