---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/heartbeat/current/change-index-name.html
---

# Change the index name [change-index-name]

Heartbeat uses data streams named `heartbeat-[version]`. To use a different name, set the [`index`](/reference/heartbeat/elasticsearch-output.md#index-option-es) option in the {{es}} output. You also need to configure the `setup.template.name` and `setup.template.pattern` options to match the new name. For example:

```sh
output.elasticsearch.index: "customname-%{[agent.version]}"
setup.template.name: "customname-%{[agent.version]}"
setup.template.pattern: "customname-%{[agent.version]}"
```

For a full list of template setup options, see [Elasticsearch index template](/reference/heartbeat/configuration-template.md).

Remember to change the index name when you load dashboards via the Kibana UI.

