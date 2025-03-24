---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/winlogbeat/current/_winlogbeat_overview.html
  - https://www.elastic.co/guide/en/beats/winlogbeat/current/index.html
---

# Winlogbeat

Winlogbeat ships Windows event logs to Elasticsearch or Logstash. You can install it as a Windows service.

Winlogbeat reads from one or more event logs using Windows APIs, filters the events based on user-configured criteria, then sends the event data to the configured outputs (Elasticsearch or Logstash). Winlogbeat watches the event logs so that new event data is sent in a timely manner. The read position for each event log is persisted to disk to allow Winlogbeat to resume after restarts.

Winlogbeat can capture event data from any event logs running on your system. For example, you can capture events such as:

* application events
* hardware events
* security events
* system events

Winlogbeat is an Elastic [Beat](https://www.elastic.co/beats). It’s based on the `libbeat` framework. For more information, see the [Beats Platform Reference](/reference/index.md).

