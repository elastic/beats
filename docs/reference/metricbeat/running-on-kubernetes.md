---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/running-on-kubernetes.html
---

# Run Metricbeat on Kubernetes [running-on-kubernetes]

You can use Metricbeat [Docker images](/reference/metricbeat/running-on-docker.md) on Kubernetes to retrieve cluster metrics.

::::{tip}
Running {{ecloud}} on Kubernetes? See [Run {{beats}} on ECK](docs-content://deploy-manage/deploy/cloud-on-k8s/beats.md).
::::


% However, version {{stack-version}} of Metricbeat has not yet been released, so no Docker image is currently available for this version.


## Kubernetes deploy manifests [_kubernetes_deploy_manifests]

You deploy Metricbeat as a [DaemonSet](https://kubernetes.io/docs/concepts/workloads/controllers/daemonset/) to ensure that there’s a running instance on each node of the cluster. These instances are used to retrieve most metrics from the host, such as system metrics, Docker stats, and metrics from all the services running on top of Kubernetes.

In addition, one of the Pods in the DaemonSet will constantly hold a *leader lock* which makes it responsible for handling cluster-wide monitoring. This instance is used to retrieve metrics that are unique for the whole cluster, such as Kubernetes events or [kube-state-metrics](https://github.com/kubernetes/kube-state-metrics). You can find more information about leader election configuration options at [*Autodiscover*](/reference/metricbeat/configuration-autodiscover.md).

Note: If you are upgrading from older versions, please make sure there are no redundant parts as left-overs from the old manifests. Deployment specification and its ConfigMaps might be the case.

Everything is deployed under the `kube-system` namespace by default. To change the namespace, modify the manifest file.

To download the manifest file, run:

```sh
curl -L -O https://raw.githubusercontent.com/elastic/beats/master/deploy/kubernetes/metricbeat-kubernetes.yaml
```

::::{warning}
**If you are using Kubernetes 1.7 or earlier:** Metricbeat uses a hostPath volume to persist internal data. It’s located under `/var/lib/metricbeat-data`. The manifest uses folder autocreation (`DirectoryOrCreate`), which was introduced in Kubernetes 1.8. You need to remove `type: DirectoryOrCreate` from the manifest and create the host folder yourself.

::::



## Settings [_settings]

By default, Metricbeat sends events to an existing Elasticsearch deployment, if present. To specify a different destination, change the following parameters in the manifest file:

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


### Running Metricbeat on control plane nodes [_running_metricbeat_on_control_plane_nodes]

Kubernetes control plane nodes can use [taints](https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/) to limit the workloads that can run on them. To run Metricbeat on control plane nodes you may need to update the Daemonset spec to include proper tolerations:

```yaml
spec:
 tolerations:
 - key: node-role.kubernetes.io/control-plane
   effect: NoSchedule
```


### Red Hat OpenShift configuration [_red_hat_openshift_configuration]

If you are using Red Hat OpenShift, you need to specify additional settings in the manifest file and grant the `metricbeat` service account access to the privileged SCC:

1. In the manifest file, edit the `metricbeat-daemonset-modules` ConfigMap, and specify the following settings under `kubernetes.yml` in the `data` section:

    ```yaml
      kubernetes.yml: |-
        - module: kubernetes
          metricsets:
            - node
            - system
            - pod
            - container
            - volume
          period: 10s
          host: ${NODE_NAME}
          hosts: ["https://${NODE_NAME}:10250"]
          bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token
          ssl.certificate_authorities:
            - /path/to/kubelet-service-ca.crt
        - module: kubernetes
          metricsets:
            - proxy
          period: 10s
          host: ${NODE_NAME}
          hosts: ["localhost:29101"]
    ```

    ::::{note}
    `kubelet-service-ca.crt` can be any CA bundle that contains the issuer of the certificate used in the Kubelet API. According to each specific installation of Openshift this can be found either in `secrets` or in `configmaps`. In some installations it can be available as part of the service account secret, in `/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt`. In case of using [Openshift installer](https://github.com/openshift/installer/blob/master/docs/user/gcp/install.md) for GCP then the following `configmap` can be mounted in Metricbeat Pod and use `ca-bundle.crt` in `ssl.certificate_authorities`:
    ::::


    ```shell
    Name:         kubelet-serving-ca
    Namespace:    openshift-kube-apiserver
    Labels:       <none>
    Annotations:  <none>

    Data
    ====
    ca-bundle.crt:
    ```

2. If `https` is used to access `kube-state-metrics`, add the following settings to the `metricbeat-daemonset-config` ConfigMap under the kubernetes autodiscover configuration for the `state_*` metricsets:

    ```yaml
      bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token
      ssl.certificate_authorities:
        - /var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt
    ```

3. Grant the `metricbeat` service account access to the privileged SCC:

    ```shell
    oc adm policy add-scc-to-user privileged system:serviceaccount:kube-system:metricbeat
    ```

    This command enables the container to be privileged as an administrator for OpenShift.

4. If the namespace where elastic-agent is running has the `"openshift.io/node-selector"` annotation set, elastic-agent might not run on all nodes. In this case consider overriding the node selector for the namespace to allow scheduling on any node:

    ```shell
    oc patch namespace kube-system -p \
    '{"metadata": {"annotations": {"openshift.io/node-selector": ""}}}'
    ```

    This command sets the node selector for the project to an empty string. If you don’t run this command, the default node selector will skip control plane nodes.


::::{note}
for openshift versions prior to the version 4.x additionally you need to modify the `DaemonSet` container spec in the manifest file to enable the container to run as privileged:
::::


```yaml
  securityContext:
    runAsUser: 0
    privileged: true
```


## Load {{kib}} dashboards [_load_kib_dashboards]

Metricbeat comes packaged with various pre-built {{kib}} dashboards that you can use to visualize metrics about your Kubernetes environment.

If these dashboards are not already loaded into {{kib}}, you must [install Metricbeat](/reference/metricbeat/metricbeat-installation-configuration.md) on any system that can connect to the {{stack}}, and then run the `setup` command to load the dashboards. To learn how, see [Load {{kib}} dashboards](/reference/metricbeat/load-kibana-dashboards.md).

::::{important}
If you are using a different output other than {{es}}, such as {{ls}}, you need to [Load the index template manually](/reference/metricbeat/metricbeat-template.md#load-template-manually) and [*Load {{kib}} dashboards*](/reference/metricbeat/load-kibana-dashboards.md).

::::



## Deploy [_deploy]

Metricbeat gets some metrics from [kube-state-metrics](https://github.com/kubernetes/kube-state-metrics#usage). If `kube-state-metrics` is not already running, deploy it now (see the [Kubernetes deployment](https://github.com/kubernetes/kube-state-metrics#kubernetes-deployment) docs).

To deploy Metricbeat to Kubernetes, run:

```sh
kubectl create -f metricbeat-kubernetes.yaml
```

To check the status, run:

```sh
$ kubectl --namespace=kube-system  get ds/metricbeat

NAME       DESIRED   CURRENT   READY     UP-TO-DATE   AVAILABLE   NODE-SELECTOR   AGE
metricbeat   32        32        0         32           0           <none>          1m
```

Metrics should start flowing to Elasticsearch.


## Deploying Metricbeat to collect cluster-level metrics in large clusters [_deploying_metricbeat_to_collect_cluster_level_metrics_in_large_clusters]

The size and the number of nodes in a Kubernetes cluster can be fairly large at times, and in such cases the Pod that will be collecting cluster level metrics might face performance issues due to resources limitations. In this case users might consider to avoid using the leader election strategy and instead run a dedicated, standalone Metricbeat instance using a Deployment in addition to the DaemonSet.

