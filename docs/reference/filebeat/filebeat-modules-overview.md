---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-modules-overview.html
---

# Modules

Filebeat modules simplify the collection, parsing, and visualization of common log formats.

A typical module (say, for the Nginx logs) is composed of one or more filesets (in the case of Nginx, `access` and `error`). A fileset contains the following:

* Filebeat input configurations, which contain the default paths where to look for the log files. These default paths depend on the operating system. The Filebeat configuration is also responsible with stitching together multiline events when needed.
* {{es}} [ingest pipeline](docs-content://manage-data/ingest/transform-enrich/ingest-pipelines.md) definition, which is used to parse the log lines.
* Fields definitions, which are used to configure {{es}} with the correct types for each field. They also contain short descriptions for each of the fields.
* Sample {{kib}} dashboards, when available, that can be used to visualize the log files.

Filebeat automatically adjusts these configurations based on your environment and loads them to the respective {{stack}} components.

If a module configuration is updated, the {{es}} ingest pipeline definition is not reloaded automatically. To reload the ingest pipeline, set `filebeat.overwrite_pipelines: true` and manually [load the ingest pipelines](/reference/filebeat/load-ingest-pipelines.md).


## Get started [_get_started]

To learn how to configure and run Filebeat modules:

* Get started by reading [Quick start: installation and configuration](/reference/filebeat/filebeat-installation-configuration.md).
* Dive into the documentation for each [module](/reference/filebeat/filebeat-modules.md).

