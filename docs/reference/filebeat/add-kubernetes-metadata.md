---
navigation_title: "add_kubernetes_metadata"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/add-kubernetes-metadata.html
---

# Add Kubernetes metadata [add-kubernetes-metadata]


The `add_kubernetes_metadata` processor annotates each event with relevant metadata based on which Kubernetes pod the event originated from. This processor only adds metadata to the events that do not have it yet present.

At startup, it detects an `in_cluster` environment and caches the Kubernetes-related metadata. Events are only annotated if a valid configuration is detected. If it’s not able to detect a valid Kubernetes configuration, the events are not annotated with Kubernetes-related metadata.

Each event is annotated with:

* Pod Name
* Pod UID
* Namespace
* Labels

In addition, the node and namespace metadata are added to the pod metadata.

The `add_kubernetes_metadata` processor has two basic building blocks:

* Indexers
* Matchers

Indexers use pod metadata to create unique identifiers for each one of the pods. These identifiers help to correlate the metadata of the observed pods with actual events. For example, the `ip_port` indexer can take a Kubernetes pod and create identifiers for it based on all its `pod_ip:container_port` combinations.

Matchers use information in events to construct lookup keys that match the identifiers created by the indexers. For example, when the `fields` matcher takes `["metricset.host"]` as a lookup field, it would construct a lookup key with the value of the field `metricset.host`. When one of these lookup keys matches with one of the identifiers, the event is enriched with the metadata of the identified pod.

When `add_kubernetes_metadata` is used with Filebeat, it uses the `container` indexer and the `logs_path`. So events whose path in `log.file.path` contains a reference to a container ID are enriched with metadata of the pod of this container.

This behaviour can be disabled by disabling default indexers and matchers in the configuration:

```yaml
processors:
  - add_kubernetes_metadata:
      default_indexers.enabled: false
      default_matchers.enabled: false
```

