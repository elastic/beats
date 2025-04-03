---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-osquery.html
---

# Osquery fields [exported-fields-osquery]

Fields exported by the `osquery` module


## osquery [_osquery]


## result [_result]

Common fields exported by the result metricset.

**`osquery.result.name`**
:   The name of the query that generated this event.

type: keyword


**`osquery.result.action`**
:   For incremental data, marks whether the entry was added or removed. It can be one of "added", "removed", or "snapshot".

type: keyword


**`osquery.result.host_identifier`**
:   The identifier for the host on which the osquery agent is running. Normally the hostname.

type: keyword


**`osquery.result.unix_time`**
:   Unix timestamp of the event, in seconds since the epoch. Used for computing the `@timestamp` column.

type: long


**`osquery.result.calendar_time`**
:   String representation of the collection time, as formatted by osquery.

type: keyword


