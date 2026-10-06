---
name: beats-docs
description: >
  Beats-specific documentation rules. Use with the docs-authoring skill when a
  change edits docs/ in the beats repository, or edits a fields.yml or docs.md
  under a beat's _meta directory that generates a page there. Covers which files
  are published, which are generated, and where a version label comes from.
---

# Beats docs

Read `docs-authoring` first. This file adds only what is specific to the beats repository. The repo's own guide is [docs/extend/contributing-docs.md](https://github.com/elastic/beats/blob/main/docs/extend/contributing-docs.md); follow it where it is more specific than this skill.

## What is published

`docs/` is the docset (`docs/docset.yml`, product `beats`). It is Markdown, built by docs-builder.

Production does not build `main`. In docs-builder `config/assembler.yml`, `beats` uses the stack refs: `current` is the version branch production publishes (`9.5` today), and `next` and `edge` are `main`. A change is on the live site only once it is on the `current` branch. Open the pull request against `main` and backport it to that branch when it should be live now. Read `current` from `assembler.yml` rather than trusting the version in this sentence.

- `docs/reference/<beat>/` is the reference for that Beat: `auditbeat`, `filebeat`, `heartbeat`, `metricbeat`, `packetbeat`, `winlogbeat`, `libbeat`.
- `docs/extend/` is the developer guide, including how to contribute to the docs.
- `docs/release-notes/` is the breaking changes, deprecations, and known issues. These pages are hand-written. A changelog fragment is not one of them.
- `docs/reference/_snippets/` holds text included by more than one page. Change a snippet only when every page that includes it should change.

Product names that have a substitution in `docs/docset.yml` are written as `{{filebeat}}`, `{{agent}}`, `{{kib}}`, and the rest of that list. Match the page you are editing: do not spell out a name that page writes as a substitution, and do not introduce one it does not use.

## Generated pages

A page whose body starts with `% This file is generated!` is generated. The comment names the script. Edit the source, then regenerate:

- Field descriptions and the `version` that becomes the page's `applies_to` live in the `fields.yml` under that module's `_meta` directory.
- Module prose lives in the `docs.md` under `_meta`.
- `make update` from the repository root regenerates the pages. It overwrites generated files with no prompt, so do not hand-edit one and expect the edit to survive.

`title` in `fields.yml` is the page title, so capitalize it. `description` is full sentences with punctuation.

## Version labels on generated content

For a generated page, set the version in `fields.yml`, not by editing `applies_to` in the Markdown. `version` takes `preview`, `beta`, `ga`, and `deprecated`, and a field can carry more than one to show how it changed:

```yaml
version:
  preview: 9.0.0
  ga: 9.2.0
```

For a hand-written page, set `applies_to` in the page, following `docs-authoring`.

## Changelog

A docs-only change does not get a changelog fragment; add the `skip-changelog` label. A user-visible behavior change still gets a fragment, as the Changelog section of `AGENTS.md` describes. The fragment and the doc page are separate: write both, and do not put the changelog text into the page.
