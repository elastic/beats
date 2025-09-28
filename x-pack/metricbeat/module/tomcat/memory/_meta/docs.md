::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


Memory Metricset retrieves the JMX key `java.lang:type=Memory` using Jolokia. It exposes the following metrics:

* `tomcat.memory.heap.usage.committed`: Committed heap memory usage.
* `tomcat.memory.heap.usage.max`: Max heap memory usage.
* `tomcat.memory.heap.usage.used`: Used heap memory usage.
* `tomcat.memory.heap.usage.init`: Initial heap memory usage.
* `tomcat.memory.other.usage.committed`: Committed non-heap memory usage.
* `tomcat.memory.other.usage.max`: Max non-heap memory usage.
* `tomcat.memory.other.usage.used`: Used non-heap memory usage.
* `tomcat.memory.other.usage.init`: Initial non-heap memory usage.
