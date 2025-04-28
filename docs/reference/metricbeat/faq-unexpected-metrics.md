---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/faq-unexpected-metrics.html
---

# Metricbeat collects system metrics for interfaces you didn't configure [faq-unexpected-metrics]

The [System](/reference/metricbeat/metricbeat-module-system.md) module specifies several metricsets that are enabled by default unless you explicitly disable them. To disable a default metricset, comment it out in the `modules.d/system.yml` configuration file. If *all* metricsets are commented out and the System module is enabled, Metricbeat uses the default metricsets.

For example, to disable the `network` metricset, comment it out:

```yaml
  - module: system
    period: 10s
    metricsets:
      - cpu
      - load
      - memory
      #- network
      - process
      - process_summary
      - socket_summary
      #- entropy
      #- core
      #- diskio
      #- socket
```

You cannot override the default configuration by adding another module definition to the configuration. There is no concept of inheritance. Metricbeat combines all module configurations at runtime. This enables you to specify module definitions that use different combinations of metricsets, periods, and hosts.

