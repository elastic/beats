---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/running-on-kubernetes.html
applies_to:
  stack: ga
  serverless: ga
---

# Run Filebeat on Kubernetes [running-on-kubernetes]

You can use Filebeat [Docker images](/reference/filebeat/running-on-docker.md) on Kubernetes to retrieve and ship container logs.

::::{tip}
Running {{ecloud}} on Kubernetes? See [Run {{beats}} on ECK](docs-content://deploy-manage/deploy/cloud-on-k8s/beats.md).
::::


## Kubernetes deploy manifests for Filebeat [_kubernetes_deploy_manifests]

You deploy Filebeat as a [DaemonSet](https://kubernetes.io/docs/concepts/workloads/controllers/daemonset/) to ensure there’s a running instance on each node of the cluster.

The container logs host folder (`/var/log/containers`) is mounted on the Filebeat container. Filebeat starts an input for the files and begins harvesting them as soon as they appear in the folder.

Everything is deployed under the `kube-system` namespace by default. To change the namespace, modify the manifest file.

To download the manifest file, run:

```sh subs=true
curl -L -O https://raw.githubusercontent.com/elastic/beats/{{ version.stack | M.M }}/deploy/kubernetes/filebeat-kubernetes.yaml
```

::::{warning}
**If you are using Kubernetes 1.7 or earlier:** Filebeat uses a hostPath volume to persist internal data. It’s located under `/var/lib/filebeat-data`. The manifest uses folder autocreation (`DirectoryOrCreate`), which was introduced in Kubernetes 1.8. You need to remove `type: DirectoryOrCreate` from the manifest, and create the host folder yourself.
::::

To support runtime environments different from Docker, like CRI-O or containerd, configure the `paths` as follows:

:::::{tab-set}

::::{tab-item} Single input

Use a single [filestream](/reference/filebeat/filebeat-input-filestream.md) input to ingest all container logs.

```yaml
filebeat.inputs:
- type: filestream
  id: container-logs <1>
  prospector.scanner.symlinks: true <2>
  parsers:
    - container: ~
  paths:
    - /var/log/containers/*.log <3>
  processors:
    - add_kubernetes_metadata:
         host: ${NODE_NAME}
         default_indexers.enabled: false
         default_matchers.enabled: false
         indexers:
            - pod_uid:
         matchers:
            - logs_path:
                 logs_path: "/var/log/pods/" <3>
                 resource_type: "pod" <3>
```
1. All `filestream` inputs require a unique ID. One input will be created for all container logs.
2. Container logs use symlinks, so they need to be enabled.
3. Path for all container logs.

::::

::::{tab-item} One input per container

Use [autodiscover](//reference/filebeat/configuration-autodiscover.md#_kubernetes) to generate a
[filestream](/reference/filebeat/filebeat-input-filestream.md) input per
container.

```yaml
 filebeat.autodiscover:
   providers:
     - type: kubernetes
       node: ${NODE_NAME}
       hints.enabled: true
       hints.default_config:
         type: filestream
         id: container-${data.kubernetes.container.id} <1>
         prospector.scanner.symlinks: true <2>
         parsers:
           - container: ~
         paths:
           - /var/log/containers/*-${data.kubernetes.container.id}.log <3>
```

1. All `filestream` inputs require a unique ID.
2. Container logs use symlinks, so they need to be enabled.
3. A path for each container, so the input will only ingest the logs from its
container.

::::
:::::

## Settings [_settings]

By default, Filebeat sends events to an existing Elasticsearch deployment, if present. To specify a different destination, change the following parameters in the manifest file:

```yaml
- name: ELASTICSEARCH_HOST
  value: elasticsearch
- name: ELASTICSEARCH_PORT
  value: "9200"
- name: ELASTICSEARCH_USERNAME
  value: elastic
- name: ELASTICSEARCH_PASSWORD
  value: changeme
```

### Running Filebeat on control plane nodes [_running_filebeat_on_control_plane_nodes]

Kubernetes control plane nodes can use [taints](https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/) to limit the workloads that can run on them. To run Filebeat on control plane nodes, you may need to update the Daemonset spec to include proper tolerations:

```yaml
spec:
 tolerations:
 - key: node-role.kubernetes.io/control-plane
   effect: NoSchedule
```


### Red Hat OpenShift configuration [_red_hat_openshift_configuration]

If you are using Red Hat OpenShift, you need to specify additional settings in the manifest file and enable the container to run as privileged. Filebeat needs to run as a privileged container to mount logs written on the node (hostPath) and read them.

1. Modify the `DaemonSet` container spec in the manifest file:

    ```yaml
      securityContext:
        runAsUser: 0
        privileged: true
    ```

2. Grant the `filebeat` service account access to the privileged SCC:

    ```shell
    oc adm policy add-scc-to-user privileged system:serviceaccount:kube-system:filebeat
    ```

    This command enables the container to be privileged as an administrator for OpenShift.

3. Override the default node selector for the `kube-system` namespace (or your custom namespace) to allow for scheduling on any node:

    ```shell
    oc patch namespace kube-system -p \
    '{"metadata": {"annotations": {"openshift.io/node-selector": ""}}}'
    ```

    This command sets the node selector for the project to an empty string. If you don’t run this command, the default node selector will skip control plane nodes.

## Log rotation [_logrotation]

Filebeat supports reading from rotating log files, [including GZIP files](/reference/filebeat/filebeat-input-filestream.md#reading-gzip-files).
However, some log rotation strategies can result in lost or duplicate events
when using Filebeat to forward messages. For more information, refer to
[Log rotation results in lost or duplicate events](/reference/filebeat/file-log-rotation.md).

Kubernetes stores logs on `/var/log/pods` and uses symlinks on `/var/log/containers`
for active log files. For full details, refer to the official
[Kubernetes documentation on log rotation](https://kubernetes.io/docs/concepts/cluster-administration/logging/#log-rotation).

Ingest rotated logs by enabling decompression of GZIP files and changing the monitored
path to `/var/log/pods/` instead of `/var/log/containers`, which only contains
active log files.

::::{important}
When you change the path on an existing deployment,
Filebeat reads all existing files in the new directory from the beginning.
This action causes a one-time re-ingestion of the log files.

After the initial scan, Filebeat tracks files normally and will only
ingest new log data.
::::

The following are examples of configurations for ingesting rotated log files:

### Single input

Use a single [filestream](/reference/filebeat/filebeat-input-filestream.md) input to ingest all container logs.

::::{applies-switch}
:group: log-rotation

:::{applies-item} stack: ga 9.3

```yaml
    filebeat.inputs:
       - type: filestream
         id: kubernetes-container-logs
         compression: auto <1>
         parsers:
            - container: ~
         paths:
            - /var/log/pods/*/*/*.log* <2>
         prospector:
            scanner:
               fingerprint.enabled: true
         file_identity.fingerprint: ~
         processors:
            - add_kubernetes_metadata:
                 host: ${NODE_NAME}
                 default_indexers.enabled: false
                 default_matchers.enabled: false
                 indexers:
                    - pod_uid:
                 matchers:
                    - logs_path:
                         logs_path: "/var/log/pods/" <3>
                         resource_type: "pod" <3>
```

1. {applies_to}`stack: ga 9.3.0` Enable gzip detection and decompression. Refer to [Reading GZIP files](/reference/filebeat/filebeat-input-filestream.md#reading-gzip-files).

2. `/var/log/pods/` contains the active log files as well as the rotated log files.

3. `add_kubernetes_metadata` needs to be configured to match pod metadata based
   on the new path, `/var/log/pods/`.

:::{note}
With this configuration, [add_kubernetes_metadata](/reference/filebeat/add-kubernetes-metadata.md#_logs_path)
adds pod metadata, which does not include
container data (such as `kubernetes.container.name`). If you need container
metadata, you must consider using autodiscover instead. Refer to the
[autodiscover documentation](/reference/filebeat/configuration-autodiscover.md#_kubernetes) for details.
:::

:::

:::{applies-item} stack: beta 9.2

```yaml
    filebeat.inputs:
       - type: filestream
         id: kubernetes-container-logs
         gzip_experimental: true <1>
         parsers:
            - container: ~
         paths:
            - /var/log/pods/*/*/*.log* <2>
         prospector:
            scanner:
               fingerprint.enabled: true
         file_identity.fingerprint: ~
         processors:
            - add_kubernetes_metadata:
                 host: ${NODE_NAME}
                 default_indexers.enabled: false
                 default_matchers.enabled: false
                 indexers:
                    - pod_uid:
                 matchers:
                    - logs_path:
                         logs_path: "/var/log/pods/" <3>
                         resource_type: "pod" <3>
```

1. {applies_to}`stack: removed 9.3+, beta =9.2` Enable gzip decompression. Refer to [Reading GZIP files](/reference/filebeat/filebeat-input-filestream.md#reading-gzip-files).

2. `/var/log/pods/` contains the active log files as well as the rotated log files.

3. `add_kubernetes_metadata` needs to be configured to match pod metadata based
   on the new path, `/var/log/pods/`.

:::{note}
With this configuration, [add_kubernetes_metadata](/reference/filebeat/add-kubernetes-metadata.md#_logs_path)
adds pod metadata, which does not include
container data (such as `kubernetes.container.name`). If you need container
metadata, you must consider using autodiscover instead. Refer to the
[autodiscover documentation](/reference/filebeat/configuration-autodiscover.md#_kubernetes) for details.
:::

:::

::::

### One input per container

Use [autodiscover](//reference/filebeat/configuration-autodiscover.md#_kubernetes) to generate a
[filestream](/reference/filebeat/filebeat-input-filestream.md) input per
container.

::::{applies-switch}
:group: log-rotation

:::{applies-item} stack: ga 9.3

```yaml
     filebeat.autodiscover:
        id: kubernetes-container-logs-${data.kubernetes.pod.uid}-${data.kubernetes.container.name}
        compression: auto <1>
        paths:
          - /var/log/pods/${data.kubernetes.namespace}_${data.kubernetes.pod.name}_${data.kubernetes.pod.uid}/${data.kubernetes.container.name}/*.log* <2>

        parsers:
          - container: ~
        prospector:
          scanner:
            fingerprint.enabled: true
        file_identity.fingerprint: ~
```

1. {applies_to}`stack: ga 9.3.0` Enable gzip detection and decompression. Refer to [Reading GZIP files](/reference/filebeat/filebeat-input-filestream.md#reading-gzip-files).

2. `/var/log/pods/` contains the active log files as well as the rotated log files.
   The input is configured to only read logs from the container it's for.

:::

:::{applies-item} stack: beta 9.2

```yaml
     filebeat.autodiscover:
        id: kubernetes-container-logs-${data.kubernetes.pod.uid}-${data.kubernetes.container.name}
        gzip_experimental: true <1>
        paths:
          - /var/log/pods/${data.kubernetes.namespace}_${data.kubernetes.pod.name}_${data.kubernetes.pod.uid}/${data.kubernetes.container.name}/*.log* <2>

        parsers:
          - container: ~
        prospector:
          scanner:
            fingerprint.enabled: true
        file_identity.fingerprint: ~
```

1. {applies_to}`stack: removed 9.3+, beta =9.2` Enable gzip decompression. Refer to [Reading GZIP files](/reference/filebeat/filebeat-input-filestream.md#reading-gzip-files).

2. `/var/log/pods/` contains the active log files as well as the rotated log files.
   The input is configured to only read logs from the container it's for.

:::

::::


## Load {{kib}} dashboards [_load_kib_dashboards]

Filebeat comes packaged with various pre-built {{kib}} dashboards that you can use to visualize logs from your Kubernetes environment.

If these dashboards are not already loaded into {{kib}}, you must [install Filebeat](/reference/filebeat/filebeat-installation-configuration.md) on any system that can connect to the {{stack}}, and then run the `setup` command to load the dashboards. To learn how, see [Load {{kib}} dashboards](/reference/filebeat/load-kibana-dashboards.md).

The `setup` command does not load the ingest pipelines used to parse log lines. By default, ingest pipelines are set up automatically the first time you run Filebeat and connect to {{es}}.

::::{important}
If you are using an output other than {{es}}, such as {{ls}}, you need to:

- [Load the index template manually](/reference/filebeat/filebeat-template.md#load-template-manually)
- [Load {{kib}} dashboards](/reference/filebeat/load-kibana-dashboards.md)
- [Load ingest pipelines](/reference/filebeat/load-ingest-pipelines.md)
::::


## Deploy [_deploy]

To deploy Filebeat to Kubernetes, run:

```sh
kubectl create -f filebeat-kubernetes.yaml
```

To check the status, run:

```sh
$ kubectl --namespace=kube-system get ds/filebeat

NAME       DESIRED   CURRENT   READY     UP-TO-DATE   AVAILABLE   NODE-SELECTOR   AGE
filebeat   32        32        0         32           0           <none>          1m
```

Log events should start flowing to Elasticsearch. The events are annotated with metadata added by the [add_kubernetes_metadata](/reference/filebeat/add-kubernetes-metadata.md) processor.


## Parsing JSON logs [_parsing_json_logs]

The application logs from workloads running on Kubernetes are usually in JSON format. In such cases, special handling can be applied to parse the JSON logs properly and decode them into fields.

You can configure [Filebeat autodiscover](/reference/filebeat/configuration-autodiscover.md) to identify and parse JSON logs in two different ways:

- [Using templates and `ndjson` parser options](#templates-and-parser-options)
- [Using hints and annotations](#hints-and-annotations)

We will illustrate this using an example of one pod with two containers where only the logs of one container are in JSON format.

Example log:

```
{"type":"log","@timestamp":"2020-11-16T14:30:13+00:00","tags":["warning","plugins","licensing"],"pid":7,"message":"License information could not be obtained from Elasticsearch due to Error: No Living connections error"}
```

### Using templates and `ndjson` parser options [templates-and-parser-options]

To use this method to parse the JSON logs in our example, configure autodiscover:

```yaml
filebeat.autodiscover:
  providers:
    - type: kubernetes
      node: ${NODE_NAME}
      templates:
        - condition:
            contains:
              kubernetes.container.name: "no-json-logging"
          config:
            - type: filestream
              id: container-${data.kubernetes.container.id}
              prospector.scanner.symlinks: true
              parsers:
                - container: ~
              paths:
                - /var/log/containers/*-${data.kubernetes.container.id}.log
        - condition:
            contains:
              kubernetes.container.name: "json-logging"
          config:
            - type: filestream
              id: container-${data.kubernetes.container.id}
              prospector.scanner.symlinks: true
              parsers:
                - container: ~
                - ndjson:
                    target: ""
                    add_error_key: true
                    message_key: message
              paths:
                - /var/log/containers/*-${data.kubernetes.container.id}.log
```

### Using hints and annotations [hints-and-annotations]

To configure autodiscover to parse JSON logs using this method, it is important to annotate the pod to only parse logs of the correct container as JSON logs. To achieve this, construct the annotations like this:

```yaml
co.elastic.logs.<kubernetes_container_name>/json.keys_under_root: "true"
co.elastic.logs.<kubernetes_container_name>/json.add_error_key: "true"
co.elastic.logs.<kubernetes_container_name>/json.message_key: "message"
```

For the example we're using:

1. Configure autodiscover:

    ```yaml
    filebeat.autodiscover:
      providers:
        - type: kubernetes
          node: ${NODE_NAME}
          hints.enabled: true
          hints.default_config:
            type: filestream
            id: container-${data.kubernetes.container.id}
            prospector.scanner.symlinks: true
            parsers:
              - container: ~
            paths:
              - /var/log/containers/*-${data.kubernetes.container.id}.log
    ```

2. Then annotate the pod:

    ```yaml
    annotations:
      co.elastic.logs.json-logging/json.keys_under_root: "true"
      co.elastic.logs.json-logging/json.add_error_key: "true"
      co.elastic.logs.json-logging/json.message_key: "message"
    ```
