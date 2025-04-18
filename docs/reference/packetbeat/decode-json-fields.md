---
navigation_title: "decode_json_fields"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/decode-json-fields.html
---

# Decode JSON fields [decode-json-fields]


The `decode_json_fields` processor decodes fields containing JSON strings and replaces the strings with valid JSON objects.

```yaml
processors:
  - decode_json_fields:
      fields: ["field1", "field2", ...]
      process_array: false
      max_depth: 1
      target: ""
      overwrite_keys: false
      add_error_key: true
```

The `decode_json_fields` processor has the following configuration settings:

`fields`
:   The fields containing JSON strings to decode.

`process_array`
:   (Optional) A Boolean value that specifies whether to process arrays. The default is `false`.

`max_depth`
:   (Optional) The maximum parsing depth. A value of `1`  will decode the JSON objects in fields indicated in `fields`, a value of `2` will also decode the objects embedded in the fields of these parsed documents. The default is `1`.

`target`
:   (Optional) The field under which the decoded JSON will be written. By default, the decoded JSON object replaces the string field from which it was read. To merge the decoded JSON fields into the root of the event, specify `target` with an empty string (`target: ""`). Note that the `null` value (`target:`) is treated as if the field was not set.

`overwrite_keys`
:   (Optional) A Boolean value that specifies whether existing keys in the event are overwritten by keys from the decoded JSON object. The default value is `false`.

`expand_keys`
:   (Optional) A Boolean value that specifies whether keys in the decoded JSON should be recursively de-dotted and expanded into a hierarchical object structure. For example, `{"a.b.c": 123}` would be expanded into `{"a":{"b":{"c":123}}}`.

`add_error_key`
:   (Optional) If set to `true` and an error occurs while decoding JSON keys, the `error` field will become a part of the event with the error message. If set to `false`, there will not be any error in the event’s field. The default value is `false`.

`document_id`
:   (Optional) JSON key that’s used as the document ID. If configured, the field will be removed from the original JSON document and stored in `@metadata._id`

