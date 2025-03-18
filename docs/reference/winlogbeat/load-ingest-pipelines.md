---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/winlogbeat/current/load-ingest-pipelines.html
---

# Load ingest pipelines [load-ingest-pipelines]

Winlogbeat modules are implemented using {{es}} ingest node pipelines.  The events receive their transformations within {{es}}.  The ingest node pipelines must be loaded into {{es}}.  This can happen one of several ways.


## On connection to {{es}} [winlogbeat-load-pipeline-auto]

Winlogbeat will send ingest pipelines automatically to {{es}} if the {{es}} output is enabled.

Make sure the user specified in `winlogbeat.yml` is [authorized to set up Winlogbeat](/reference/winlogbeat/privileges-to-setup-beats.md).

If Winlogbeat is sending events to {{ls}} or another output you need to load the ingest pipelines with the `setup` command or manually.


## setup command [winlogbeat-load-pipeline-setup]

On a machine that has Winlogbeat installed and has {{es}} configured as the output, run the `setup` command with the `--pipelines` option specified.  For example, the following command loads the ingest pipelines:

```sh
PS > .\winlogbeat.exe setup --pipelines
```

Make sure the user specified in `winlogbeat.yml` is [authorized to set up Winlogbeat](/reference/winlogbeat/privileges-to-setup-beats.md).


## Manually install pipelines [winlogbeat-load-pipeline-manual]

On a machine that has Winlogbeat installed export the the pipelines to disk. This can be done with the `export` command with `pipelines` option specified.  For example, the following command exports the ingest pipelines:

```sh
PS> .\winlogbeat.exe export pipelines --es.version=7.16.0
```

Once the pipelines have been exported you can load them into {{es}} with the `_ingest/pipeline` REST API call.  The user making the REST API call will need to have the `ingest_admin` role assigned to them.