You can find more information about the available indexers and matchers, and some examples in [Indexers and matchers](#kubernetes-indexers-and-matchers).

The configuration below enables the processor when filebeat is run as a pod in Kubernetes.

```yaml
processors:
  - add_kubernetes_metadata:
      #labels.dedot: true
      #annotations.dedot: true
```

The configuration below enables the processor on a Beat running as a process on the Kubernetes node.

```yaml
processors:
  - add_kubernetes_metadata:
      host: <hostname>
      # If kube_config is not set, KUBECONFIG environment variable will be checked
      # and if not present it will fall back to InCluster
      kube_config: $Filebeat Reference/.kube/config
      #labels.dedot: true
      #annotations.dedot: true
```

The configuration below has the default indexers and matchers disabled and enables ones that the user is interested in.

```yaml
processors:
  - add_kubernetes_metadata:
      host: <hostname>
      # If kube_config is not set, KUBECONFIG environment variable will be checked
      # and if not present it will fall back to InCluster
      kube_config: ~/.kube/config
      default_indexers.enabled: false
      default_matchers.enabled: false
      indexers:
        - ip_port:
      matchers:
        - fields:
            lookup_fields: ["metricset.host"]
      #labels.dedot: true
      #annotations.dedot: true
```

The `add_kubernetes_metadata` processor has the following configuration settings:

`host`
:   (Optional) Specify the node to scope filebeat to in case it cannot be accurately detected, as when running filebeat in host network mode.

`scope`
:   (Optional) Specify if the processor should have visibility at the node level or at the entire cluster level. Possible values are `node` and `cluster`. Scope is `node` by default.

`namespace`
:   (Optional) Select the namespace from which to collect the metadata. If it is not set, the processor collects metadata from all namespaces. It is unset by default.

`add_resource_metadata`
:   (Optional) Specify filters and configuration for the extra metadata, that will be added to the event. Configuration parameters:
    * `node` or `namespace`: Specify labels and annotations filters for the extra metadata coming from node and namespace. By default all labels are included while annotations are not. To change default behaviour `include_labels`, `exclude_labels` and `include_annotations` can be defined. Those settings are useful when storing labels and annotations that require special handling to avoid overloading the storage output. Note: wildcards are not supported for those settings. The enrichment of `node` or `namespace` metadata can be individually disabled by setting `enabled: false`.
    * `deployment`: If resource is `pod` and it is created from a `deployment`, by default the deployment name is added, this can be disabled by setting `deployment: false`.
    * `cronjob`: If resource is `pod` and it is created from a `cronjob`, by default the cronjob name is added, this can be disabled by setting `cronjob: false`.

    Example:

    ```yaml
          add_resource_metadata:
            namespace:
              include_labels: ["namespacelabel1"]
              #labels.dedot: true
              #annotations.dedot: true
            node:
              include_labels: ["nodelabel2"]
              include_annotations: ["nodeannotation1"]
              #labels.dedot: true
              #annotations.dedot: true
            deployment: false
            cronjob: false
    ```

`kube_config`
:   (Optional) Use given config file as configuration for Kubernetes client. It defaults to `KUBECONFIG` environment variable if present.

`use_kubeadm`
:   (Optional) Default true. By default requests to kubeadm config map are made in order to enrich cluster name by requesting /api/v1/namespaces/kube-system/configmaps/kubeadm-config API endpoint.

`kube_client_options`
:   (Optional) Additional options can be configured for Kubernetes client. Currently client QPS and burst are supported, if not set Kubernetes client’s [default QPS and burst](https://pkg.go.dev/k8s.io/client-go/rest#pkg-constants) will be used. Example:

```yaml
      kube_client_options:
        qps: 5
        burst: 10
```

`cleanup_timeout`
:   (Optional) Specify the time of inactivity before stopping the running configuration for a container. This is `60s` by default.

`sync_period`
:   (Optional) Specify the timeout for listing historical resources.

`default_indexers.enabled`
:   (Optional) Enable or disable default pod indexers when you want to specify your own.

`default_matchers.enabled`
:   (Optional) Enable or disable default pod matchers when you want to specify your own.

`labels.dedot`
:   (Optional) Default to be true. If set to true, then `.` in labels will be replaced with `_`.

`annotations.dedot`
:   (Optional) Default to be true. If set to true, then `.` in labels will be replaced with `_`.


## Indexers and matchers [kubernetes-indexers-and-matchers]

## Indexers [_indexers]

Indexers use pods metadata to create unique identifiers for each one of the pods.

Available indexers are:

`container`
:   Identifies the pod metadata using the IDs of its containers.

`ip_port`
:   Identifies the pod metadata using combinations of its IP and its exposed ports. When using this indexer metadata is identified using the IP of the pods, and the combination if `ip:port` for each one of the ports exposed by its containers.

`pod_name`
:   Identifies the pod metadata using its namespace and its name as `namespace/pod_name`.

`pod_uid`
:   Identifies the pod metadata using the UID of the pod.


## Matchers [_matchers]

Matchers are used to construct the lookup keys that match with the identifiers created by indexes.

### `field_format` [_field_format]

Looks up pod metadata using a key created with a string format that can include event fields.

This matcher has an option `format` to define the string format. This string format can contain placeholders for any field in the event.

For example, the following configuration uses the `ip_port` indexer to identify the pod metadata by combinations of the pod IP and its exposed ports, and uses the destination IP and port in events as match keys:

```yaml
processors:
- add_kubernetes_metadata:
    ...
    default_indexers.enabled: false
    default_matchers.enabled: false
    indexers:
      - ip_port:
    matchers:
      - field_format:
          format: '%{[destination.ip]}:%{[destination.port]}'
```


### `fields` [_fields]

Looks up pod metadata using as key the value of some specific fields. When multiple fields are defined, the first one included in the event is used.

This matcher has an option `lookup_fields` to define the files whose value will be used for lookup.

For example, the following configuration uses the `ip_port` indexer to identify pods, and defines a matcher that uses the destination IP or the server IP for the lookup, the first it finds in the event:

```yaml
processors:
- add_kubernetes_metadata:
    ...
    default_indexers.enabled: false
    default_matchers.enabled: false
    indexers:
      - ip_port:
    matchers:
      - fields:
          lookup_fields: ['destination.ip', 'server.ip']
```

It’s also possible to extract the matching key from fields using a regex pattern. The optional `regex_pattern` field can be used to set the pattern. The pattern **must** contain a capture group named `key`, whose value will be used as the matching key.

For example, the following configuration uses the `container` indexer to identify containers by their id, and extracts the matching key from the cgroup id field added to system process metrics. This field has the form `cri-containerd-<id>.scope`, so we need a regex pattern to obtain the container id.

```yaml
processors:
  - add_kubernetes_metadata:
      indexers:
        - container:
      matchers:
        - fields:
            lookup_fields: ['system.process.cgroup.id']
            regex_pattern: 'cri-containerd-(?P<key>[0-9a-z]+)\.scope'
```


### `logs_path` [_logs_path]

Looks up pod metadata using identifiers extracted from the log path stored in the `log.file.path` field.

This matcher has the following configuration settings:

`logs_path`
:   (Optional) Base path of container logs. If not specified, it uses the default logs path of the platform where Filebeat is running: for Linux - `/var/lib/docker/containers/`, Windows - `C:\\ProgramData\\Docker\\containers`. To change the default value: container ID must follow right after the `logs_path` - `<log_path>/<container_id>`, where `container_id` is a 64-character-long hexadecimal string.

`resource_type`
:   (Optional) Type of the resource to obtain the ID of. Valid `resource_type`:
    * `pod`: to make the lookup based on the pod UID. When `resource_type` is set to `pod`, `logs_path` must be set as well, supported path in this case:
      * `/var/lib/kubelet/pods/` used to read logs from mounted into the pod volumes, those logs end up under `/var/lib/kubelet/pods/<pod UID>/volumes/<volume name>/...` To use `/var/lib/kubelet/pods/` as a `log_path`, `/var/lib/kubelet/pods` must be mounted into the filebeat Pods.
      * `/var/log/pods/` Note: when using `resource_type: 'pod'` logs will be enriched only with pod metadata: pod id, pod name, etc., not container metadata.
    * `container`: to make the lookup based on the container ID, `logs_path` must be set to `/var/log/containers/`. It defaults to `container`.


To be able to use `logs_path` matcher filebeat input path must be a subdirectory of directory defined in `logs_path` configuration setting.

The default configuration is able to lookup the metadata using the container ID when the logs are collected from the default docker logs path (`/var/lib/docker/containers/<container ID>/...` on Linux).

For example the following configuration would use the pod UID when the logs are collected from `/var/lib/kubelet/pods/<pod UID>/...`.

```yaml
processors:
- add_kubernetes_metadata:
    ...
    default_indexers.enabled: false
    default_matchers.enabled: false
    indexers:
      - pod_uid:
    matchers:
      - logs_path:
          logs_path: '/var/lib/kubelet/pods'
          resource_type: 'pod'
```



