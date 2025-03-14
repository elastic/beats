---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/libbeat/current/config-gile-format-refs.html
---

# Reference variables [config-gile-format-refs]

Beats settings can reference other settings splicing multiple optionally custom named settings into new values. References use the same syntax as [Environment variables](/reference/libbeat/config-file-format-env-vars.md) do. Only fully collapsed setting names can be referenced to.

For example the filebeat registry file defaults to:

```yaml
filebeat.registry: ${path.data}/registry
```

With `path.data` being an implicit config setting, that is overridable from command line, as well as in the configuration file.

Example referencing `es.host` in `output.elasticsearch.hosts`:

```yaml
es.host: '${ES_HOST:localhost}'

output.elasticsearch:
  hosts: ['http://${es.host}:9200']
```

Introducing `es.host`, the host can be overwritten from command line using `-E es.host=another-host`.

Plain references, having no default value and are not spliced with other references or strings can reference complete namespaces.

These setting with duplicate content:

```yaml
namespace1:
  subnamespace:
    host: localhost
    sleep: 1s

namespace2:
  subnamespace:
    host: localhost
    sleep: 1s
```

can be rewritten to

```yaml
namespace1: ${shared}
namespace2: ${shared}

shared:
  subnamespace:
    host: localhost
    sleep: 1s
```

when using plain references.

