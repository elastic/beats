---
navigation_title: "detect_mime_type"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/heartbeat/current/detect-mime-type.html
---

# Detect mime type [detect-mime-type]


The `detect_mime_type` processor attempts to detect a mime type for a field that contains a given stream of bytes. The `field` key contains the field used as the data source and the `target` key contains the field to populate with the detected type. It’s supported to use `@metadata.` prefix for `target` and set the value in the event metadata instead of fields.

```yaml
processors:
  - detect_mime_type:
      field: http.request.body.content
      target: http.request.mime_type
```

In the example above: - http.request.body.content is used as the source and http.request.mime_type is set to the detected mime type

See [Conditions](/reference/heartbeat/defining-processors.md#conditions) for a list of supported conditions.

