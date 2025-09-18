---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/kubernetes-autodiscover.html
applies_to:
  stack: ga
---

# Files are not fully ingested when using autodiscover (Kubernetes, Docker, etc)[kube-foo]

By default Filebeat closes files as soon as they are removed, this can
cause Filebeat not to ingest the last log lines if files are removed
shortly after the last entries were written. This is a common cause of
data loss when using Kubernetes autodiscover.

To prevent this from happening set:
- `close.on_state_change.removed: false` for the Filestream input
- `close_removed: false` for the Log or Container input.

If using hints, this also needs to be set for:
 - <= 9.0.7 if running Filebeat 8.x or 9.0.x
 - <= 9.1.4 if running Filebeat 9.1.x

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
