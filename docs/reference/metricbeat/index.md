---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-overview.html
  - https://www.elastic.co/guide/en/beats/metricbeat/current/index.html
---

# Metricbeat

Metricbeat is a lightweight shipper that you can install on your servers to periodically collect metrics from the operating system and from services running on the server. Metricbeat takes the metrics and statistics that it collects and ships them to the output that you specify, such as Elasticsearch or Logstash.

Metricbeat helps you monitor your servers by collecting metrics from the system and services running on the server, such as:

* [Apache](/reference/metricbeat/metricbeat-module-apache.md)
* [HAProxy](/reference/metricbeat/metricbeat-module-haproxy.md)
* [MongoDB](/reference/metricbeat/metricbeat-module-mongodb.md)
* [MySQL](/reference/metricbeat/metricbeat-module-mysql.md)
* [Nginx](/reference/metricbeat/metricbeat-module-nginx.md)
* [PostgreSQL](/reference/metricbeat/metricbeat-module-postgresql.md)
* [Redis](/reference/metricbeat/metricbeat-module-redis.md)
* [System](/reference/metricbeat/metricbeat-module-system.md)
* [Zookeeper](/reference/metricbeat/metricbeat-module-zookeeper.md)

See [Modules](/reference/metricbeat/metricbeat-modules.md) for the complete list of supported services.

Metricbeat can insert the collected metrics directly into Elasticsearch or send them to Logstash, Redis, or Kafka.

Metricbeat is an Elastic [Beat](https://www.elastic.co/beats). It’s based on the `libbeat` framework. For more information, see the [Beats Platform Reference](/reference/index.md).

