---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/libbeat/current/index.html
  - https://www.elastic.co/guide/en/beats/libbeat/current/beats-reference.html
  - https://www.elastic.co/guide/en/beats/libbeat/current/getting-started.html
  - https://www.elastic.co/guide/en/serverless/current/elasticsearch-ingest-data-through-beats.html
---

# Beats [beats-reference]

{{beats}} are open source data shippers that you install as agents on your servers to send operational data to [{{es}}](https://www.elastic.co/products/elasticsearch). Elastic provides {{beats}} for capturing:

Audit data
:   [Auditbeat overview](https://www.elastic.co/products/beats/auditbeat)
:   [Auditbeat documentation](/reference/auditbeat/index.md)

Log files and journals
:   [Filebeat overview](https://www.elastic.co/products/beats/filebeat)
:   [Filebeat documentation](/reference/filebeat/index.md)

Availability
:   [Heartbeat overview](https://www.elastic.co/products/beats/heartbeat)
:   [Heartbeat documentation](/reference/heartbeat/index.md)

Metrics
:   [Metricbeat overview](https://www.elastic.co/products/beats/metricbeat)
:   [Metricbeat documentation](/reference/metricbeat/index.md)

Network traffic
:   [Packetbeat overview](https://www.elastic.co/products/beats/packetbeat)
:   [Packetbeat documentation](/reference/packetbeat/index.md)

Windows event logs
:   [Winlogbeat overview](https://www.elastic.co/products/beats/winlogbeat)
:   [Winlogbeat documentation](/reference/winlogbeat/index.md)


{{beats}} can send data directly to {{es}} or through [{{ls}}](https://www.elastic.co/products/logstash), where you can further process and enhance the data, before visualizing it in [{{kib}}](https://www.elastic.co/products/logstash).

![Beats Platform](libbeat/images/beats-platform.png)

Want to get up and running quickly with infrastructure metrics monitoring and centralized log analytics? Try out the {{metrics-app}} and the {{logs-app}} in {{kib}}. For more details, check out [Analyze metrics](docs-content://solutions/observability/infra-and-hosts/analyze-infrastructure-host-metrics.md) and [Monitor logs](docs-content://solutions/observability/logs/explore-logs.md).


## Need to capture other kinds of data? [_need_to_capture_other_kinds_of_data]

If you have a specific use case to solve, we encourage you to create a [community Beat](/reference/libbeat/community-beats.md). We’ve created an infrastructure to simplify the process. The *libbeat* library, written entirely in Go, offers the API that all Beats use to ship data to Elasticsearch, configure the input options, implement logging, and more. To learn how to create a new Beat, see [Contribute to Beats](../extend/index.md).
