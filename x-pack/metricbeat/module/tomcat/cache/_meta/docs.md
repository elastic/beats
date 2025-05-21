::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


Cache Metricset retrieves the JMX key `Catalina:context=*,host=*,name=Cache,type=WebResourceRoot`. It exposes the following metrics:

* `tomcat.cache.mbean`: Mbean that this event is related to.
* `tomcat.cache.hit.total`: The number of requests for resources that were served from the cache.
* `tomcat.cache.size.total.kb`: The current estimate of the cache size in kB
* `tomcat.cache.size.max.kb`: The maximum permitted size of the cache in kB
* `tomcat.cache.lookup.total`: The number of requests for resources
* `tomcat.cache.ttl.ms`: The time-to-live for cache entries in milliseconds

This is a default metricset. If the host module is unconfigured, this metricset is enabled by default.
