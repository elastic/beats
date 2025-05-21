This module collects metrics from [Jolokia agents](https://jolokia.org/reference/html/agents.md) running on a target JMX server or dedicated proxy server. The default metricset is `jmx`.

To collect metrics, Metricbeat communicates with a Jolokia HTTP/REST endpoint that exposes the JMX metrics over HTTP/REST/JSON.


## Compatibility [_compatibility_25]

The Jolokia module is tested with Jolokia 1.5.0. It should work with version 1.2.2 and later.
