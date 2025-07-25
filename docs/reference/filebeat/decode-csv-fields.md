---
navigation_title: "decode_csv_fields"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/decode-csv-fields.html
applies_to:
  stack: preview
---

# Decode CSV fields [decode-csv-fields]


::::{warning}
This functionality is in technical preview and may be changed or removed in a future release. Elastic will work to fix any issues, but features in technical preview are not subject to the support SLA of official GA features.
::::


The `decode_csv_fields` processor decodes fields containing records in comma-separated format (CSV). It will output the values as an array of strings. This processor is available for Filebeat.

```yaml
processors:
  - decode_csv_fields:
      fields:
        message: decoded.csv
      separator: ","
      ignore_missing: false
      overwrite_keys: true
      trim_leading_space: false
      fail_on_error: true
```

The `decode_csv_fields` has the following settings:

`fields`
:   This is a mapping from the source field containing the CSV data to the destination field to which the decoded array will be written.

`separator`
:   (Optional) Character to be used as a column separator. The default is the comma character. For using a TAB character you must set it to "\t".

`ignore_missing`
:   (Optional) Whether to ignore events which lack the source field. The default is `false`, which will fail processing of an event if a field is missing.

`overwrite_keys`
:   Whether the target field is overwritten if it already exists. The default is false, which will fail processing of an event when `target` already exists.

`trim_leading_space`
:   Whether extra space after the separator is trimmed from values. This works even if the separator is also a space. The default is `false`.

`fail_on_error`
:   (Optional) If set to true, in case of an error the changes to the event are reverted, and the original event is returned. If set to `false`, processing continues also if an error happens. Default is `true`.

