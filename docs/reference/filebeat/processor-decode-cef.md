---
navigation_title: "decode_cef"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/processor-decode-cef.html
---

# Decode CEF [processor-decode-cef]


The `decode_cef` processor decodes Common Event Format (CEF) messages. It follows the specification defined in [ *Micro Focus Security ArcSight Common Event Format, Version 25*](https://archive.org/download/commoneventformatv25/CommonEventFormatV25.pdf). This processor is available in Filebeat. This is an example CEF message.

`CEF:0|SomeVendor|TheProduct|1.0|100|connection to malware C2 successfully stopped|10|src=192.0.2.10 dst=203.0.113.2 spt=31224`

Any content that precedes `CEF:` is ignored. This allows the processor to directly parse CEF content from messages that contain syslog headers.

Below is an example configuration that decodes the `message` field as CEF after renaming it to `event.original`. It is best to rename `message` to `event.original` because the decoded CEF data contains its own `message` field.

```yaml
processors:
  - rename:
      fields:
        - {from: "message", to: "event.original"}
  - decode_cef:
      field: event.original
```

The `decode_cef` processor has the following configuration settings.

| Name | Required | Default | Description |
| --- | --- | --- | --- |
| `field` | no | message | Source field containing the CEF message to be parsed. |
| `target_field` | no | cef | Target field where the parsed CEF object will be written. |
| `ecs` | no | true | Generate Elastic Common Schema (ECS) fields from the CEF data. Certain CEF header and extension values will be used to populate ECS fields. |
| `timezone` | no | UTC | IANA time zone name (e.g. `America/New_York`) or fixed time offset (e.g. `+0200`) to use when parsing times that do not contain a time zone. `Local` may be specified to use the machine’s local time zone. |  |
| `ignore_missing` | no | false | Ignore errors when the source field is missing. |
| `ignore_failure` | no | false | Ignore failures when the source field does not contain a CEF message. |  |
| `ignore_empty_values` | no | false | Ignore CEF extensions with empty values (e.g. `spt= type=1`) |
| `id` | no |  | An identifier for this processor instance. Useful for debugging. |

