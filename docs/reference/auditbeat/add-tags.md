---
meta_title: "Add tags to events with add_tags in Auditbeat"
description: "Use the add_tags processor in Auditbeat to add tags to the tags field of each event, or to a custom target field."
navigation_title: "add_tags"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/auditbeat/current/add-tags.html
applies_to:
  stack: ga
  serverless: ga
---

# Add tags in Auditbeat [add-tags]


The `add_tags` processor adds tags to a list of tags. If the target field already exists, the tags are appended to the existing list of tags.

`tags`
:   List of tags to add.

`target`
:   (Optional) Field the tags will be added to. Defaults to `tags`. Setting tags in `@metadata` is not supported.

For example, this configuration:

```yaml
processors:
  - add_tags:
      tags: [web, production]
      target: "environment"
```

Adds the environment field to every event:

```json
{
  "environment": ["web", "production"]
}
```

