---
navigation_title: "Monitor"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/winlogbeat/current/monitoring.html
---

# Monitor Winlogbeat [monitoring]


You can use the {{stack}} {{monitor-features}} to gain insight into the health of Winlogbeat instances running in your environment.

To monitor Winlogbeat, make sure monitoring is enabled on your {{es}} cluster, then configure the method used to collect Winlogbeat metrics. You can use one of following methods:

* [Internal collection](/reference/winlogbeat/monitoring-internal-collection.md) - Internal collectors send monitoring data directly to your monitoring cluster.
* [{{metricbeat}} collection](/reference/winlogbeat/monitoring-metricbeat-collection.md) - {{metricbeat}} collects monitoring data from your Winlogbeat instance and sends it directly to your monitoring cluster.

