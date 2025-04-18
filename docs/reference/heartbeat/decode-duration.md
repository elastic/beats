---
navigation_title: "decode_duration"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/heartbeat/current/decode-duration.html
---

# Decode duration [decode-duration]


The `decode_duration` processor decodes a Go-style duration string into a specific `format`.

For more information about the Go `time.Duration` string style, refer to the [Go documentation](https://pkg.go.dev/time#Duration).

| Name | Required | Default | Description |  |
| --- | --- | --- | --- | --- |
| `field` | yes |  | Which field of event needs to be decoded as `time.Duration` |  |
| `format` | yes | `milliseconds` | Supported formats: `milliseconds`/`seconds`/`minutes`/`hours` |  |

```yaml
processors:
  - decode_duration:
      field: "app.rpc.cost"
      format: "milliseconds"
```

