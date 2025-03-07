---
navigation_title: "Monitor"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/monitoring.html
---

# Monitor Packetbeat [monitoring]


You can use the {{stack}} {{monitor-features}} to gain insight into the health of Packetbeat instances running in your environment.

To monitor Packetbeat, make sure monitoring is enabled on your {{es}} cluster, then configure the method used to collect Packetbeat metrics. You can use one of following methods:

* [Internal collection](/reference/packetbeat/monitoring-internal-collection.md) - Internal collectors send monitoring data directly to your monitoring cluster.
* [{{metricbeat}} collection](/reference/packetbeat/monitoring-metricbeat-collection.md) - {{metricbeat}} collects monitoring data from your Packetbeat instance and sends it directly to your monitoring cluster.

