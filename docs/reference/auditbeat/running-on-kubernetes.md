---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/auditbeat/current/running-on-kubernetes.html
---

# Running Auditbeat on Kubernetes [running-on-kubernetes]

Auditbeat [Docker images](/reference/auditbeat/running-on-docker.md) can be used on Kubernetes to check files integrity.

::::{tip}
Running {{ecloud}} on Kubernetes? See [Run {{beats}} on ECK](docs-content://deploy-manage/deploy/cloud-on-k8s/beats.md).
::::


% However, version {{stack-version}} of Auditbeat has not yet been released, so no Docker image is currently available for this version.


## Kubernetes deploy manifests [_kubernetes_deploy_manifests]

By deploying Auditbeat as a [DaemonSet](https://kubernetes.io/docs/concepts/workloads/controllers/daemonset/) we ensure we get a running instance on each node of the cluster.

Everything is deployed under `kube-system` namespace, you can change that by updating the YAML file.

To get the manifests just run:

```sh
curl -L -O https://raw.githubusercontent.com/elastic/beats/master/deploy/kubernetes/auditbeat-kubernetes.yaml
```

::::{warning}
If you are using Kubernetes 1.7 or earlier: Auditbeat uses a hostPath volume to persist internal data, it’s located under /var/lib/auditbeat-data. The manifest uses folder autocreation (`DirectoryOrCreate`), which was introduced in Kubernetes 1.8. You will need to remove `type: DirectoryOrCreate` from the manifest and create the host folder yourself.

::::



## Settings [_settings]

Some parameters are exposed in the manifest to configure logs destination, by default they will use an existing Elasticsearch deploy if it’s present, but you may want to change that behavior, so just edit the YAML file and modify them:

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


### Running Auditbeat on control plane nodes [_running_auditbeat_on_control_plane_nodes]

Kubernetes control plane nodes can use [taints](https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/) to limit the workloads that can run on them. To run Auditbeat on control plane nodes you may need to update the Daemonset spec to include proper tolerations:

```yaml
spec:
 tolerations:
 - key: node-role.kubernetes.io/control-plane
   effect: NoSchedule
```


## Deploy [_deploy]

To deploy Auditbeat to Kubernetes just run:

```sh
kubectl create -f auditbeat-kubernetes.yaml
```

Then you should be able to check the status by running:

```sh
$ kubectl --namespace=kube-system get ds/auditbeat

NAME       DESIRED   CURRENT   READY     UP-TO-DATE   AVAILABLE   NODE-SELECTOR   AGE
auditbeat   32        32        0         32           0           <none>          1m
```

::::{warning}
Auditbeat is able to monitor the file integrity of files in pods, to do that, the directories with the container root file systems have to be mounted as volumes in the Auditbeat container. For example, containers executed with containerd have their root file systems under `/run/containerd`. The [reference manifest](https://raw.githubusercontent.com/elastic/beats/master/deploy/kubernetes/auditbeat-kubernetes.yaml) contains an example of this.

::::


