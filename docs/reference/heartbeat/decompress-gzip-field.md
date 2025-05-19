---
navigation_title: "decompress_gzip_field"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/heartbeat/current/decompress-gzip-field.html
---

# Decompress gzip fields [decompress-gzip-field]


The `decompress_gzip_field` processor specifies a field to gzip decompress. The `field` key contains a `from: old-key` and a `to: new-key` pair. `from` is the origin and `to` the target name of the field.

To overwrite fields either first rename the target field or use the `drop_fields` processor to drop the field and then decompress the field.

```yaml
processors:
  - decompress_gzip_field:
      field:
        from: "field1"
        to: "field2"
      ignore_missing: false
      fail_on_error: true
```

In the example above:
- field1 is decoded in field2

The `decompress_gzip_field` processor has the following configuration settings:

`ignore_missing`
:   (Optional) If set to true, no error is logged in case a key which should be decompressed is missing. Default is `false`.

`fail_on_error`
:   (Optional) If set to true, in case of an error the decompression of fields is stopped and the original event is returned. If set to false, decompression continues also if an error happened during decoding. Default is `true`.

See [Conditions](/reference/heartbeat/defining-processors.md#conditions) for a list of supported conditions.

