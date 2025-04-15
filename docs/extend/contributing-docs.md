---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/devguide/current/contributing-docs.html
applies_to:
  stack: discontinued 8.18
---

# Contributing to the docs [contributing-docs]

The Beats documentation follows the tagging guidelines described in the [Docs HOWTO](https://github.com/elastic/docs/blob/master/README.asciidoc). However it extends these capabilities in a couple ways:

* The documentation makes extensive use of [AsciiDoc conditionals](https://docs.asciidoctor.org/asciidoc/latest/directives/conditionals/) to provide content that is reused across multiple books. This means that there might not be a single source file for each published HTML page. Some files are shared across multiple books, either as complete pages or snippets. For more details, refer to [Where to find the Beats docs source](#where-to-find-files).
* The documentation includes some files that are generated from YAML source or pieced together from content that lives in `_meta` directories under the code (for example, the module and exported fields documentation). For more details, refer to [Generated docs](#generated-docs).


## Where to find the Beats docs source [where-to-find-files]

Because the Beats documentation makes use of shared content, doc generation scripts, and componentization, the source files are located in several places:

| Documentation | Location of source files |
| --- | --- |
| Main docs for the Beat, including index files | `<beatname>/docs` |
| Shared docs and Beats Platform Reference | `libbeat/docs` |
| Processor docs | `docs` folders under processors in `libbeat/processors/`,`x-pack/<beatname>/processors/`, and `x-pack/libbeat/processors/` |
| Output docs | `docs` folders under outputs in `libbeat/outputs/` |
| Module docs | `_meta` folders under modules and datasets in `libbeat/module/`,`<beatname>/module/`, and `x-pack/<beatname>/module/` |

The [conf.yaml](https://github.com/elastic/docs/blob/master/conf.yaml) file in the `docs` repo shows all the resources used to build each book. This file is used to drive the classic docs build and is the source of truth for file locations.

::::{tip}
If you can’t find the source for a page you want to update, go to the published page at www.elastic.co and click the Edit link to navigate to the source.
::::


The Beats documentation build also has dependencies on the following files in the [docs](https://github.com/elastic/docs) repo:

* `shared/versions/stack/<version>.asciidoc`
* `shared/attributes.asciidoc`


## Generated docs [generated-docs]

After updating `docs.asciidoc` files in `_meta` directories, you must run the doc collector scripts to regenerate the docs.

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

* `auditbeat/docs/modules_list.asciidoc`
* `auditbeat/docs/modules/*.asciidoc`

[filebeat/scripts/docs_collector.py](https://github.com/elastic/beats/blob/main/filebeat/scripts/docs_collector.py) generates:

* `filebeat/docs/modules_list.asciidoc`
* `filebeat/docs/modules/*.asciidoc`

[metricbeat/scripts/mage/docs_collector.go](https://github.com/elastic/beats/blob/main/metricbeat/scripts/mage/docs_collector.go) generates:

* `metricbeat/docs/modules_list.asciidoc`
* `metricbeat/docs/modules/*.asciidoc`

[libbeat/scripts/generate_fields_docs.py](https://github.com/elastic/beats/blob/main/libbeat/scripts/generate_fields_docs.py) generates

* `auditbeat/docs/fields.asciidoc`
* `filebeat/docs/fields.asciidoc`
* `heartbeat/docs/fields.asciidoc`
* `metricbeat/docs/fields.asciidoc`
* `packetbeat/docs/fields.asciidoc`
* `winlogbeat/docs/fields.asciidoc`
