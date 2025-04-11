---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-module-openai.html
  # That link will 404 until 8.18 is current
  # (see https://www.elastic.co/guide/en/beats/metricbeat/8.18/metricbeat-module-openai.html)
---

# openai module [metricbeat-module-openai]

::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


This is the openai module.


## Example configuration [_example_configuration_49]

The openai module supports the standard configuration options that are described in [Modules](configuration-metricbeat.md). Here is an example configuration:

```yaml
metricbeat.modules:
- module: openai
  metricsets: ["usage"]
  enabled: false
  period: 1h

  # # Project API Keys - Multiple API keys can be specified for different projects
  # api_keys:
  # - key: "api_key1"
  # - key: "api_key2"

  # # API Configuration
  # ## Base URL for the OpenAI usage API endpoint
  # api_url: "https://api.openai.com/v1/usage"
  # ## Custom headers to be included in API requests
  # headers:
  # - "k1: v1"
  # - "k2: v2"
  ## Rate Limiting Configuration
  # rate_limit:
  #   limit: 12 # seconds between requests
  #   burst: 1  # max concurrent requests
  # ## Request Timeout Duration
  # timeout: 30s

  # # Data Collection Configuration
  # collection:
  #   ## Number of days to look back when collecting usage data
  #   lookback_days: 30
  #   ## Whether to collect usage data in realtime. Defaults to false as how
  #   # OpenAI usage data is collected will end up adding duplicate data to ES
  #   # and also making it harder to do analytics. Best approach is to avoid
  #   # realtime collection and collect only upto last day (in UTC). So, there's
  #   # at most 24h delay.
  #   realtime: false
```


## Metricsets [_metricsets_56]

The following metricsets are available:

* [usage](metricbeat-metricset-openai-usage.md)


