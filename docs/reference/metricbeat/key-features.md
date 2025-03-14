---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/key-features.html
---

# Key metricbeat features [key-features]

Metricbeat has some key features that are critical to how it works:

* [Metricbeat error events](#metricbeat-error-events)
* [No aggregations when data is fetched](#no-aggregations)
* [Sends more than just numbers](#more-than-numbers)
* [Multiple metrics in one event](#multiple-events-in-one)

## Metricbeat error events [metricbeat-error-events]

Metricbeat sends more than just metrics. When it cannot retrieve metrics, it sends error events. The error is not simply a flag, but a full error string that is created during fetching from the host systems. This enables you to monitor not only the metrics, but also any errors that occur during metrics monitoring.

Because you see the full error message, you can track down the error faster. Metricbeat is installed locally on the host machine, which means that you can differentiate errors that happen locally from other issues, such as network problems.

Each metricset is retrieved based on a predefined period, so when Metricbeat fails to retrieve metrics for more than one interval, you can infer that there is potentially something wrong with the host or host connectivity.


## No aggregations when data is fetched [no-aggregations]

Metricbeat doesn’t do aggregations like gauge, sum, counters, and so on. Metricbeat sends the raw data retrieved from the host to the output for processing. When using Elasticsearch, this has the advantage that all raw data is available on the Elasticsearch host for drilling down into the details, and the data can be reprocessed at any time. It also reduces the complexity of Metricbeat.


## Sends more than just numbers [more-than-numbers]

Metricbeat sends more than just numbers. The metrics that Metricbeat sends can also contain strings to report status information. This is useful when you’re using Elasticsearch to store the metrics data. Because each metricset has a predefined structure, Elasticsearch knows in advance which types will be stored in Elasticsearch, and it can optimize storage.

Basic meta information about each metric (such as the host) is also sent as part of each event.


## Multiple metrics in one event [multiple-events-in-one]

Rather than containing a single metric, each event created by Metricbeat contains a list of metrics. This means that you can retrieve all the metrics in a single request to the host system, resulting in less load on the host system. If you are sending the metrics to Elasticsearch as the output, Elasticsearch can directly store and query the metrics as a nested JSON document, making it very efficient for sending metrics data to Elasticsearch.

Because the full raw event data is available, Metricbeat or Elasticsearch can do any required transformations on the data later. For example, if you need to store data in the [Metrics2.0](http://metrics20.org/) format, you could generate the format out of the existing event by splitting up the full event into multiple metrics2.0 events.

Meta information about the type of each metric is stored in the mapping template. Meta information that is common to all metric events, such as host and timestamp, is part of the event structure itself  and is only stored once for all events in the metricset.

Having all the related metrics in a single event also makes it easier to look at other values when one of the metrics for a service seems off.


