::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


Requests Metricset retrieves the JMX key `Catalina:name=*,type=GlobalRequestProcessor` using Jolokia. It exposes the following metrics:

* `tomcat.requests.mbean`: Number of requests processed
* `tomcat.requests.total`: Number of requests processed
* `tomcat.requests.bytes.received`: Amount of data received, in bytes
* `tomcat.requests.bytes.sent`: Amount of data sent, in bytes
* `tomcat.requests.processing.ms`: Total time to process the requests
* `tomcat.requests.errors.total`: Number of errors

This is a default metricset. If the host module is unconfigured, this metricset is enabled by default.
