---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/load-ingest-pipelines.html
applies_to:
  stack: ga
  serverless: ga
---

# Load ingest pipelines [load-ingest-pipelines]

Packetbeat modules are implemented using {{es}} ingest node pipelines.  The events receive their transformations within {{es}}.  The ingest node pipelines must be loaded into {{es}}.  This can happen one of several ways.


## On connection to {{es}} [packetbeat-load-pipeline-auto]

Packetbeat will send ingest pipelines automatically to {{es}} if the {{es}} output is enabled.

Make sure the user specified in `packetbeat.yml` is [authorized to set up Packetbeat](/reference/packetbeat/privileges-to-setup-beats.md).

{applies_to}`stack: ga 9.6` To turn off automatic pipeline loading during normal publishing, set:

```yaml
setup.pipelines.enabled: false
```

Use this setting only when the required pipelines have already been loaded or are managed separately.

If Packetbeat is sending events to {{ls}} or another output you need to load the ingest pipelines with the `setup` command or manually.


## Manually install pipelines [packetbeat-load-pipeline-manual]

Pipelines can be loaded them into {{es}} with the `_ingest/pipeline` REST API call. The user making the REST API call will need to have the `ingest_admin` role assigned to them.

