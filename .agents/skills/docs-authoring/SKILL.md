---
name: docs-authoring
description: >
  Write or edit user-facing documentation. Use when a change adds or edits a
  published doc page, a snippet, navigation, or the source that generates a page.
  Decides whether a change needs docs, where the content goes, what must be
  verified against the product, and how to scope it with applies_to. Does not
  cover changelog fragments or release-note entries.
metadata:
  source: elastic-docs-authoring
  source_version: "1.0.0"
---

# Documentation

A page is published as written. Write for the person doing the task, and write it as the final text they will read.

Ask when the change does not tell you who it is for or what they can now do. Never invent a UI label, a default, a limit, a permission, or a behavior, and never generate a screenshot. Name the screenshot that is needed and where it goes.

## Whether to write

| The change | What to do |
|---|---|
| Nothing new for the reader to choose, configure, or do | No page change. Say so. |
| A page already says this | Show where, and narrow or drop the edit |
| A page states something that is now false | Fix that page |
| A workflow gained a step or option a reader would otherwise miss | Add it |
| A page is deliberately general and the request adds specifics | Usually not a gap. Say so |
| Something the page documents was removed | Keep the content and scope it. Do not delete it |

"Remove X" describes the product change, not the edit. The docs are cumulative, and readers on versions that still have the feature still need the page. For a GA or deprecated feature in a versioned product, keep the content and mark it `removed <version>` with `applies_to`. Deleting is reasonable only when the feature was only ever a preview or beta, or only ever existed in a product that has no versions. Say which reading you took.

Concluding that nothing needs documenting is a valid result. Report it and stop.

## Where it goes

Start from the assumption that an existing page should absorb the change. Adding a page is the exception.

Search before you add. Search the published docs and the `docs/` tree of this repository, including snippets and pages that are not linked yet. A search that did not run is not evidence that nothing covers the topic, so do not report "nothing covers this yet" unless you looked.

Propose the lightest change that closes the gap, in this order: a sentence in an existing section, a new section, a new page. Pause when you are about to add a page, or when the change touches several pages.

Put each fact in one place. A caveat that belongs on every page belongs on the one page that owns the concept, with the others linking to it. A sentence, admonition, or requirement repeated at the top of every page in a section is that mistake. A shared snippet is not, when its only job is to say which of two similar systems the page documents: no `applies_to` value can say that, and the wording lives once. Check the nearest `_snippets/` directory before writing prose that more than one page needs.

## Verify, then write

List every concrete claim the page will make: each UI string, field name, setting, default, limit, permission, and behavior. Then verify each one against the product source as it is now, not against the issue or the pull request description. A diff shows one change. The current source shows what the reader will meet.

Anything you cannot verify goes back to the author as a question. It does not go in the page.

Because the page covers every supported version, a statement that is true only in the latest code is wrong for most readers. Confirm each claim for the versions the page covers, or scope it.

The product's own lifecycle field outranks the request. A release-note label, a bare version number, and "on by default" all mean something shipped. None of them means generally available. A backport label is not a released version, and neither is a merged pull request. Use the minor version in an `applies_to` badge. When a critical difference really is patch-specific, explain the exact patch in prose.

## Versioning

One page covers every version. Do not copy a page per release. Mark a difference with `applies_to` on the page or on the section that differs.

```yaml
applies_to:
  stack: ga 9.1
  serverless: ga
```

A feature that exists only in some versions gets `applies_to` on that section. When it is removed from a versioned product, keep the section and add the removal, for example `stack: removed 9.1`. When the page is generated, the repository contributing guide says where the version is set. Do not hand-edit `applies_to` in a generated page.

If `docs-applies-to-tagging` is installed, let it choose the values and the placement. Collect the version, lifecycle, and deployment answers and hand them over.

## Write the page

Write what the reader can now do, see, configure, or avoid. The notes you were given describe an implementation. Nothing later will translate them for you, so do it as you draft. Cut anything that does not serve the task, including a detail that is there only because the issue mentioned it.

Order the page for the task: what it is, then how to do it, then the edge cases. Open a new option or mode on the outcome, not the control. For example, use **View documents as JSON**, then write the sentence that names the control. The labels and defaults you verified are a checklist the page must not contradict. They are not an outline.

- Address the reader as you. Do not call them users or customers.
- Present tense. Active voice. One idea per sentence.
- Sentence case for headings. No trailing period on a heading.
- Name the product, feature, page, or setting on first mention. No internal names: packages, functions, struct fields, ticket numbers, or team names.
- Backticks for settings, field names, file names, CLI flags, and values the reader types.
- Quote a UI label only when it would otherwise read as prose. Feature names are capitalized and not quoted.
- US English.

Use the substitutions the repository defines (`{{kib}}`, `{{es}}`, and the product's own) the way the surrounding page does. Do not spell out a name that page writes as a substitution, and do not introduce one it does not use.

## Links and navigation

A new page goes in the `toc.yml` for its section. Navigation is per docset, and some sections nest their toc, so copy the shape of the section you are adding to rather than inventing one.

Use descriptive link text rather than a bare URL or generic call to action. A link to a page in another Elastic docset uses that docset's form (`docs-content://...`, `integration-docs://...`), not `https://www.elastic.co/docs/...`.

A page you moved, renamed, or deleted needs an entry in `redirects.yml`. If `docs-redirects` is installed, hand it the old path and the new one.

A link to a page being added in another repository cannot resolve until that change publishes. Leave the link, name the change it waits on, and do not point it at a placeholder. Every other unresolved link is a defect.

## Generated pages

If a page says it is generated, edit the source it names and regenerate. An edit to the generated file is overwritten on the next run. The repository contributing guide names the source and the command.

## Checks

Apply these rules while you write, not after. Run the repository's docs build and Vale checks. If the `elastic-docs-skills` plugin is installed, run the individual checks that apply for content type, `applies_to`, style, frontmatter, jargon, syntax, and redirects. `docs-draft-feature-docs` is an end-to-end drafting workflow, not a post-edit checker. Do not copy its companion skills into this repository.

Reference: [Elastic style guide](https://www.elastic.co/docs/contribute-docs/style-guide), [cumulative docs](https://www.elastic.co/docs/contribute-docs/how-to/cumulative-docs), [`applies_to` reference](https://www.elastic.co/docs/contribute-docs/how-to/cumulative-docs/reference), [content types](https://www.elastic.co/docs/contribute-docs/content-types).
