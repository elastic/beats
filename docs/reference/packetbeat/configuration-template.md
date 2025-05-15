---
navigation_title: "Elasticsearch index template"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/configuration-template.html
---

# Configure Elasticsearch index template loading [configuration-template]


The `setup.template` section of the `packetbeat.yml` config file specifies the [index template](docs-content://manage-data/data-store/templates.md) to use for setting mappings in Elasticsearch. If template loading is enabled (the default), Packetbeat loads the index template automatically after successfully connecting to Elasticsearch.

::::{note}
A connection to Elasticsearch is required to load the index template. If the configured output is not Elasticsearch (or {{ech}}), you must [load the template manually](/reference/packetbeat/packetbeat-template.md#load-template-manually).
::::


You can adjust the following settings to load your own template or overwrite an existing one.

**`setup.template.enabled`**
:   Set to false to disable template loading. If this is set to false, you must [load the template manually](/reference/packetbeat/packetbeat-template.md#load-template-manually).

**`setup.template.name`**
:   The name of the template. The default is `packetbeat`. The Packetbeat version is always appended to the given name, so the final name is `packetbeat-%{[agent.version]}`.

**`setup.template.pattern`**
:   The template pattern to apply to the default index settings. The default pattern is `packetbeat`. The Packetbeat version is always included in the pattern, so the final pattern is `packetbeat-%{[agent.version]}`.

    Example:

    ```yaml
    setup.template.name: "packetbeat"
    setup.template.pattern: "packetbeat"
    ```


**`setup.template.fields`**
:   The path to the YAML file describing the fields. The default is `fields.yml`. If a relative path is set, it is considered relative to the config path. See the [Directory layout](/reference/packetbeat/directory-layout.md) section for details.

**`setup.template.overwrite`**
:   A boolean that specifies whether to overwrite the existing template. The default is false. Do not enable this option if you start more than one instance of Packetbeat at the same time. It can overload {{es}} by sending too many template update requests.

**`setup.template.settings`**
:   A dictionary of settings to place into the `settings.index` dictionary of the Elasticsearch template. For more details about the available Elasticsearch mapping options, please see the Elasticsearch [mapping reference](docs-content://manage-data/data-store/mapping.md).

    Example:

    ```yaml
    setup.template.name: "packetbeat"
    setup.template.fields: "fields.yml"
    setup.template.overwrite: false
    setup.template.settings:
      index.number_of_shards: 1
      index.number_of_replicas: 1
    ```


**`setup.template.settings._source`**
:   A dictionary of settings for the `_source` field. For the available settings, please see the Elasticsearch [reference](elasticsearch://reference/elasticsearch/mapping-reference/mapping-source-field.md).

    Example:

    ```yaml
    setup.template.name: "packetbeat"
    setup.template.fields: "fields.yml"
    setup.template.overwrite: false
    setup.template.settings:
      _source.enabled: false
    ```


**`setup.template.append_fields`**
:   A list of fields to be added to the template and {{kib}} index pattern. This setting adds new fields. It does not overwrite or change existing fields.

    This setting is useful when your data contains fields that Packetbeat doesn’t know about in advance.

    If `append_fields` is specified along with `overwrite: true`, Packetbeat overwrites the existing template and applies the new template when creating new indices. Existing indices are not affected. If you’re running multiple instances of Packetbeat with different `append_fields` settings, the last one writing the template takes precedence.

    Any changes to this setting also affect the {{kib}} index pattern.

    Example config:

    ```yaml
    setup.template.overwrite: true
    setup.template.append_fields:
    - name: test.name
      type: keyword
    - name: test.hostname
      type: long
    ```


**`setup.template.json.enabled`**
:   Set to `true` to load a JSON-based template file. Specify the path to your {{es}} index template file and set the name of the template.

    ```yaml
    setup.template.json.enabled: true
    setup.template.json.path: "template.json"
    setup.template.json.name: "template-name"
    setup.template.json.data_stream: false
    ```


::::{note}
If the JSON template is used, the `fields.yml` is skipped for the template generation.
::::


::::{note}
If the JSON template is a data stream, set `setup.template.json.data_stream`.
::::


