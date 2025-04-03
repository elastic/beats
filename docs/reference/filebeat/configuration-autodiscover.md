---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/configuration-autodiscover.html
---

# Autodiscover [configuration-autodiscover]

::::{warning}
If you still have `log` or `container` inputs in your autodiscover templates please follow [our official guide](/reference/filebeat/migrate-to-filestream.md) to migrate existing `log` inputs to `filestream` inputs.

The `log` input is deprecated in version 7.16 and disabled in version 9.0.
::::

When you run applications on containers, they become moving targets to the monitoring system. Autodiscover allows you to track them and adapt settings as changes happen. By defining configuration templates, the autodiscover subsystem can monitor services as they start running.

You define autodiscover settings in the  `filebeat.autodiscover` section of the `filebeat.yml` config file. To enable autodiscover, you specify a list of providers.


## Providers [_providers_2]

Autodiscover providers work by watching for events on the system and translating those events into internal autodiscover events with a common format. When you configure the provider, you can optionally use fields from the autodiscover event to set conditions that, when met, launch specific configurations.

On start, Filebeat will scan existing containers and launch the proper configs for them. Then it will watch for new start/stop events. This ensures you don’t need to worry about state, but only define your desired configs.


#### Docker [_docker_2]

The Docker autodiscover provider watches for Docker containers to start and stop.

It has the following settings:

`host`
:   (Optional) Docker socket (UNIX or TCP socket). It uses `unix:///var/run/docker.sock` by default.

`ssl`
:   (Optional) SSL configuration to use when connecting to the Docker socket.

`cleanup_timeout`
:   (Optional) Specify the time of inactivity before stopping the running configuration for a container, 60s by default.

`labels.dedot`
:   (Optional) Default to be false. If set to true, replace dots in labels with `_`.

These are the fields available within config templating. The `docker.*` fields will be available on each emitted event. event:

* host
* port
* docker.container.id
* docker.container.image
* docker.container.name
* docker.container.labels

For example:

```yaml
{
  "host": "10.4.15.9",
  "port": 6379,
  "docker": {
    "container": {
      "id": "382184ecdb385cfd5d1f1a65f78911054c8511ae009635300ac28b4fc357ce51"
      "name": "redis",
      "image": "redis:3.2.11",
      "labels": {
        "io.kubernetes.pod.namespace": "default"
        ...
      }
    }
  }
}
```

You can define a set of configuration templates to be applied when the condition matches an event. Templates define a condition to match on autodiscover events, together with the list of configurations to launch when this condition happens.

