---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/configuration-autodiscover-hints.html
---

# Hints based autodiscover [configuration-autodiscover-hints]

::::{warning}
If you still have `log` or `container` inputs in your autodiscover templates please follow [our official guide](/reference/filebeat/migrate-to-filestream.md) to migrate existing `log` inputs to `filestream` inputs.

The `log` input is deprecated in version 7.16 and disabled in version 9.0.
::::

Filebeat supports autodiscover based on hints from the provider. The hints system looks for hints in Kubernetes Pod annotations or Docker labels that have the prefix `co.elastic.logs`. As soon as the container starts, Filebeat will check if it contains any hints and launch the proper config for it. Hints tell Filebeat how to get logs for the given container. By default logs will be retrieved from the container using the `filestream` input. You can use hints to modify this behavior. This is the full list of supported hints:


### `co.elastic.logs/enabled` [_co_elastic_logsenabled]

Filebeat gets logs from all containers by default, you can set this hint to `false` to ignore the output of the container. Filebeat won’t read or send logs from it. If default config is disabled, you can use this annotation to enable log retrieval only for containers with this set to `true`. If you are aiming to use this with Kubernetes, have in mind that annotation values can only be of string type so you will need to explicitly define this as `"true"` or `"false"` accordingly.


### `co.elastic.logs/multiline.*` [_co_elastic_logsmultiline]

Multiline settings. See [Multiline messages](/reference/filebeat/multiline-examples.md) for a full list of all supported options.


### `co.elastic.logs/json.*` [_co_elastic_logsjson]

