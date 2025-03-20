---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-module-golang.html
---

# Golang module [metricbeat-module-golang]

The golang module collects metrics by submitting HTTP GET requests to [golang-expvar-api](https://golang.org/pkg/expvar/).


## Example configuration [_example_configuration_27]

The Golang module supports the standard configuration options that are described in [Modules](/reference/metricbeat/configuration-metricbeat.md). Here is an example configuration:

```yaml
metricbeat.modules:
- module: golang
  #metricsets:
  #  - expvar
  #  - heap
  period: 10s
  hosts: ["localhost:6060"]
  heap.path: "/debug/vars"
  expvar:
    namespace: "example"
    path: "/debug/vars"
```

This module supports TLS connections when using `ssl` config field, as described in [SSL](/reference/metricbeat/configuration-ssl.md). It also supports the options described in [Standard HTTP config options](/reference/metricbeat/configuration-metricbeat.md#module-http-config-options).


## Metricsets [_metricsets_32]

The following metricsets are available:

* [expvar](/reference/metricbeat/metricbeat-metricset-golang-expvar.md)
* [heap](/reference/metricbeat/metricbeat-metricset-golang-heap.md)



