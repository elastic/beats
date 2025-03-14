---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-module-jolokia.html
---

# Jolokia module [metricbeat-module-jolokia]

This module collects metrics from [Jolokia agents](https://jolokia.org/reference/html/agents.md) running on a target JMX server or dedicated proxy server. The default metricset is `jmx`.

To collect metrics, Metricbeat communicates with a Jolokia HTTP/REST endpoint that exposes the JMX metrics over HTTP/REST/JSON.


## Compatibility [_compatibility_25]

The Jolokia module is tested with Jolokia 1.5.0. It should work with version 1.2.2 and later.


## Example configuration [_example_configuration_34]

The Jolokia module supports the standard configuration options that are described in [Modules](/reference/metricbeat/configuration-metricbeat.md). Here is an example configuration:

```yaml
metricbeat.modules:
- module: jolokia
  #metricsets: ["jmx"]
  period: 10s
  hosts: ["localhost"]
  namespace: "metrics"
  #path: "/jolokia/?ignoreErrors=true&canonicalNaming=false"
  #username: "user"
  #password: "secret"
  jmx.mappings:
    #- mbean: 'java.lang:type=Runtime'
    #  attributes:
    #    - attr: Uptime
    #      field: uptime
    #- mbean: 'java.lang:type=Memory'
    #  attributes:
    #    - attr: HeapMemoryUsage
    #      field: memory.heap_usage
    #    - attr: NonHeapMemoryUsage
    #      field: memory.non_heap_usage
    # GC Metrics - this depends on what is available on your JVM
    #- mbean: 'java.lang:type=GarbageCollector,name=ConcurrentMarkSweep'
    #  attributes:
    #    - attr: CollectionTime
    #      field: gc.cms_collection_time
    #    - attr: CollectionCount
    #      field: gc.cms_collection_count

  jmx.application:
  jmx.instance:
```

This module supports TLS connections when using `ssl` config field, as described in [SSL](/reference/metricbeat/configuration-ssl.md). It also supports the options described in [Standard HTTP config options](/reference/metricbeat/configuration-metricbeat.md#module-http-config-options).


## Metricsets [_metricsets_40]

The following metricsets are available:

* [jmx](/reference/metricbeat/metricbeat-metricset-jolokia-jmx.md)


