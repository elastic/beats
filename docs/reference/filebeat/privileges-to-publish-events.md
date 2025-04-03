---
navigation_title: "Create a _publishing_ user"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/privileges-to-publish-events.html
---

# Grant privileges and roles needed for publishing [privileges-to-publish-events]


Users who publish events to {{es}} need to create and write to Filebeat indices. To minimize the privileges required by the writer role, use the [setup role](/reference/filebeat/privileges-to-setup-beats.md) to pre-load dependencies. This section assumes that you’ve run the setup.

When using ILM, turn off the ILM setup check in the Filebeat config file before running Filebeat to publish events:

```yaml
setup.ilm.check_exists: false
```

To grant the required privileges:

1. Create a **writer role**, called something like `filebeat_writer`, that has the following privileges:

    ::::{note}
    The `monitor` cluster privilege and the `create_doc` and `auto_configure` privileges on `filebeat-*` indices are required in every configuration.
    ::::


    | Type | Privilege | Purpose |
    | --- | --- | --- |
    | Cluster | `monitor` | Retrieve cluster details (e.g. version) |
    | Cluster | `read_ilm` | Read the ILM policy when connecting to clusters that support ILM.Not needed when `setup.ilm.check_exists` is `false`. |
    | Cluster | `read_pipeline` | Check for ingest pipelines used by modules. Needed when using modules. |
    | Index | `create_doc` on `filebeat-*` indices | Write events into {{es}} |
    | Index | `auto_configure` on `filebeat-*` indices | Update the datastream mapping. Consider either disabling entirely or adding therule `-{{beat_default_index_prefix}}-*` to the cluster settings[action.auto_create_index](https://www.elastic.co/docs/api/doc/elasticsearch/operation/operation-create)to prevent unwanted indices creations from the agents. |

    Omit any privileges that aren’t relevant in your environment.

2. Assign the **writer role** to users who will index events into {{es}}.

