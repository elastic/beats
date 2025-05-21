---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/heartbeat/current/configuring-ingest-node.html
---

# Parse data using an ingest pipeline [configuring-ingest-node]

When you use {{es}} for output, you can configure Heartbeat to use an [ingest pipeline](docs-content://manage-data/ingest/transform-enrich/ingest-pipelines.md) to pre-process documents before the actual indexing takes place in {{es}}. An ingest pipeline is a convenient processing option when you want to do some extra processing on your data, but you do not require the full power of {{ls}}. For example, you can create an ingest pipeline in {{es}} that consists of one processor that removes a field in a document followed by another processor that renames a field.

After defining the pipeline in {{es}}, you simply configure Heartbeat to use the pipeline. To configure Heartbeat, you specify the pipeline ID in the `parameters` option under `elasticsearch` in the `heartbeat.yml` file:

```yaml
output.elasticsearch:
  hosts: ["localhost:9200"]
  pipeline: my_pipeline_id
```

For example, let’s say that you’ve defined the following pipeline in a file named `pipeline.json`:

```json
{
    "description": "Test pipeline",
    "processors": [
        {
            "lowercase": {
                "field": "agent.name"
            }
        }
    ]
}
```

To add the pipeline in {{es}}, you would run:

```shell
curl -H 'Content-Type: application/json' -XPUT 'http://localhost:9200/_ingest/pipeline/test-pipeline' -d@pipeline.json
```

Then in the `heartbeat.yml` file, you would specify:

```yaml
output.elasticsearch:
  hosts: ["localhost:9200"]
  pipeline: "test-pipeline"
```

When you run Heartbeat, the value of `agent.name` is converted to lowercase before indexing.

For more information about defining a pre-processing pipeline, see the [ingest pipeline](docs-content://manage-data/ingest/transform-enrich/ingest-pipelines.md) documentation.

