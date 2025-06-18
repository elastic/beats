---
navigation_title: "now"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/winlogbeat/current/now.html
---

# Now [now]


The `now` processor sets the current timestamp to the specified field of the event. The `now` processor will overwrite the target field if it already exists.

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


