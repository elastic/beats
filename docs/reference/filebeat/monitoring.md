---
navigation_title: "Monitor"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/monitoring.html
---

# Monitor Filebeat [monitoring]


You can use the {{stack}} {{monitor-features}} to gain insight into the health of Filebeat instances running in your environment.

To monitor Filebeat, make sure monitoring is enabled on your {{es}} cluster, then configure the method used to collect Filebeat metrics. You can use one of following methods:

* [Internal collection](/reference/filebeat/monitoring-internal-collection.md) - Internal collectors send monitoring data directly to your monitoring cluster.
* [{{metricbeat}} collection](/reference/filebeat/monitoring-metricbeat-collection.md) - {{metricbeat}} collects monitoring data from your Filebeat instance and sends it directly to your monitoring cluster.

