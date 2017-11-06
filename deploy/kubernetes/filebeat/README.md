# Filebeat

## Ship logs from Kubernetes to Elasticsearch

### Kubernetes DaemonSet

By deploying filebeat as a [DaemonSet](https://kubernetes.io/docs/concepts/workloads/controllers/daemonset/)
we ensure we get a running filebeat daemon on each node of the cluster.

Docker logs host folder (`/var/lib/docker/containers`) is mounted on the
filebeat container. Filebeat will start a prospector for these files and start
harvesting them as they appear.

Everything is deployed under `kube-system` namespace, you can change that by
updating YAML manifests under this folder.

### Settings

We use official [Beats Docker images](https://github.com/elastic/beats-docker),
as they allow external files configuration, a [ConfigMap](https://kubernetes.io/docs/tasks/configure-pod-container/configure-pod-configmap/)
is used for kubernetes specific settings. Check [filebeat-configmap.yaml](filebeat-configmap.yaml)
for details.

Also, [filebeat-daemonset.yaml](filebeat-daemonset.yaml) uses a set of environment
variables to configure Elasticsearch output:

Variable | Default | Description
-------- | ------- | -----------
ELASTICSEARCH_HOST | elasticsearch | Elasticsearch host
ELASTICSEARCH_PORT | 9200 | Elasticsearch port
ELASTICSEARCH_USERNAME | elastic | Elasticsearch username for HTTP auth
ELASTICSEARCH_PASSWORD | changeme | Elasticsearch password

If there is an existing `elasticsearch` service in the kubernetes cluster these
defaults will use it.