JSON settings. See [`ndjson`](/reference/filebeat/filebeat-input-filestream.md#filebeat-input-filestream-ndjson) for a full list of all supported options.

For example, the following hints with json options:

```yaml
co.elastic.logs/json.message_key: "log"
co.elastic.logs/json.add_error_key: "true"
```

will lead to the following input configuration:

`filestream`

```yaml
parsers:
  - ndjson:
      message_key: "log"
      add_error_key: "true"
```


### `co.elastic.logs/include_lines` [_co_elastic_logsinclude_lines]

A list of regular expressions to match the lines that you want Filebeat to include. See [Inputs](/reference/filebeat/configuration-filebeat-options.md) for more info.


### `co.elastic.logs/exclude_lines` [_co_elastic_logsexclude_lines]

A list of regular expressions to match the lines that you want Filebeat to exclude. See [Inputs](/reference/filebeat/configuration-filebeat-options.md) for more info.


### `co.elastic.logs/module` [_co_elastic_logsmodule]

Instead of using raw `docker` input, specifies the module to use to parse logs from the container. See [Modules](/reference/filebeat/filebeat-modules.md) for the list of supported modules.


### `co.elastic.logs/fileset` [_co_elastic_logsfileset]

When module is configured, map container logs to module filesets. You can either configure a single fileset like this:

```yaml
co.elastic.logs/fileset: access
```

Or configure a fileset per stream in the container (stdout and stderr):

```yaml
co.elastic.logs/fileset.stdout: access
co.elastic.logs/fileset.stderr: error
```


### `co.elastic.logs/raw` [_co_elastic_logsraw]

When an entire input/module configuration needs to be completely set the `raw` hint can be used. You can provide a stringified JSON of the input configuration. `raw` overrides every other hint and can be used to create both a single or a list of configurations.

```yaml
co.elastic.logs/raw: "[{\"containers\":{\"ids\":[\"${data.container.id}\"]},\"multiline\":{\"negate\":\"true\",\"pattern\":\"^test\"},\"type\":\"docker\"}]"
```


### `co.elastic.logs/processors` [_co_elastic_logsprocessors]

Define a processor to be added to the Filebeat input/module configuration. See [Processors](/reference/filebeat/filtering-enhancing-data.md) for the list of supported processors.

If processors configuration uses list data structure, object fields must be enumerated. For example, hints for the `rename` processor configuration below

```yaml
processors:
  - rename:
      fields:
        - from: "a.g"
          to: "e.d"
      fail_on_error: true
```

will look like:

```yaml
co.elastic.logs/processors.rename.fields.0.from: "a.g"
co.elastic.logs/processors.rename.fields.1.to: "e.d"
co.elastic.logs/processors.rename.fail_on_error: 'true'
```

If processors configuration uses map data structure, enumeration is not needed. For example, the equivalent to the `add_fields` configuration below

```yaml
processors:
  - add_fields:
      target: project
      fields:
        name: myproject
```

is

```yaml
co.elastic.logs/processors.1.add_fields.target: "project"
co.elastic.logs/processors.1.add_fields.fields.name: "myproject"
```

In order to provide ordering of the processor definition, numbers can be provided. If not, the hints builder will do arbitrary ordering:

```yaml
co.elastic.logs/processors.1.dissect.tokenizer: "%{key1} %{key2}"
co.elastic.logs/processors.dissect.tokenizer: "%{key2} %{key1}"
```

In the above sample the processor definition tagged with `1` would be executed first.


### `co.elastic.logs/pipeline` [_co_elastic_logspipeline]

Define an ingest pipeline ID to be added to the Filebeat input/module configuration.

```yaml
co.elastic.logs/pipeline: custom-pipeline
```

When hints are used along with templates, then hints will be evaluated only in case there is no template’s condition that resolves to true. For example:

```yaml
filebeat.autodiscover.providers:
  - type: docker
    hints.enabled: true
    hints.default_config:
      type: filestream
      id: container-${data.container.id}
      prospector.scanner.symlinks: true
      parsers:
        - container: ~
      paths:
        - /var/lib/docker/containers/${data.container.id}/*.log
    templates:
      - condition:
          equals:
            docker.container.labels.type: "pipeline"
        config:
          - type: filestream
            id: container-${data.docker.container.id}
            prospector.scanner.symlinks: true
            parsers:
              - container: ~
            paths:
              - "/var/lib/docker/containers/${data.docker.container.id}/*.log"
            pipeline: my-pipeline
```

In this example first the condition `docker.container.labels.type: "pipeline"` is evaluated and if not matched the hints will be processed and if there is again no valid config the `hints.default_config` will be used.


## Kubernetes [_kubernetes_2]

Kubernetes autodiscover provider supports hints in Pod annotations. To enable it just set `hints.enabled`:

```yaml
filebeat.autodiscover:
  providers:
    - type: kubernetes
      hints.enabled: true
```

You can configure the default config that will be launched when a new container is seen, like this:

```yaml
filebeat.autodiscover:
  providers:
    - type: kubernetes
      hints.enabled: true
      hints.default_config:
        type: filestream
        id: container-${data.container.id}
        prospector.scanner.symlinks: true
        parsers:
          - container: ~
        paths:
          - /var/log/containers/*-${data.container.id}.log  # CRI path
```

You can also disable default settings entirely, so only Pods annotated like `co.elastic.logs/enabled: true` will be retrieved:

```yaml
filebeat.autodiscover:
  providers:
    - type: kubernetes
      hints.enabled: true
      hints.default_config.enabled: false
```

You can annotate Kubernetes Pods with useful info to spin up Filebeat inputs or modules:

```yaml
annotations:
  co.elastic.logs/multiline.pattern: '^\['
  co.elastic.logs/multiline.negate: true
  co.elastic.logs/multiline.match: after
```


### Multiple containers [_multiple_containers]

When a pod has multiple containers, the settings are shared unless you put the container name in the hint. For example, these hints configure multiline settings for all containers in the pod, but set a specific `exclude_lines` hint for the container called `sidecar`.

```yaml
annotations:
  co.elastic.logs/multiline.pattern: '^\['
  co.elastic.logs/multiline.negate: true
  co.elastic.logs/multiline.match: after
  co.elastic.logs.sidecar/exclude_lines: '^DBG'
```


### Multiple sets of hints [_multiple_sets_of_hints]

When a container needs multiple inputs to be defined on it, sets of annotations can be provided with numeric prefixes. If there are hints that don’t have a numeric prefix then they get grouped together into a single configuration.

```yaml
annotations:
  co.elastic.logs/exclude_lines: '^DBG'
  co.elastic.logs/1.include_lines: '^DBG'
  co.elastic.logs/1.processors.dissect.tokenizer: "%{key2} %{key1}"
```

The above configuration would generate two input configurations. The first input handles only debug logs and passes it through a dissect tokenizer. The second input handles everything but debug logs.


### Namespace Defaults [_namespace_defaults]

Hints can be configured on the Namespace’s annotations as defaults to use when Pod level annotations are missing. The resultant hints are a combination of Pod annotations and Namespace annotations with the Pod’s taking precedence. To enable Namespace defaults configure the `add_resource_metadata` for Namespace objects as follows:

```yaml
filebeat.autodiscover:
  providers:
    - type: kubernetes
      hints.enabled: true
      add_resource_metadata:
        namespace:
          include_annotations: ["nsannotation1"]
```


## Docker [_docker_3]

Docker autodiscover provider supports hints in labels. To enable it just set `hints.enabled`:

```yaml
filebeat.autodiscover:
  providers:
    - type: docker
      hints.enabled: true
```

You can configure the default config that will be launched when a new container is seen, like this:

```yaml
filebeat.autodiscover:
  providers:
    - type: docker
      hints.enabled: true
      hints.default_config:
        type: filestream
        id: container-${data.container.id}
        prospector.scanner.symlinks: true
        parsers:
          - container: ~
        paths:
          - /var/log/containers/*-${data.container.id}.log  # CRI path
```

You can also disable default settings entirely, so only containers labeled with `co.elastic.logs/enabled: true` will be retrieved:

```yaml
filebeat.autodiscover:
  providers:
    - type: docker
      hints.enabled: true
      hints.default_config.enabled: false
```

You can label Docker containers with useful info to spin up Filebeat inputs, for example:

```yaml
  co.elastic.logs/module: nginx
  co.elastic.logs/fileset.stdout: access
  co.elastic.logs/fileset.stderr: error
```

The above labels configure Filebeat to use the Nginx module to harvest logs for this container. Access logs will be retrieved from stdout stream, and error logs from stderr.

You can label Docker containers with useful info to decode logs structured as JSON messages, for example:

```yaml
  co.elastic.logs/json.keys_under_root: true
  co.elastic.logs/json.add_error_key: true
  co.elastic.logs/json.message_key: log
```


## Nomad [_nomad_2]

Nomad autodiscover provider supports hints using the [`meta` stanza](https://www.nomadproject.io/docs/job-specification/meta.html). To enable it just set `hints.enabled`:

```yaml
filebeat.autodiscover:
  providers:
    - type: nomad
      hints.enabled: true
```

You can configure the default config that will be launched when a new job is seen, like this:

```yaml
filebeat.autodiscover:
  providers:
    - type: nomad
      hints.enabled: true
      hints.default_config:
        type: filestream
        id: ${data.nomad.task.name}-${data.nomad.allocation.id} # unique ID required
        paths:
          - /opt/nomad/alloc/${data.nomad.allocation.id}/alloc/logs/${data.nomad.task.name}.*
```

You can also disable the default config such that only logs from jobs explicitly annotated with `"co.elastic.logs/enabled" = "true"` will be collected:

```yaml
filebeat.autodiscover:
  providers:
    - type: nomad
      hints.enabled: true
      hints.default_config:
        enabled: false
        type: filestream
        id: ${data.nomad.task.name}-${data.nomad.allocation.id} # unique ID required
        paths:
          - /opt/nomad/alloc/${data.nomad.allocation.id}/alloc/logs/${data.nomad.task.name}.*
```

You can annotate Nomad Jobs using the `meta` stanza with useful info to spin up Filebeat inputs or modules:

```json
meta {
  "co.elastic.logs/enabled"           = "true"
  "co.elastic.logs/multiline.pattern" = "^\\["
  "co.elastic.logs/multiline.negate"  = "true"
  "co.elastic.logs/multiline.match"   = "after"
}
```

If you are using autodiscover then in most cases you will want to use the [`add_nomad_metadata`](/reference/filebeat/add-nomad-metadata.md) processor to enrich events with Nomad metadata. This example configures {{Filebeat}} to connect to the local Nomad agent over HTTPS and adds the Nomad allocation ID to all events from the input. Later in the pipeline the `add_nomad_metadata` processor will use that ID to enrich the event.

```yaml
filebeat.autodiscover:
  providers:
    - type: nomad
      address: https://localhost:4646
      hints.enabled: true
      hints.default_config:
        enabled: false <1>
        type: filestream
        id: ${data.nomad.task.name}-${data.nomad.allocation.id} <2>
        paths:
          - /opt/nomad/alloc/${data.nomad.allocation.id}/alloc/logs/${data.nomad.task.name}.*
        processors:
          - add_fields: <3>
              target: nomad
              fields:
                allocation.id: ${data.nomad.allocation.id}

processors:
  - add_nomad_metadata: <4>
      when.has_fields.fields: [nomad.allocation.id]
      address: https://localhost:4646
      default_indexers.enabled: false
      default_matchers.enabled: false
      indexers:
        - allocation_uuid:
      matchers:
        - fields:
            lookup_fields:
              - 'nomad.allocation.id'
```

1. The default config is disabled meaning any task without the `"co.elastic.logs/enabled" = "true"` metadata will be ignored.
2. Unique ID is required.
3. The `add_fields` processor populates the `nomad.allocation.id` field with the Nomad allocation UUID.
4. The `add_nomad_metadata` processor is configured at the global level so that it is only instantiated one time which saves resources.
