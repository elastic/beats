---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-sql.html
---

# SQL fields [exported-fields-sql]

SQL module fetches metrics from a SQL database

**`sql.driver`**
:   Driver used to execute the query.

type: keyword


**`sql.query`**
:   Query executed to collect metrics.

type: keyword


**`sql.metrics.numeric.*`**
:   Numeric metrics collected.

type: object


**`sql.metrics.string.*`**
:   Non-numeric values collected.

type: object


**`sql.metrics.boolean.*`**
:   Boolean values collected.

type: object


