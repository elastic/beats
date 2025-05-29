---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/how-metricbeat-works.html
---

# How Metricbeat works [how-metricbeat-works]

Metricbeat consists of modules and metricsets. A Metricbeat *module* defines the basic logic for collecting data from a specific service, such as Redis, MySQL, and so on. The module specifies details about the service, including how to connect, how often to collect metrics, and which metrics to collect.

Each module has one or more metricsets. A *metricset* is the part of the module that fetches and structures the data. Rather than collecting each metric as a separate event, metricsets retrieve a list of multiple related metrics in a single request to the remote system. So, for example, the Redis module provides an `info` metricset that collects information and statistics from Redis by running the [`INFO`](http://redis.io/commands/INFO) command and parsing the returned result.

![Modules and metricsets](images/module-overview.png)

Likewise, the MySQL module provides a `status` metricset that collects data from MySQL by running a [`SHOW GLOBAL STATUS`](http://dev.mysql.com/doc/refman/5.7/en/show-status.md) SQL query. Metricsets make it easier for you by grouping sets of related metrics together in a single request returned by the remote server. Most modules have default metricsets that are enabled if there are no user-enabled metricsets.

Metricbeat retrieves metrics by periodically interrogating the host system based on the `period` value that you specify when you configure the module. Because multiple metricsets can send requests to the same service, Metricbeat reuses connections whenever possible. If Metricbeat cannot connect to the host system within the time specified by the `timeout` config setting, it returns an error. Metricbeat sends the events asynchronously, which means the event retrieval is not acknowledged. If the configured output is not available, events may be lost.

When Metricbeat encounters an error (for example, when it cannot connect to the host system), it sends an event error to the specified output. This means that Metricbeat always sends an event, even when there is a failure. This allows you to monitor for errors and see debug messages to help you diagnose what went wrong.

The following topics provide more detail about the structure of Metricbeat events:

* [Event structure](/reference/metricbeat/metricbeat-event-structure.md)
* [Error event structure](/reference/metricbeat/error-event-structure.md)

For more about the benefits of using Metricbeat, see [Key metricbeat features](/reference/metricbeat/key-features.md).