Conditions match events from the provider. Providers use the same format for [Conditions](/reference/filebeat/defining-processors.md#conditions) that processors use.

Configuration templates can contain variables from the autodiscover event. They can be accessed under the `data` namespace. For example, with the example event, "`${data.port}`" resolves to `6379`.

Filebeat supports templates for inputs and modules.

```yaml
filebeat.autodiscover:
  providers:
    - type: docker
      templates:
        - condition:
            contains:
              docker.container.image: redis
          config:
            - type: filestream
              id: container-${data.docker.container.id}
              prospector.scanner.symlinks: true
              parsers:
                - container: ~
              paths:
                - /var/lib/docker/containers/${data.docker.container.id}/*.log
              exclude_lines: ["^\\s+[\\-`('.|_]"]  # drop asciiart lines
```

This configuration launches a `docker` logs input for all containers running an image with `redis` in the name. `labels.dedot` defaults to be `true` for docker autodiscover, which means dots in docker labels are replaced with *_* by default.

If you are using modules, you can override the default input and use the docker input instead.

```yaml
filebeat.autodiscover:
  providers:
    - type: docker
      templates:
        - condition:
            contains:
              docker.container.image: redis
          config:
            - module: redis
              log:
                input:
                  type: filestream
                  id: container-${data.docker.container.id}
                  prospector.scanner.symlinks: true
                  parsers:
                    - container: ~
                  paths:
                    - /var/lib/docker/containers/${data.docker.container.id}/*.log
```

::::{warning}
When using autodiscover, you have to be careful when defining config templates, especially if they are reading from places holding information for several containers. For instance, under this file structure:

`/mnt/logs/<container_id>/*.log`

You can define a config template like this:

**Wrong settings**:

```yaml
autodiscover.providers:
  - type: docker
    templates:
      - condition.contains:
          docker.container.image: nginx
        config:
          - type: filestream
            id: "some-unique-id"
            paths:
              - "/mnt/logs/*/*.log"
```

That would read all the files under the given path several times (one per nginx container). What you really want is to scope your template to the container that matched the autodiscover condition. Good settings:

```yaml
autodiscover.providers:
  - type: docker
    templates:
      - condition.contains:
          docker.container.image: nginx
        config:
          - type: filestream
            id: container-${data.docker.container.id}
            paths:
              - "/mnt/logs/${data.docker.container.id}/*.log"
```

::::



#### Kubernetes [_kubernetes]

The Kubernetes autodiscover provider watches for Kubernetes nodes, pods, services to start, update, and stop.

The `kubernetes` autodiscover provider has the following configuration settings:

`node`
:   (Optional) Specify the node to scope filebeat to in case it cannot be accurately detected, as when running filebeat in host network mode.

`namespace`
:   (Optional) Select the namespace from which to collect the events from the resources. If it is not set, the provider collects them from all namespaces. It is unset by default. The namespace configuration only applies to kubernetes resources that are namespace scoped and if `unique` field is set to `false`.

`cleanup_timeout`
:   (Optional) Specify the time of inactivity before stopping the running configuration for a container, 60s by default.

`kube_config`
:   (Optional) Use given config file as configuration for Kubernetes client. If kube_config is not set, KUBECONFIG environment variable will be checked and if not present it will fall back to InCluster.

`kube_client_options`
:   (Optional) Additional options can be configured for Kubernetes client. Currently client QPS and burst are supported, if not set Kubernetes client’s [default QPS and burst](https://pkg.go.dev/k8s.io/client-go/rest#pkg-constants) will be used. Example:

```yaml
      kube_client_options:
        qps: 5
        burst: 10
```

`resource`
:   (Optional) Select the resource to do discovery on. Currently supported Kubernetes resources are `pod`, `service` and `node`. If not configured `resource` defaults to `pod`.

`scope`
:   (Optional) Specify at what level autodiscover needs to be done at. `scope` can either take `node` or `cluster` as values. `node` scope allows discovery of resources in the specified node. `cluster` scope allows cluster wide discovery. Only `pod` and `node` resources can be discovered at node scope.

`add_resource_metadata`
:   (Optional) Specify filters and configration for the extra metadata, that will be added to the event. Configuration parameters:

    * `node` or `namespace`: Specify labels and annotations filters for the extra metadata coming from node and namespace. By default all labels are included while annotations are not. To change default behaviour `include_labels`, `exclude_labels` and `include_annotations` can be defined. Those settings are useful when storing labels and annotations that require special handling to avoid overloading the storage output. Note: wildcards are not supported for those settings. The enrichment of `node` or `namespace` metadata can be individually disabled by setting `enabled: false`.
    * `deployment`: If resource is `pod` and it is created from a `deployment`, by default the deployment name isn’t added, this can be enabled by setting `deployment: true`.
    * `cronjob`: If resource is `pod` and it is created from a `cronjob`, by default the cronjob name isn’t added, this can be enabled by setting `cronjob: true`.

        Example:


```yaml
      add_resource_metadata:
        namespace:
          include_labels: ["namespacelabel1"]
        node:
          include_labels: ["nodelabel2"]
          include_annotations: ["nodeannotation1"]
        # deployment: false
        # cronjob: false
```

`unique`
:   (Optional) Defaults to `false`. Marking an autodiscover provider as unique results into making the provider to enable the provided templates only when it will gain the leader lease. This setting can only be combined with `cluster` scope. When `unique` is enabled, `resource` and `add_resource_metadata` settings are not taken into account.

`leader_lease`
:   (Optional) Defaults to `filebeat-cluster-leader`. This will be name of the lock lease. One can monitor the status of the lease with `kubectl describe lease beats-cluster-leader`. Different Beats that refer to the same leader lease will be competitors in holding the lease and only one will be elected as leader each time.

`leader_leaseduration`
:   (Optional) Duration that non-leader candidates will wait to force acquire the lease leadership. Defaults to `15s`.

`leader_renewdeadline`
:   (Optional) Duration that the leader will retry refreshing its leadership before giving up. Defaults to `10s`.

`leader_retryperiod`
:   (Optional) Duration that the metricbeat instances running to acquire the lease should wait between tries of actions. Defaults to `2s`.

Configuration templates can contain variables from the autodiscover event. These variables can be accessed under the `data` namespace, e.g. to access Pod IP: `${data.kubernetes.pod.ip}`.

These are the fields available within config templating. The `kubernetes.*` fields will be available on each emitted event:


##### Generic fields: [_generic_fields]

* host


##### Pod specific: [_pod_specific]

| Key | Type | Description |
| --- | --- | --- |
| `port` | `string` | Pod port. If pod has multiple ports exposed should be used `ports.<port-name>` instead |
| `kubernetes.namespace` | `string` | Namespace, where the Pod is running |
| `kubernetes.namespace_uuid` | `string` | UUID of the Namespace, where the Pod is running |
| `kubernetes.namespace_annotations.*` | `object` | Annotations of the Namespace, where the Pod is running. Annotations should be used in not dedoted format, e.g. `kubernetes.namespace_annotations.app.kubernetes.io/name` |
| `kubernetes.pod.name` | `string` | Name of the Pod |
| `kubernetes.pod.uid` | `string` | UID of the Pod |
| `kubernetes.pod.ip` | `string` | IP of the Pod |
| `kubernetes.labels.*` | `object` | Object of the Pod labels. Labels should be used in not dedoted format, e.g. `kubernetes.labels.app.kubernetes.io/name` |
| `kubernetes.annotations.*` | `object` | Object of the Pod annotations. Annotations should be used in not dedoted format, e.g. `kubernetes.annotations.test.io/test` |
| `kubernetes.container.name` | `string` | Name of the container |
| `kubernetes.container.runtime` | `string` | Runtime of the container |
| `kubernetes.container.id` | `string` | ID of the container |
| `kubernetes.container.image` | `string` | Image of the container |
| `kubernetes.node.name` | `string` | Name of the Node |
| `kubernetes.node.uid` | `string` | UID of the Node |
| `kubernetes.node.hostname` | `string` | Hostname of the Node |


##### Node specific: [_node_specific]

| Key | Type | Description |
| --- | --- | --- |
| `kubernetes.labels.*` | `object` | Object of labels of the Node |
| `kubernetes.annotations.*` | `object` | Object of annotations of the Node |
| `kubernetes.node.name` | `string` | Name of the Node |
| `kubernetes.node.uid` | `string` | UID of the Node |
| `kubernetes.node.hostname` | `string` | Hostname of the Node |


##### Service specific: [_service_specific]

| Key | Type | Description |
| --- | --- | --- |
| `port` | `string` | Service port |
| `kubernetes.namespace` | `string` | Namespace of the Service |
| `kubernetes.namespace_uuid` | `string` | UUID of the Namespace of the Service |
| `kubernetes.namespace_annotations.*` | `object` | Annotations of the Namespace of the Service. Annotations should be used in not dedoted format, e.g. `kubernetes.namespace_annotations.app.kubernetes.io/name` |
| `kubernetes.labels.*` | `object` | Object of the Service labels |
| `kubernetes.annotations.*` | `object` | Object of the Service annotations |
| `kubernetes.service.name` | `string` | Name of the Service |
| `kubernetes.service.uid` | `string` | UID of the Service |

If the `include_annotations` config is added to the provider config, then the list of annotations present in the config are added to the event.

If the `include_labels` config is added to the provider config, then the list of labels present in the config will be added to the event.

If the `exclude_labels` config is added to the provider config, then the list of labels present in the config will be excluded from the event.

if the `labels.dedot` config is set to be `true` in the provider config, then `.` in labels will be replaced with `_`. By default it is `true`.

if the `annotations.dedot` config is set to be `true` in the provider config, then `.` in annotations will be replaced with `_`. By default it is `true`.

::::{note}
Starting from 8.6 release `kubernetes.labels.*` used in config templating are not dedoted regardless of `labels.dedot` value. This config parameter only affects the fields added in the final Elasticsearch document. For example, for a pod with label `app.kubernetes.io/name=ingress-nginx` the matching condition should be `condition.equals: kubernetes.labels.app.kubernetes.io/name: "ingress-nginx"`. If `labels.dedot` is set to `true`(default value) the label will be stored in Elasticsearch as `kubernetes.labels.app_kubernetes_io/name`. The same applies for kubernetes annotations.
::::


For example:

```yaml
{
  "host": "172.17.0.21",
  "port": 9090,
  "kubernetes": {
    "container": {
      "id": "bb3a50625c01b16a88aa224779c39262a9ad14264c3034669a50cd9a90af1527",
      "image": "prom/prometheus",
      "name": "prometheus"
    },
    "labels": {
      "project": "prometheus",
      ...
    },
    "namespace": "default",
    "node": {
      "name": "minikube"
    },
    "pod": {
      "name": "prometheus-2657348378-k1pnh"
    }
  },
}
```

Filebeat supports templates for inputs and modules.

```yaml
filebeat.autodiscover:
  providers:
    - type: kubernetes
      templates:
        - condition:
            equals:
              kubernetes.namespace: kube-system
          config:
            - type: filestream
              id: container-${data.kubernetes.container.id}
              prospector.scanner.symlinks: true
              parsers:
                - container: ~
              paths:
                - /var/log/containers/*-${data.kubernetes.container.id}.log
              exclude_lines: ["^\\s+[\\-`('.|_]"]  # drop asciiart lines
```

This configuration launches a `docker` logs input for all containers of pods running in the Kubernetes namespace `kube-system`.

If you are using modules, you can override the default input and use the docker input instead.

```yaml
filebeat.autodiscover:
  providers:
    - type: kubernetes
      templates:
        - condition:
            equals:
              kubernetes.container.image: "redis"
          config:
            - module: redis
              log:
                input:
                  type: filestream
                  id: container-${data.kubernetes.container.id}
                  prospector.scanner.symlinks: true
                  parsers:
                    - container: ~
                  paths:
                    - /var/log/containers/*-${data.kubernetes.container.id}.log
```


#### Jolokia [_jolokia]

The Jolokia autodiscover provider uses Jolokia Discovery to find agents running in your host or your network.

The configuration of this provider consists in a set of network interfaces, as well as a set of templates as in other providers. The network interfaces will be the ones used for discovery probes, each item of `interfaces` has these settings:

`name`
:   the name of the interface (e.g. `br0`), it can contain a wildcard as suffix to apply the same settings to multiple network interfaces of the same type (e.g. `br*`).

`interval`
:   time between probes (defaults to 10s)

`grace_period`
:   time since the last reply to consider an instance stopped (defaults to 30s)

`probe_timeout`
:   max time to wait for responses since a probe is sent (defaults to 1s)

Jolokia Discovery mechanism is supported by any Jolokia agent since version 1.2.0, it is enabled by default when Jolokia is included in the application as a JVM agent, but disabled in other cases as the OSGI or WAR (Java EE) agents. In any case, this feature is controlled with two properties:

* `discoveryEnabled`, to enable the feature
* `discoveryAgentUrl`, if set, this is the URL announced by the agent when being discovered, setting this parameter implicitly enables the feature

There are multiple ways of setting these properties, and they can vary from application to application, please refer to the documentation of your application to find the more suitable way to set them in your case.

Jolokia Discovery is based on UDP multicast requests. Agents join the multicast group 239.192.48.84, port 24884, and discovery is done by sending queries to this group. You have to take into account that UDP traffic between Filebeat and the Jolokia agents has to be allowed. Also notice that this multicast address is in the 239.0.0.0/8 range, that is reserved for private use within an organization, so it can only be used in private networks.

These are the available fields during within config templating. The `jolokia.*` fields will be available on each emitted event.

* jolokia.agent.id
* jolokia.agent.version
* jolokia.secured
* jolokia.server.product
* jolokia.server.vendor
* jolokia.server.version
* jolokia.url

Filebeat supports templates for inputs and modules:

```yaml
filebeat.autodiscover:
  providers:
    - type: jolokia
      interfaces:
      - name: lo
      templates:
      - condition:
          contains:
            jolokia.server.product: "kafka"
        config:
        - module: kafka
          log:
            enabled: true
            var.paths:
            - /var/log/kafka/*.log
```

This configuration starts a jolokia module that collects logs of kafka if it is running. Discovery probes are sent using the local interface.


#### Nomad [_nomad]

::::{warning}
This functionality is in technical preview and may be changed or removed in a future release. Elastic will work to fix any issues, but features in technical preview are not subject to the support SLA of official GA features.
::::


The Nomad autodiscover provider watches for Nomad jobs to start, update, and stop.

The `nomad` autodiscover provider has the following configuration settings:

`address`
:   (Optional) Specify the address of the Nomad agent. By default it will try to talk to a Nomad agent running locally (`http://127.0.0.1:4646`).

`region`
:   (Optional) Region to use. If not provided, the default agent region is used.

`namespace`
:   (Optional) Namespace to use. If not provided the `default` namespace is used.

`secret_id`
:   (Optional) SecretID to use if ACL is enabled in Nomad. This is an example ACL policy to apply to the token.

```json
namespace "*" {
  policy = "read"
}
node {
  policy = "read"
}
agent {
  policy = "read"
}
```

`node`
:   (Optional) Specify the node to scope filebeat to in case it cannot be accurately detected when `node` scope is used.

`scope`
:   (Optional) Specify at what level autodiscover needs to be done at. `scope` can either take `node` or `cluster` as values. `node` scope allows discovery of resources in the specified node. `cluster` scope allows cluster wide discovery. Defaults to `node`.

`wait_time`
:   (Optional) Limits how long a Watch will block. If not specified (or set to `0`) the default configuration from the agent will be used.

`allow_stale`
:   (Optional) allows any Nomad server (non-leader) to service a read. This normally means that the local node where filebeat is allocated will service filebeat’s requests. Defaults to `true`.

The configuration of templates and conditions is similar to that of the Docker provider. Configuration templates can contain variables from the autodiscover event. They can be accessed under `data` namespace.

These are the available fields during config templating. The `nomad.*` fields will be available on each emitted event.

* nomad.allocation.id
* nomad.allocation.name
* nomad.allocation.status
* nomad.datacenter
* nomad.job.name
* nomad.job.type
* nomad.namespace
* nomad.region
* nomad.task.name
* nomad.task.service.canary_tags
* nomad.task.service.name
* nomad.task.service.tags

If the `include_labels` config is added to the provider config, then the list of labels present in the config will be added to the event.

If the `exclude_labels` config is added to the provider config, then the list of labels present in the config will be excluded from the event.

if the `labels.dedot` config is set to be `true` in the provider config, then `.` in labels will be replaced with `_`.

For example:

```yaml
{
  ...
  "region": "europe",
  "allocation": {
    "name": "coffeshop.api[0]",
    "id": "35eba07f-e5e4-20ac-6def-85117bee6efb",
    "status": "running"
  },
  "datacenters": [
    "europe-west4"
  ],
  "namespace": "default",
  "job": {
    "type": "service",
    "name": "coffeshop"
  },
  "task": {
    "service": {
      "name": [
        "coffeshop"
      ],
      "tags": [
        "coffeshop",
        "nginx"
      ],
      "canary_tags": [
        "coffeshop"
      ]
    },
    "name": "api"
  },
  ...
}
```

Filebeat supports templates for inputs and modules.

```yaml
filebeat.autodiscover:
  providers:
    - type: nomad
      node: nomad1
      scope: local
      hints.enabled: true
      allow_stale: true
      templates:
        - condition:
            equals:
              nomad.namespace: web
          config:
            - type: filestream
              id: ${data.nomad.task.name}-${data.nomad.allocation.id} # unique ID required
              paths:
                - /var/lib/nomad/alloc/${data.nomad.allocation.id}/alloc/logs/${data.nomad.task.name}.stderr.[0-9]*
              exclude_lines: ["^\\s+[\\-`('.|_]"]  # drop asciiart lines
```

This configuration launches a `filestream` input for all jobs under the `web` Nomad namespace.

If you are using modules, you can override the default input and customize it to read from the `${data.nomad.task.name}.stdout` and/or `${data.nomad.task.name}.stderr` files.

```yaml
filebeat.autodiscover:
  providers:
    - type: nomad
      templates:
        - condition:
            equals:
              nomad.task.service.tags: "redis"
          config:
            - module: redis
              log:
                input:
                  type: filestream
                  id: ${data.nomad.task.name}-${data.nomad.allocation.id} # unique ID required
                  paths:
                    - /var/lib/nomad/alloc/${data.nomad.allocation.id}/alloc/logs/${data.nomad.task.name}.*
```

::::{warning}
The `docker` input is currently not supported. Nomad doesn’t expose the container ID associated with the allocation. Without the container ID, there is no way of generating the proper path for reading the container’s logs.
::::
