---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-module-iis.html
---

# IIS module [metricbeat-module-iis]

:::::{admonition} Prefer to use {{agent}} for this use case?
Refer to the [Elastic Integrations documentation](integration-docs://reference/iis/index.md).

::::{dropdown} Learn more
{{agent}} is a single, unified way to add monitoring for logs, metrics, and other types of data to a host. It can also protect hosts from security threats, query data from operating systems, forward data from remote services or hardware, and more. Refer to the documentation for a detailed [comparison of {{beats}} and {{agent}}](docs-content://reference/fleet/index.md).

::::


:::::


IIS (Internet Information Services) is a secure, reliable, and scalable Web server that provides an easy to manage platform for developing and hosting Web applications and services.

The `iis` module will periodically retrieve IIS related metrics using performance counters such as:

* System/Process counters like the the overall server and CPU usage for the IIS Worker Process and memory (currently used and available memory for the IIS Worker Process).
* IIS performance counters like Web Service: Bytes Received/Sec, Web Service: Bytes Sent/Sec, etc, which are helpful to track to identify potential spikes in traffic.
* Web Service Cache counters in order to monitor user mode cache and output cache.

The `iis` module mericsets are `webserver`, `website` and `application_pool`.

```yaml
- module: iis
  metricsets:
    - webserver
    - website
    - application_pool
  enabled: true
  period: 10s

 # filter on application pool names
 # application_pool.name: []
```


## Metricsets [_metricsets_37]


### `webserver` [_webserver]

A light metricset using the windows perfmon metricset as the base metricset. This metricset allows users to retrieve aggregated metrics for the entire webserver,


### `website` [_website]

A light metricset using the windows perfmon metricset as the base metricset. This metricset will collect metrics of specific sites, users can configure which websites they want to monitor, else, all are considered.


### `application_pool` [_application_pool]

This metricset will collect metrics of specific application pools, users can configure which websites they want to monitor, else, all are considered.


### Module-specific configuration notes [_module_specific_configuration_notes_8]

`application_pool.name`
:   []string, users can specify the application pools they would like to monitor.


### Example configuration [_example_configuration_32]

The IIS module supports the standard configuration options that are described in [Modules](/reference/metricbeat/configuration-metricbeat.md). Here is an example configuration:

```yaml
metricbeat.modules:
- module: iis
  metricsets:
    - webserver
    - website
    - application_pool
  enabled: true
  period: 10s

 # filter on application pool names
 # application_pool.name: []
```


### Metricsets [_metricsets_38]

The following metricsets are available:

* [application_pool](/reference/metricbeat/metricbeat-metricset-iis-application_pool.md)
* [webserver](/reference/metricbeat/metricbeat-metricset-iis-webserver.md)
* [website](/reference/metricbeat/metricbeat-metricset-iis-website.md)




