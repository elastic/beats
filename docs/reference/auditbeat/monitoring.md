---
navigation_title: "Monitor"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/auditbeat/current/monitoring.html
---

# Monitor Auditbeat [monitoring]


You can use the {{stack}} {{monitor-features}} to gain insight into the health of Auditbeat instances running in your environment.

To monitor Auditbeat, make sure monitoring is enabled on your {{es}} cluster, then configure the method used to collect Auditbeat metrics. You can use one of following methods:

* [Internal collection](/reference/auditbeat/monitoring-internal-collection.md) - Internal collectors send monitoring data directly to your monitoring cluster.
* [{{metricbeat}} collection](/reference/auditbeat/monitoring-metricbeat-collection.md) - {{metricbeat}} collects monitoring data from your Auditbeat instance and sends it directly to your monitoring cluster.

