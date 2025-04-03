---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-template.html
---

# Load the Elasticsearch index template [filebeat-template]

{{es}} uses [index templates](docs-content://manage-data/data-store/templates.md) to define:

* Settings that control the behavior of your data stream and backing indices. The settings include the lifecycle policy used to manage backing indices as they grow and age.
* Mappings that determine how fields are analyzed. Each mapping sets the [{{es}} datatype](elasticsearch://reference/elasticsearch/mapping-reference/field-data-types.md) to use for a specific data field.

The recommended index template file for Filebeat is installed by the Filebeat packages. If you accept the default configuration in the `filebeat.yml` config file, Filebeat loads the template automatically after successfully connecting to {{es}}. If the template already exists, it’s not overwritten unless you configure Filebeat to do so.

::::{note}
A connection to {{es}} is required to load the index template. If the output is not {{es}} (or {{ess}}), you must [load the template manually](#load-template-manually).
::::


This page shows how to change the default template loading behavior to:

* [Load your own index template](#load-custom-template)
* [Overwrite an existing index template](#overwrite-template)
* [Disable automatic index template loading](#disable-template-loading)
* [Load the index template manually](#load-template-manually)

For a full list of template setup options, see [Elasticsearch index template](/reference/filebeat/configuration-template.md).


## Load your own index template [load-custom-template]

To load your own index template, set the following options:

```yaml
setup.template.name: "your_template_name"
setup.template.fields: "path/to/fields.yml"
```

If the template already exists, it’s not overwritten unless you configure Filebeat to do so.

You can load templates for both data streams and indices.


## Overwrite an existing index template [overwrite-template]

::::{warning}
Do not enable this option for more than one instance of Filebeat. If you start multiple instances at the same time, it can overload your {{es}} with too many template update requests.
::::


To overwrite a template that’s already loaded into {{es}}, set:

```yaml
setup.template.overwrite: true
```


## Disable automatic index template loading [disable-template-loading]

You may want to disable automatic template loading if you’re using an output other than {{es}} and need to load the template manually. To disable automatic template loading, set:

```yaml
setup.template.enabled: false
```

If you disable automatic template loading, you must load the index template manually.


## Load the index template manually [load-template-manually]

To load the index template manually, run the [`setup`](/reference/filebeat/command-line-options.md#setup-command) command. A connection to {{es}} is required.  If another output is enabled, you need to temporarily disable that output and enable {{es}} by using the `-E` option. The examples here assume that Logstash output is enabled. You can omit the `-E` flags if {{es}} output is already enabled.

If you are connecting to a secured {{es}} cluster, make sure you’ve configured credentials as described in the [Quick start: installation and configuration](/reference/filebeat/filebeat-installation-configuration.md).

If the host running Filebeat does not have direct connectivity to {{es}}, see [Load the index template manually (alternate method)](#load-template-manually-alternate).

To load the template, use the appropriate command for your system.

**deb and rpm:**

```sh
filebeat setup --index-management -E output.logstash.enabled=false -E 'output.elasticsearch.hosts=["localhost:9200"]'
```

**mac:**

```sh
./filebeat setup --index-management -E output.logstash.enabled=false -E 'output.elasticsearch.hosts=["localhost:9200"]'
```

**linux:**

```sh
./filebeat setup --index-management -E output.logstash.enabled=false -E 'output.elasticsearch.hosts=["localhost:9200"]'
```

**docker:**

```sh subs=true
docker run --rm docker.elastic.co/beats/filebeat:{{stack-version}} setup --index-management -E output.logstash.enabled=false -E 'output.elasticsearch.hosts=["localhost:9200"]'
```

**win:**

Open a PowerShell prompt as an Administrator (right-click the PowerShell icon and select **Run As Administrator**).

From the PowerShell prompt, change to the directory where you installed Filebeat, and run:

```sh
PS > .\filebeat.exe setup --index-management -E output.logstash.enabled=false -E 'output.elasticsearch.hosts=["localhost:9200"]'
```


### Force Kibana to look at newest documents [force-kibana-new]

If you’ve already used Filebeat to index data into {{es}}, the index may contain old documents. After you load the index template, you can delete the old documents from `filebeat-*` to force Kibana to look at the newest documents.

Use this command:

**deb and rpm:**

```sh
curl -XDELETE 'http://localhost:9200/filebeat-*'
```

**mac:**

```sh
curl -XDELETE 'http://localhost:9200/filebeat-*'
```

**linux:**

```sh
curl -XDELETE 'http://localhost:9200/filebeat-*'
```

**win:**

```sh
PS > Invoke-RestMethod -Method Delete "http://localhost:9200/filebeat-*"
```

This command deletes all indices that match the pattern `filebeat`. Before running this command, make sure you want to delete all indices that match the pattern.


## Load the index template manually (alternate method) [load-template-manually-alternate]

If the host running Filebeat does not have direct connectivity to {{es}}, you can export the index template to a file, move it to a machine that does have connectivity, and then install the template manually.

To export the index template, run:

**deb and rpm:**

```sh
filebeat export template > filebeat.template.json
```

**mac:**

```sh
./filebeat export template > filebeat.template.json
```

**linux:**

```sh
./filebeat export template > filebeat.template.json
```

**win:**

```sh subs=true
PS > .\filebeat.exe export template --es.version {{stack-version}} | Out-File -Encoding UTF8 filebeat.template.json
```

To install the template, run:

**deb and rpm:**

```sh subs=true
curl -XPUT -H 'Content-Type: application/json' http://localhost:9200/_index_template/filebeat-{{stack-version}} -d@filebeat.template.json
```

**mac:**

```sh subs=true
curl -XPUT -H 'Content-Type: application/json' http://localhost:9200/_index_template/filebeat-{{stack-version}} -d@filebeat.template.json
```

**linux:**

```sh subs=true
curl -XPUT -H 'Content-Type: application/json' http://localhost:9200/_index_template/filebeat-{{stack-version}} -d@filebeat.template.json
```

**win:**

```sh subs=true
PS > Invoke-RestMethod -Method Put -ContentType "application/json" -InFile filebeat.template.json -Uri http://localhost:9200/_index_template/filebeat-{{stack-version}}
```

Once you have loaded the index template, load the data stream as well. If you do not load it, you have to give the publisher user `manage` permission on filebeat-{{stack-version}} index.

**deb and rpm:**

```sh subs=true
curl -XPUT http://localhost:9200/_data_stream/filebeat-{{stack-version}}
```

**mac:**

```sh subs=true
curl -XPUT http://localhost:9200/_data_stream/filebeat-{{stack-version}}
```

**linux:**

```sh subs=true
curl -XPUT http://localhost:9200/_data_stream/filebeat-{{stack-version}}
```

**win:**

```sh subs=true
PS > Invoke-RestMethod -Method Put -Uri http://localhost:9200/_data_stream/filebeat-{{stack-version}}
```

