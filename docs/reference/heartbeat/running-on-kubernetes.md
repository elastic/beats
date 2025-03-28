---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/heartbeat/current/running-on-kubernetes.html
---

# Running Heartbeat on Kubernetes [running-on-kubernetes]

Heartbeat [Docker images](/reference/heartbeat/running-on-docker.md) can be used on Kubernetes to check resources uptime.

::::{tip}
Running {{ecloud}} on Kubernetes? See [Run {{beats}} on ECK](docs-content://deploy-manage/deploy/cloud-on-k8s/beats.md)
::::


% However, version {{stack-version}} of Heartbeat has not yet been released, so no Docker image is currently available for this version.


## Kubernetes deploy manifests [_kubernetes_deploy_manifests]

A single Heartbeat can check for uptime of the whole cluster.

Everything is deployed under `kube-system` namespace, you can change that by updating the YAML file.

To get the manifests just run:

```sh
curl -L -O https://raw.githubusercontent.com/elastic/beats/master/deploy/kubernetes/heartbeat-kubernetes.yaml
```

::::{warning}
If you are using Kubernetes 1.7 or earlier: Heartbeat uses a hostPath volume to persist internal data, it’s located under /var/lib/heartbeat-data. The manifest uses folder autocreation (`DirectoryOrCreate`), which was introduced in Kubernetes 1.8. You will need to remove `type: DirectoryOrCreate` from the manifest and create the host folder yourself.

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


## Deploy [_deploy]

To deploy Heartbeat to Kubernetes just run:

```sh
kubectl create -f heartbeat-kubernetes.yaml
```

Then you should be able to check the status by running:

```sh
$ kubectl --namespace=kube-system get deployment/heartbeat

NAME        READY   UP-TO-DATE   AVAILABLE   AGE
heartbeat   1/1     1            1           1m
```


## Running Heartbeat as unprivileged user [_running_heartbeat_as_unprivileged_user]

Under Kubernetes, Heartbeat can run as a non-root user, but requires some privileged network capabilities to operate correctly. Ensure that the `NET_RAW` capability is available to the container.

```yaml subs=true
containers:
- name: heartbeat
  image: docker.elastic.co/beats/heartbeat:{{stack-version}}
  securityContext:
    runAsUser: 1000
    runAsGroup: 1000
    capabilities:
      add: [ NET_RAW ]
```

