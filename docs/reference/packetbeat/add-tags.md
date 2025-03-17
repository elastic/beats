---
navigation_title: "add_tags"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/add-tags.html
---

# Add tags [add-tags]


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

