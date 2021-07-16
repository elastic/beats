# Main concepts
Stackdriver == Cloudwatch :)

* Light modules
* Stackdriver requests are based on *gimme last 5s or 5m of compute.cpu.usage* so lots of requests are needed.
* *Some* metadata is included on Stackdriver, not all.
* If you want the metadata that is missing, some interfaces must be implemented.
* Stackdriver returns an array of points, each with a timestamp, values and labels. Their only correlation is the metric. So you can get 5 points, of 5 different minutes in time, 2 of them with labels a=b and 3 with c=d, but all of them will be `cpu.usage.pct` for example.
* There are 4 possible types of metadata available: User, Label, System and Metadata. This is called **Metadata** in code.
* In code there's the concept of Event ID. It's a "hash" of the timestamp and the Metadata. This is an "algorithm" which tries to group metric points into the same events. The idea is that if we get `X points *
Y metrics` we don't send `X*Y` number of events but, instead, we group them into as few events as possible by trying to find which of them shares ID. There are implementations for stackdriver and compute based metrics.

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

* `stackdriver/metadata_services.go`: returns a service to fetch metadata from a config struct
* `stackdriver/metrics_requester.go`: Knows the logic to request metrics to Stackdriver, parse timestamps, etc.
* `stackdriver/metricset.go`: Good old friend. Also has the logic to create single events for each group of data that matches some common patterns like labels and timestamp.
* `stackdriver/response_parser.go`: Parses the incoming object from stackdriver in some in-between `KeyValuePoint` data that is similar to Elasticsearch events.
* `stackdriver/timeseries.go`: Groups TimeSeries responses into common Elasticsearch friendly events
* `stackdriver/compute/identity.go`: implemented by GCP services that can add some short of data to group their metrics (like instance id on Compute or topic in PubSub)
* `stackdriver/compute/metadata.go`: Specific compute metadata logic.

# Happy path

`metricbeat` -> `stackdriver/metricset.go` -> `metrics_requester.go` -> `response_parser.go` -> `metadata_services.go` (if labels=true, `stackdriver/compute.metadata.go` only `compute` metricset at the moment) -> `timeseries.go` (if labels=true `stackdriver/compute/identity.go` only compute ATM) -> `metricset.go` -> `elasticsearch`

Or in plain words:

* Use a requester to fetch metrics from Stackdriver.
* Use the parser which knows how to get the key-values and where they are and create an array of `KeyValuePoint` which are a in-between data structure to work with in the metricset.
* If available, request also labels from the service and add them to the `KeyValuePoint`.
* Create N Elasticsearch-friendly events based on IDs created from `KeyValuePoint` and groping the ones that shares ID.
* Send them to Elasticsearch
