# Main concepts
GCP Cloud Monitoring API == Cloudwatch :)

(Also known as Stackdriver before)

* Light modules
* Stackdriver requests are based on *gimme last 5s or 5m of compute.cpu.usage* so lots of requests are needed.
* *Some* metadata is included on GCP Cloud Monitoring API, not all.
* If you want the metadata that is missing, some interfaces must be implemented.
* GCP Cloud Monitoring API returns an array of points, each with a timestamp, values and labels. Their only correlation is the metric. So you can get 5 points, of 5 different minutes in time, 2 of them with labels a=b and 3 with c=d, but all of them will be `cpu.usage.pct` for example.
* There are 4 possible types of metadata available: User, Label, System and Metadata. This is called **Metadata** in code. (All 4 metadata types are collected under Metadata).
* In code there's the concept of Event ID. It's a "hash" of the timestamp and the Metadata. This is an "algorithm" which tries to group metric points into the same events. The idea is that if we get `X points * Y metrics` we don't send `X*Y` number of events but, instead, we group them into as few events as possible by trying to find which of them shares ID. There are implementations for GCP Cloud Monitoring API and compute based metrics.

# **root**
## `metadata.go`
3 Interfaces to implement special metadata retrieval for specific metricsets.

## `contants.go`
Avoid pieces of code like:
```go
// A value in seconds, right?
d.getMetric("sec")

// Not really... it was something related with security
d.getMetric(constants.FIELD_SECURITY_PATH)
```

## Other files

* `gcp/metadata_services.go`: returns a service to fetch metadata from a config struct
* `gcp/metrics_requester.go`: Knows the logic to request metrics to gcp, parse timestamps, etc.
* `gcp/metricset.go`: Good old friend. Also has the logic to create single events for each group of data that matches some common patterns like labels and timestamp.
* `gcp/response_parser.go`: Parses the incoming object from GCP Cloud Monitoring API in some in-between `KeyValuePoint` data that is similar to Elasticsearch events.
* `gcp/timeseries.go`: Groups TimeSeries responses into common Elasticsearch friendly events
* `gcp/compute/identity.go`: implemented by GCP services that can add some short of data to group their metrics (like instance id on Compute or topic in PubSub)
* `gcp/compute/metadata.go`: Specific compute metadata logic.

# Happy path

`metricbeat` -> `gcp/metricset.go` -> `metrics_requester.go` -> `response_parser.go` -> `metadata_services.go` (if labels=true, `gcp/compute.metadata.go` only `compute` metricset at the moment) -> `timeseries.go` (if labels=true `gcp/compute/identity.go` only compute ATM) -> `metricset.go` -> `elasticsearch`

Or in plain words:

* Use a requester to fetch metrics from GCP Cloud Monitoring API.
* Use the parser which knows how to get the key-values and where they are and create an array of `KeyValuePoint` which are a in-between data structure to work with in the metricset.
* If available, request also labels from the service and add them to the `KeyValuePoint`.
* Create N Elasticsearch-friendly events based on IDs created from `KeyValuePoint` and groping the ones that shares ID.
* Send them to Elasticsearch

# Launch stages

GCP metrics have 5 different stages: GA, BETA, ALPHA, EARLY_ACCESS, or DEPRECATED.

We only support GA metrics. Eventually we support BETA.
We do not support ALPHA or EARLY_ACCESS.

DEPRECATED metrics are not removed until after deprecation period expire and they get removed from GCP Monitoring API.

# Links

[Overview of GCP Metric list](https://cloud.google.com/monitoring/api/metrics)

# Update fields

Run `make update` to update `fields.go` from each metricset `fields.yml`

# Light-weight modules

The implementation is within `metrics` metricset. That metricset allows other metricsets to use it
as a "parent module" and implement the light-weigth module pattern.

# Running integration tests

Golang integration tests may be run with: `TEST_TAGS=gcp MODULE=gcp mage goIntegTest`

This command will exclude `gcp.billing` metricset, as without access to a Billing Account it will always return an empty set of metrics.
TODO: mock data so tests are not coupled with real GCP infrastructure.
