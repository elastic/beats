---
navigation_title: "replace"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/heartbeat/current/replace-fields.html
---

# Replace fields from events [replace-fields]


The `replace` processor takes a list of fields to search for a matching value and replaces the matching value with a specified string.

The `replace` processor cannot be used to create a completely new value.

::::{tip}
You can use this processor to truncate a field value or replace it with a new string value. You can also use this processor to mask PII information.
::::



## Example [_example]

The following example changes the path from `/usr/bin` to `/usr/local/bin`:

```yaml
  - replace:
      fields:
        - field: "file.path"
          pattern: "/usr/"
          replacement: "/usr/local/"
      ignore_missing: false
      fail_on_error: true
```


## Configuration settings [_configuration_settings]

| Name | Required | Default | Description |
| --- | --- | --- | --- |
| `fields` | Yes |  | List of one or more items. Each item contains a `field: field-name`, `pattern: regex-pattern`, and `replacement: replacement-string`, where:<br><br>* `field` is the original field name. You can use the `@metadata.` prefix in this field to replace values in the event metadata instead of event fields.<br>* `pattern` is the regex pattern to match the field’s value<br>* `replacement` is the replacement string to use to update the field’s value<br> |
| `ignore_missing` | No | `false` | Whether to ignore missing fields. If `true`, no error is logged if the specified field is missing. |
| `fail_on_error` | No | `true` | Whether to fail replacement of field values if an error occurs.If `true` and there’s an error, the replacement of field values is stopped, and the original event is returned.If `false`, replacement continues even if an error occurs during replacement. |

See [Conditions](/reference/heartbeat/defining-processors.md#conditions) for a list of supported conditions.

