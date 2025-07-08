::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


This module periodically fetches metrics from a containerized distribution of IBM MQ.


## Compatibility [_compatibility_23]

The ibmmq `qmgr` metricset is compatible with a containerized distribution of IBM MQ (since version 9.1.0). The Docker image starts the `runmqserver` process, which spawns the HTTP server exposing metrics in Prometheus format ([source code](https://github.com/ibm-messaging/mq-container/blob/9.1.0/internal/metrics/metrics.go))).

The Docker container lifecycle, including metrics collection, has been described in the [Internals](https://github.com/ibm-messaging/mq-container/blob/9.1.0/docs/internals.md) document.

The image provides an option to easily enable metrics exporter using an environment variable:

`MQ_ENABLE_METRICS` - Set this to `true` to generate Prometheus metrics for the Queue Manager.


## Dashboard [_dashboard_26]

The ibmmq module includes predefined dashboards with overview information of the monitored Queue Manager, including subscriptions, calls and messages.

![metricbeat ibmmq calls](images/metricbeat-ibmmq-calls.png)

![metricbeat ibmmq messages](images/metricbeat-ibmmq-messages.png)

![metricbeat ibmmq subscriptions](images/metricbeat-ibmmq-subscriptions.png)
