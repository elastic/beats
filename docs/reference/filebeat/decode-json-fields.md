---
navigation_title: "decode_json_fields"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/decode-json-fields.html
applies_to:
  stack: ga
  serverless: ga
---

# Decode JSON fields [decode-json-fields]


The `decode_json_fields` processor decodes fields containing JSON strings and replaces the strings with valid JSON objects.

```yaml
processors:
  - decode_json_fields:
      fields: ["field1", "field2", ...]
      process_array: false
      max_depth: 1
      max_object_depth: 0
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
:   (Optional) The maximum number of times JSON is decoded. A value of `1` decodes the JSON string in each field listed in `fields`. A value of `2` also decodes strings inside the decoded documents that contain JSON, for example `{"log": "{\"level\": \"info\"}"}`. Objects that are already nested in the JSON, rather than encoded as strings, are always decoded in full regardless of this setting. To limit how deeply nested objects are kept as fields, use `max_object_depth`. The default is `1`.

`max_object_depth` {applies_to}`stack: ga 9.6`
:   (Optional) The maximum nesting level of objects and arrays kept as event fields after decoding. Objects and arrays nested deeper than this are not expanded into fields; they are stored as JSON-encoded strings instead. The decoded document is at level `1`, the objects it contains are at level `2`, and so on. Arrays do not add a level. This limits the number of fields created from deeply nested input, which helps prevent mapping explosions in Elasticsearch. The default is `0`, which means no limit. The stored strings are the original JSON text from the input, unchanged. Objects and arrays deeper than the limit are not decoded, so `document_id` must name a key within the limit. Keys containing dots are expanded after decoding when `expand_keys` is `true`, so they are not counted. Refer to [the comparison](#decode-json-fields-depth) for an example.

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

## Comparing `max_depth` and `max_object_depth` [decode-json-fields-depth]

`max_depth` controls how many times JSON is decoded. It affects values that are JSON-encoded strings and has no effect on objects that are already nested. `max_object_depth` controls how deeply nested objects are kept as fields. It affects nested objects and arrays and has no effect on JSON-encoded strings.

The following table shows the result for the same input with different settings. The `log` field holds a JSON-encoded string, and `a` holds nested objects.

```json
{"log": "{\"level\": \"info\"}", "a": {"b": {"c": 1}}}
```

| Settings | `log` | `a` |
|---|---|---|
| Default | `"{\"level\": \"info\"}"` (a string) | `{"b": {"c": 1}}` |
| `max_depth: 2` | `{"level": "info"}` (decoded) | `{"b": {"c": 1}}` |
| `max_object_depth: 2` | `"{\"level\": \"info\"}"` (a string) | `{"b": "{\"c\": 1}"}` |

With `max_depth: 2`, only `log` changes, because it is the only JSON-encoded string. With `max_object_depth: 2`, only `a` changes, because `b` is nested deeper than level 2 and is stored as its original JSON text.
