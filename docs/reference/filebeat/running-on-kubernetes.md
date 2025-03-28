---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/running-on-kubernetes.html
---

# Run Filebeat on Kubernetes [running-on-kubernetes]

You can use Filebeat [Docker images](/reference/filebeat/running-on-docker.md) on Kubernetes to retrieve and ship container logs.

::::{tip}
Running {{ecloud}} on Kubernetes? See [Run {{beats}} on ECK](docs-content://deploy-manage/deploy/cloud-on-k8s/beats.md).
::::


% However, version {{stack-version}} of Filebeat has not yet been released, so no Docker image is currently available for this version.


## Kubernetes deploy manifests [_kubernetes_deploy_manifests]

You deploy Filebeat as a [DaemonSet](https://kubernetes.io/docs/concepts/workloads/controllers/daemonset/) to ensure there’s a running instance on each node of the cluster.

The container logs host folder (`/var/log/containers`) is mounted on the Filebeat container. Filebeat starts an input for the files and begins harvesting them as soon as they appear in the folder.

Everything is deployed under the `kube-system` namespace by default. To change the namespace, modify the manifest file.

To download the manifest file, run:

```sh
curl -L -O https://raw.githubusercontent.com/elastic/beats/master/deploy/kubernetes/filebeat-kubernetes.yaml
```

::::{warning}
**If you are using Kubernetes 1.7 or earlier:** Filebeat uses a hostPath volume to persist internal data. It’s located under `/var/lib/filebeat-data`. The manifest uses folder autocreation (`DirectoryOrCreate`), which was introduced in Kubernetes 1.8. You need to remove `type: DirectoryOrCreate` from the manifest and create the host folder yourself.

::::



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

Kubernetes control plane nodes can use [taints](https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/) to limit the workloads that can run on them. To run Filebeat on control plane nodes you may need to update the Daemonset spec to include proper tolerations:

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


In order to support runtime environments with Openshift (eg. CRI-O, containerd) you need to configure following path:

```yaml
filebeat.inputs:
- type: container
  paths: <1>
    - '/var/log/containers/*.log'
```

1. Same path needs to be configured in case autodiscovery needs to be enabled:

```yaml
filebeat.autodiscover:
  providers:
    - type: kubernetes
      node: ${NODE_NAME}
      hints.enabled: true
      hints.default_config:
        type: container
        paths:
          - /var/log/containers/*.log
```

::::{note}
`/var/log/containers/\*.log` is normally a symlink to `/var/log/pods/*/*.log`, so above paths can be edited accordingly
::::



## Load {{kib}} dashboards [_load_kib_dashboards]

Filebeat comes packaged with various pre-built {{kib}} dashboards that you can use to visualize logs from your Kubernetes environment.

If these dashboards are not already loaded into {{kib}}, you must [install Filebeat](/reference/filebeat/filebeat-installation-configuration.md) on any system that can connect to the {{stack}}, and then run the `setup` command to load the dashboards. To learn how, see [Load {{kib}} dashboards](/reference/filebeat/load-kibana-dashboards.md).

The `setup` command does not load the ingest pipelines used to parse log lines. By default, ingest pipelines are set up automatically the first time you run Filebeat and connect to {{es}}.

::::{important}
If you are using a different output other than {{es}}, such as {{ls}}, you need to:

* [Load the index template manually](/reference/filebeat/filebeat-template.md#load-template-manually)
* [*Load {{kib}} dashboards*](/reference/filebeat/load-kibana-dashboards.md)
* [*Load ingest pipelines*](/reference/filebeat/load-ingest-pipelines.md)

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


## Parsing json logs [_parsing_json_logs]

It is common case when collecting logs from workloads running on Kubernetes that these applications are logging in json format. In these case, special handling can be applied so as to parse these json logs properly and decode them into fields. Bellow there are provided 2 different ways of configuring [filebeat’s autodiscover](/reference/filebeat/configuration-autodiscover.md) so as to identify and parse json logs. We will use an example of one Pod with 2 containers where only one of these logs in json format.

Example log:

```
{"type":"log","@timestamp":"2020-11-16T14:30:13+00:00","tags":["warning","plugins","licensing"],"pid":7,"message":"License information could not be obtained from Elasticsearch due to Error: No Living connections error"}
```

1. Using `json.*` options with templates

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
                  - type: container
                    paths:
                      - "/var/log/containers/*-${data.kubernetes.container.id}.log"
              - condition:
                  contains:
                    kubernetes.container.name: "json-logging"
                config:
                  - type: container
                    paths:
                      - "/var/log/containers/*-${data.kubernetes.container.id}.log"
                    json.keys_under_root: true
                    json.add_error_key: true
                    json.message_key: message
    ```

2. Using `json.*` options with hints

    Key part here is to properly annotate the Pod to only parse logs of the correct container as json logs. In this, annotation should be constructed like this:

    `co.elastic.logs.<container_name>/json.keys_under_root: "true"`

    Autodiscovery configuration:

    ```yaml
    filebeat.autodiscover:
      providers:
        - type: kubernetes
          node: ${NODE_NAME}
          hints.enabled: true
          hints.default_config:
            type: container
            paths:
              - /var/log/containers/*${data.kubernetes.container.id}.log
    ```

    Then annotate the pod properly:

    ```yaml
    annotations:
        co.elastic.logs.json-logging/json.keys_under_root: "true"
        co.elastic.logs.json-logging/json.add_error_key: "true"
        co.elastic.logs.json-logging/json.message_key: "message"
    ```



## Logrotation [_logrotation]

According to [kubernetes documentation](https://kubernetes.io/docs/concepts/cluster-administration/logging/#logging-at-the-node-level) *Kubernetes is not responsible for rotating logs, but rather a deployment tool should set up a solution to address that*. Different logrotation strategies can cause issues that might make Filebeat losing events or even duplicating events. Users can find more information about Filebeat’s logrotation best practises at Filebeat’s [log rotation specific documentation](/reference/filebeat/file-log-rotation.md)

