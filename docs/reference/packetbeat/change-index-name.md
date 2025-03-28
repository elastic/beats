---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/change-index-name.html
---

# Change the index name [change-index-name]

Packetbeat uses data streams named `packetbeat-[version]`. To use a different name, set the [`index`](/reference/packetbeat/elasticsearch-output.md#index-option-es) option in the {{es}} output. You also need to configure the `setup.template.name` and `setup.template.pattern` options to match the new name. For example:

```sh
output.elasticsearch.index: "customname-%{[agent.version]}"
setup.template.name: "customname-%{[agent.version]}"
setup.template.pattern: "customname-%{[agent.version]}"
```

If you’re using pre-built Kibana dashboards, also set the `setup.dashboards.index` option. For example:

```yaml
setup.dashboards.index: "customname-*"
```

For a full list of template setup options, see [Elasticsearch index template](/reference/packetbeat/configuration-template.md).

