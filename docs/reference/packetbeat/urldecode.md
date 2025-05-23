---
navigation_title: "urldecode"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/urldecode.html
---

# URL Decode [urldecode]


The `urldecode` processor specifies a list of fields to decode from URL encoded format. Under the `fields` key, each entry contains a `from: source-field` and a `to: target-field` pair, where:

* `from` is the source field name
* `to` is the target field name (defaults to the `from` value)

```yaml
processors:
  - urldecode:
      fields:
        - from: "field1"
          to: "field2"
      ignore_missing: false
      fail_on_error: true
```

In the example above:

* field1 is decoded in field2

The `urldecode` processor has the following configuration settings:

`ignore_missing`
:   (Optional) If set to true, no error is logged in case a key which should be URL-decoded is missing. Default is `false`.

`fail_on_error`
:   (Optional) If set to true, in case of an error the URL-decoding of fields is stopped and the original event is returned. If set to false, decoding continues also if an error happened during decoding. Default is `true`.

See [Conditions](/reference/packetbeat/defining-processors.md#conditions) for a list of supported conditions.

