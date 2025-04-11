---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/advanced-settings.html
---

# Override input settings [advanced-settings]

Behind the scenes, each module starts a Filebeat input. Advanced users can add or override any input settings. For example, you can set [close_eof](/reference/filebeat/filebeat-input-log.md#filebeat-input-log-close-eof) to `true` in the module configuration:

```yaml
- module: nginx
  access:
    input:
      close_eof: true
```

Or at the command line when you run Filebeat:

```sh
-M "nginx.access.input.close_eof=true"
```

You can use wildcards to change variables or settings for multiple modules/filesets at once. For example, you can enable `close_eof` for all the filesets in the `nginx` module:

```sh
-M "nginx.*.input.close_eof=true"
```

You can also enable `close_eof` for all inputs created by any of the modules:

```sh
-M "*.*.input.close_eof=true"
```

