---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/devguide/current/contributing-docs.html
applies_to:
  stack: ga 9.0
---

# Contributing to the docs

The Beats documentation is Markdown in `docs/`, built by [docs-builder](https://github.com/elastic/docs-builder). `docs/docset.yml` defines the docset.

Beats does not publish production docs from `main`. The branch each environment builds is the `beats` entry in [docs-builder `assembler.yml`](https://github.com/elastic/docs-builder/blob/main/config/assembler.yml): production builds `current`, and the other environments build `next` and `edge`. Read those three refs there. A docs change is on the live site only once it is on the `current` branch. Open the pull request against `main`, as [Contribute to Beats](/extend/index.md#contribution-steps) describes, and backport it to the `current` branch when it should be live now.

* `docs/reference/<beat>/` is the reference for that Beat. `docs/reference/libbeat/` is shared by all of them.
* `docs/extend/` is this developer guide.
* `docs/release-notes/` holds the breaking changes, deprecations, and known issues.
* `docs/reference/_snippets/` holds text included by more than one page. Edit a snippet only when every page that includes it should change.

For wording, follow the [Elastic style guide](https://www.elastic.co/docs/contribute-docs/style-guide). Write what you can now do, see, or configure. Use "you", present tense, and sentence case headings. Put settings, field names, and file names in backticks.

Product names that have a substitution in `docs/docset.yml` are written as `{{filebeat}}`, `{{agent}}`, `{{kib}}`, and the rest of that list. Match the page you are editing.

## Where to add something [where-to-add]

Add to the page that already covers the topic. Add a new page only when no existing page can carry it, and then add it to that section's `toc.yml`.

Look through `docs/` and the published docs before you add a page. Each fact belongs in one place. Link to it from the other pages instead of repeating it.

A page you move, rename, or delete needs an entry in `docs/redirects.yml`.

Link to a page in another Elastic docset with its docset link (`docs-content://...`, `integration-docs://...`), not with an `elastic.co` URL.

## Cumulative docs [cumulative-docs]

Since Elastic Stack 9.0.0, one page stays valid across versions. Mark a version or deployment difference with `applies_to` on the page or on the section that differs, instead of copying the page. Read [Write cumulative documentation](https://www.elastic.co/docs/contribute-docs/how-to/cumulative-docs) and the [`applies_to` reference](https://www.elastic.co/docs/contribute-docs/how-to/cumulative-docs/reference).

When a feature, field, or setting is removed, keep the content and scope it with `applies_to`. Readers on versions that still have it need the page. Delete it only when it was only ever a preview or beta, or only ever existed in a product that has no versions.

For generated content, the version label comes from `fields.yml`. See [Update `fields.yml`](#update-fields).

## Generated docs [generated-docs]

Edit most Markdown files directly. These are generated, and each one says so with `% This file is generated!` at the top of the body:

* Exported fields, for example [AWS fields](/reference/metricbeat/exported-fields-aws.md)
* Module docs, for example the [AWS module](/reference/metricbeat/metricbeat-module-aws.md)
* Metricset and dataset docs, for example the [AWS billing metricset](/reference/metricbeat/metricbeat-metricset-aws-billing.md)

The comment names the script that writes the file. Edit the source, then regenerate. An edit to the generated file is overwritten on the next run.

### Update `fields.yml` [update-fields]

The `fields.yml` files in `_meta` directories describe the fields a module, dataset, fileset, or metricset emits.

* `title` becomes the page title, so capitalize it.
* `description` is full sentences with punctuation.
* `version` becomes the `applies_to` label on the generated page. Use `preview`, `beta`, `ga`, and `deprecated`. A field can carry more than one, to show how it changed:

```yaml
version:
  preview: 9.0.0
  beta: 9.1.0
  ga: 9.2.0
  deprecated: 9.3.0
```

The version can be a major, minor, or patch. The rendered label resolves to the patch.

### Update `docs.md` [update-docs]

The `docs.md` files in `_meta` directories are the source for generated module documentation.

### Generate the docs [generate-the-docs]

After you edit `fields.yml` or `docs.md`, regenerate from the repository root:

1. Use the Go version in `.go-version` at the root of the branch you are updating. See [Setting up your dev environment](./index.md#setting-up-dev-environment).
2. Run `make update`.

::::{warning}
`make update` overwrites generated files in `docs/` with no prompt. A hand edit to a generated file is lost the next time it runs.
::::

Each generated file names the script that writes it in the `% This file is generated!` comment. That comment is the list. It stays right when a script moves, and a list kept here would not.

To format the files afterward, run `make fmt`.
