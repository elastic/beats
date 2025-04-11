---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-metricset-jolokia-jmx.html
---

# Jolokia jmx metricset [metricbeat-metricset-jolokia-jmx]

The `jmx` metricset collects metrics from [Jolokia agents](https://jolokia.org/reference/html/agents.md).


## Features and configuration [_features_and_configuration_2]

Tested with Jolokia 1.5.0.

To collect metrics from a Jolokia instance, define a mapping section that specifies an MBean `ObjectName` followed by an array of attributes to fetch. For each attribute in the array, specify the Elastic field name where the returned value will be saved.

For example, to get the `Uptime` attribute from the `java.lang:type=Runtime` MBean and map it to an event field called `jolokia.testnamespace.uptime`, configure the following mapping:

```yaml
- module: jolokia
  metricsets: ["jmx"]
  hosts: ["localhost:8778"]
  namespace: "testnamespace" <1>
  http_method: "POST" <2>
  jmx.mappings:
    - mbean: 'java.lang:type=Runtime'
      attributes:
        - attr: Uptime
          field: uptime <3>
          event: uptime <4>
      target:
          url: "service:jmx:rmi:///jndi/rmi://targethost:9999/jmxrmi"
          user: "jolokia"
          password: "s!cr!t"
```

1. The `namespace` setting is required. This setting is used along with the module name to qualify field names in the output event.
2. The `http_method` setting is optional. By default all requests to Jolokia are performed using `POST` HTTP method. This setting allows only two values: `POST` or `GET`.
3. The field where the returned value will be saved. This field will be called `jolokia.testnamespace.uptime` in the output event.
4. The `event` setting is optional. Use this setting to group all attributes with the same `event` value into the same event when sending data to Elastic.


If the underlying attribute is an object (such as the `HeapMemoryUsage` attribute in `java.lang:type=Memory`), its structure will be published to Elastic "as is".

You can configure nested metric aliases by using dots in the mapping name (for example, `gc.cms_collection_time`). For more examples, see [/jolokia/jmx/test/config.yml](https://github.com/elastic/beats/blob/master/metricbeat/module/jolokia/jmx/_meta/test/config.yml).

All metrics from a single mapping will be POSTed to the defined host/port and sent to Elastic as a single event. To make it possible to differentiate between metrics from multiple similar applications running on the same host, you should configure multiple modules.

When wildcards are used, an event is sent to Elastic for each matching MBean, and an `mbean` field is added to the event.


## Accessing Jolokia via POST or GET method [_accessing_jolokia_via_post_or_get_method]

All requests to Jolokia are made by default using HTTP POST method. However, there are specific circumstances on the environment where Jolokia agent is deployed, in which POST method can be unavailable. In this case you can use HTTP GET method, by defining `http_method` attribute. In general you can use either POST or GET, but GET has the following drawbacks:

1. [Proxy requests](https://jolokia.org/reference/html/protocol.md#protocol-proxy) are not allowed.
2. If more than one `jmx.mappings` are defined, then Metricbeat will perform as many GET requests as the mappings defined. For example the following configuration with 3 mappings will create 3 GET requests, one for every MBean. On the contrary, if you use HTTP POST, Metricbeat will create only 1 request to Jolokia.

```yaml
- module: jolokia
  metricsets: ["jmx"]
  enabled: true
  period: 10s
  hosts: ["localhost:8080"]
  namespace: "jolokia_metrics"
  path: "/jolokia"
  http_method: 'GET'
  jmx.mappings:
    - mbean: 'java.lang:type=Memory'
      attributes:
       - attr: HeapMemoryUsage
         field: memory.heap_usage
       - attr: NonHeapMemoryUsage
         field: memory.non_heap_usage
    - mbean: 'Catalina:name=*,type=ThreadPool'
      attributes:
       - attr: port
         field: catalina.port
       - attr: maxConnections
         field: catalina.maxConnections
    - mbean: 'java.lang:type=Runtime'
      attributes:
       - attr: Uptime
         field: uptime
```


## Limitations [_limitations]

1. All Jolokia requests have `canonicalNaming` set to `false`. See the [Jolokia Protocol](https://jolokia.org/reference/html/protocol.md) documentation for more detail about this parameter.
2. If `http_method` is set to `GET`, then [Proxy requests](https://jolokia.org/reference/html/protocol.md#protocol-proxy) are not allowed. Thus, setting a value to `target` section is going to fail with an error.


## Exposed fields, dashboards, indexes, etc. [_exposed_fields_dashboards_indexes_etc_2]

Because this module is very general and can be tailored for any application that exposes its metrics over Jolokia, it comes with no exposed field descriptions, dashboards, or index patterns.

## Fields [_fields_127]

For a description of each field in the metricset, see the [exported fields](/reference/metricbeat/exported-fields-jolokia.md) section.

Here is an example document generated by this metricset:

```json
{
    "@timestamp": "2017-10-12T08:05:34.853Z",
    "event": {
        "dataset": "jolokia.testnamespace",
        "duration": 115000,
        "module": "jolokia"
    },
    "jolokia": {
        "testnamespace": {
            "memory": {
                "heap_usage": {
                    "committed": 514850816,
                    "init": 536870912,
                    "max": 7635730432,
                    "used": 42335648
                },
                "non_heap_usage": {
                    "committed": 32243712,
                    "init": 2555904,
                    "max": -1,
                    "used": 29999896
                }
            },
            "uptime": 70795470
        }
    },
    "metricset": {
        "name": "jmx"
    },
    "service": {
        "address": "127.0.0.1:8778",
        "type": "jolokia"
    }
}
```


