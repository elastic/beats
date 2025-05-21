::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


Threading Metricset retrieves the JMX key `Catalina:name=*,type=ThreadPool` and `java.lang:type=Threading` using Jolokia. It exposes the following metrics:

* `tomcat.threading.busy`: Current busy threads from the ThreadPool
* `tomcat.threading.max`: Max threads from the ThreadPool
* `tomcat.threading.current`: Current number of threads, taken from the ThreadPool
* `tomcat.threading.keep_alive.total`: Total keep alive on the ThreadPool
* `tomcat.threading.keep_alive.timeout.ms`: Keep alive timeout on the ThreadPool
* `tomcat.threading.started.total`: Current started threads at JVM level (from java.lang:type=Threading)
* `tomcat.threading.user.time.ms`: User time in milliseconds (from java.lang:type=Threading)
* `tomcat.threading.cpu.time.ms`: CPU time in milliseconds (from java.lang:type=Threading)
* `tomcat.threading.total`: Total threads at the JVM level (from java.lang:type=Threading)
* `tomcat.threading.peak`: Peak number of threads at JVM level (from java.lang:type=Threading)

This is a default metricset. If the host module is unconfigured, this metricset is enabled by default.
