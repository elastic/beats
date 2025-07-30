---
navigation_title: "now"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/winlogbeat/current/now.html
applies_to:
  stack: ga 9.1.0
---

# Now [now]

The `now` processor sets the current timestamp to the specified field of the event. The `now` processor will overwrite the target field if it already exists.

The specified target field can be a nested field. The `now` processor will throw an error and leave the original event unchanged if the target nested field has an existing non-object as a parent.

`field`
:   The target field.

For example, this configuration:

```yaml
processors:
  - now:
      field: event.created
```

Results in the following event:

```json
{
  "event": {
    "created": "2025-04-08T12:00:00.000000042Z"
  }
}
```

The event will be unchanged if the target nested field has an existing non-object as a parent, given:
```yaml
processors:
  - now:
      field: event.created
```

The following event will not be altered:

```json
{
  "event": "foo"
}
```
