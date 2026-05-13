---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/kubernetes-autodiscover.html
applies_to:
  stack: ga
  serverless: ga
---

# Files are not fully ingested when using autodiscover [kube-foo]

By default Filebeat closes files as soon as they are removed. This can
cause Filebeat not to ingest the last log lines if files are removed
shortly after the last entries were written. This is a common cause of
data loss when using Kubernetes autodiscover.

To prevent this from happening, set:
- `close.on_state_change.removed: false` for the Filestream input
- `close_removed: false` for the Log or Container input.

{applies_to}`stack: ga 9.0.8+` The hints based autodiscover configuration includes the
`close.on_state_change.removed` setting, set to `false` by default.

:::{note}
In Filebeat versions 8.x and between versions 9.0.0 - 9.0.7 and
9.1.0 - 9.1.4, this setting isn't specified by default, so you must
add it to the configuration manually.
:::


Here is an example of setting `close.on_state_change.removed: false`
when using hints on Kubernetes:
```yaml
filebeat.autodiscover:
  providers:
    - type: kubernetes
      hints.enabled: true
      hints.default_config:
        type: filestream
        id: container-logs-${data.container.id}
        prospector.scanner.symlinks: true
        close.on_state_change.removed: false
        parsers:
          - container: ~
        paths:
          - /var/log/containers/*-${data.container.id}.log
```
