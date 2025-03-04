---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/libbeat/current/config-file-format-namespacing.html
---

# Namespacing [config-file-format-namespacing]

All settings are structured using dictionaries and lists. Those are collapsed into "namespaced" settings, by creating a setting using the full path of the settings name and it’s parent structures names, when reading the configuration file.

For example this setting:

```yaml
output:
  elasticsearch:
    index: 'beat-%{[agent.version]}-%{+yyyy.MM.dd}'
```

gets collapsed into `output.elasticsearch.index: 'beat-%{[agent.version]}-%{+yyyy.MM.dd}'`. The full name of a setting is based on all parent structures involved.

Lists create numeric names starting with 0.

For example this filebeat setting:

```yaml
filebeat:
  inputs:
    - type: log
```

Gets collapsed into `filebeat.inputs.0.type: log`.

Alternatively to using indentation, setting names can be used in collapsed form too.

Note: having two settings with same fully collapsed path is invalid.

Simple filebeat example with partially collapsed setting names and use of compact form:

```yaml
filebeat.inputs:
- type: log
  paths: ["/var/log/*.log"]
  multiline.pattern: '^['
  multiline.match: after

output.elasticsearch.hosts: ["http://localhost:9200"]
```

