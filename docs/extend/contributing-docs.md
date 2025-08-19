---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/devguide/current/contributing-docs.html
applies_to:
  stack: ga 9.0
---

# Contributing to the docs

The Beats documentation is written in Markdown and is built using [elastic/docs-builder](https://github.com/elastic/docs-builder).

## Cumulative docs [cumulative-docs]

Starting with Elastic Stack version 9.0.0 we no longer publish a new documentation set for every minor release.
This means that a single page should stay valid over time and use version-related tags to illustrate how the product has evolved.

For information on labeling manually maintained content with product lifecycle and versioning information, refer to [Write cumulative documentation](https://elastic.github.io/docs-builder/contribute/cumulative-docs/).

For generated content, read more below in [Update `fields.yml`](#update-fields).

## Generated docs [generated-docs]

Many Markdown files in the Beats repo should be edited directly, but some are generated including:

* Exported fields (for example, [AWS fields](https://www.elastic.co/docs/reference/beats/metricbeat/exported-fields-aws))
* Module docs (for example, [AWS module](https://www.elastic.co/docs/reference/beats/metricbeat/metricbeat-module-aws))
* Metricset and dataset docs (for example, [AWS billing metricset](https://www.elastic.co/docs/reference/beats/metricbeat/metricbeat-metricset-aws-billing))

:::{tip}
Every Markdown file that is generated includes a code comment at the top of the content that states `% This file is generated!`.
:::

### Update `fields.yml` [update-fields]

The `fields.yml` files in `_meta` directories across individual beats contain descriptions of fields available in the module, dataset, fileset, or metricset. Here are some tips for optimizing `fields.yml` for generating docs:

* The `title` is used as a page title in the docs, so it’s best to capitalize it.
* The `description` at all levels should be written in full sentences and include punctuation.
* The `version` at all levels is used to label docs with product lifecycle and version-related
  information that illustrates how the product has evolved over time, which is important to
  [writing docs cumulatively](#cumulative-docs). Some tips for using `version`:

  * Supported product lifecycles include `preview`, `beta`, `ga`, and `deprecated`.
  * Multiple product lifecycles can exist for the same module or field to illustrate how it changed over time.
  * The version number can be in major, minor, or patch format, but the resulting rendered label will always resolve to the patch level.
  * Here's an example of `version` for a field that went through all product lifecycles:
    ```yaml
    version:
      preview: 9.0.0
      beta: 9.1.0
      ga: 9.2.0
      deprecated: 9.3.0
    ```

### Update `docs.md`

The `docs.md` files in `_meta` directories is used for generated module documentation.

### Generate the docs

After updating `fields.md` and `docs.md` files in `_meta` directories,
you must run the doc collector scripts to regenerate the docs:

1. Make sure you [set up your Beats development environment](./index.md#setting-up-dev-environment)
  and use the correct Go version.
    * The Go version is listed in the `version.asciidoc` file for the branch you want to update.
1. Change to the beats directory.
1. Run `make update` to run the docs collector scripts.

    ::::{warning}
    The `make update` command overwrites files in the `docs` directories **without warning**. If you accidentally update a generated file and run `make update`, your changes will be overwritten.
    ::::

    The `make` command calls the following scripts to generate the docs:

    * [**`auditbeat/scripts/docs_collector.py`**](https://github.com/elastic/beats/blob/main/auditbeat/scripts/docs_collector.py) generates:
        * `docs/reference/auditbeat/auditbeat-modules.md`
        * `docs/reference/auditbeat/auditbeat-module-*.md`
    * [**`filebeat/scripts/docs_collector.py`**](https://github.com/elastic/beats/blob/main/filebeat/scripts/docs_collector.py) generates:
      * `docs/reference/filebeat/filebeat-modules.md`
      * `docs/reference/filebeat/filebeat-module-*.md`
    * [**`metricbeat/scripts/mage/docs_collector.go`**](https://github.com/elastic/beats/blob/main/metricbeat/scripts/mage/docs_collector.go) generates:
      * `docs/reference/metricbeat/metricbeat-modules.md`
      * `docs/reference/metricbeat/metricbeat-module-*.md`
    * [**`libbeat/scripts/generate_fields_docs.py`**](https://github.com/elastic/beats/blob/main/libbeat/scripts/generate_fields_docs.py) generates:
      * `docs/reference/auditbeat/exported-fields.md`
      * `docs/reference/filebeat/exported-fields.md`
      * `docs/reference/heartbeat/exported-fields.md`
      * `docs/reference/metricbeat/exported-fields.md`
      * `docs/reference/packetbeat/exported-fields.md`
      * `docs/reference/winlogbeat/exported-fields.md`

1. (Optional) To format your files, you might also need to run this command:
    ```sh
    make fmt
    ```


