---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/devguide/current/contributing-docs.html
applies_to:
  stack: ga 9.0
---

# Contributing to the docs

The Beats documentation is written in Markdown and is built using [elastic/docs-builder](https://github.com/elastic/docs-builder). Most Markdown files should be edited directly, but some Markdown files are generated.

## Generated docs [generated-docs]

After updating `docs.md` files in `_meta` directories, you must run the doc collector scripts to regenerate the docs.

Make sure you [set up your Beats development environment](./index.md#setting-up-dev-environment) and use the correct Go version. The Go version is listed in the `version.asciidoc` file for the branch you want to update.

To run the docs collector scripts, change to the beats directory and run:

`make update`

::::{warning}
The `make update` command overwrites files in the `docs` directories **without warning**. If you accidentally update a generated file and run `make update`, your changes will be overwritten.
::::

To format your files, you might also need to run this command:

`make fmt`

The make command calls the following scripts to generate the docs:

[auditbeat/scripts/docs_collector.py](https://github.com/elastic/beats/blob/main/auditbeat/scripts/docs_collector.py) generates:

* `docs/reference/auditbeat/auditbeat-modules.md`
* `docs/reference/auditbeat/auditbeat-module-*.md`

[filebeat/scripts/docs_collector.py](https://github.com/elastic/beats/blob/main/filebeat/scripts/docs_collector.py) generates:

* `docs/reference/filebeat/filebeat-modules.md`
* `docs/reference/filebeat/filebeat-module-*.md`

[metricbeat/scripts/mage/docs_collector.go](https://github.com/elastic/beats/blob/main/metricbeat/scripts/mage/docs_collector.go) generates:

* `docs/reference/metricbeat/metricbeat-modules.md`
* `docs/reference/metricbeat/metricbeat-module-*.md`

[libbeat/scripts/generate_fields_docs.py](https://github.com/elastic/beats/blob/main/libbeat/scripts/generate_fields_docs.py) generates:

* `docs/reference/auditbeat/exported-fields.md`
* `docs/reference/filebeat/exported-fields.md`
* `docs/reference/heartbeat/exported-fields.md`
* `docs/reference/metricbeat/exported-fields.md`
* `docs/reference/packetbeat/exported-fields.md`
* `docs/reference/winlogbeat/exported-fields.md`
