# Filebeat

## Ship logs from Kubernetes to Elasticsearch

### Kubernetes DaemonSet

By deploying filebeat as a [DaemonSet](https://kubernetes.io/docs/concepts/workloads/controllers/daemonset/)
we ensure we get a running filebeat daemon on each node of the cluster.

Kubernetes stores logs on `/var/log/pods` and uses symlinks on `/var/log/containers`
for active log files. Refer to the official [Kubernetes documentation on log rotation](https://kubernetes.io/docs/concepts/cluster-administration/logging/#log-rotation)
for more details.

When the directory is mounted on the filebeat container. Filebeat will start an 
input for these files and start harvesting them as they appear.

Everything is deployed under `kube-system` namespace, you can change that by
updating YAML manifests under this folder.

Filebeat can also ship rotated logs, including the GZIP-compressed. Refer
to [Run Filebeat on Kubernetes](https://www.elastic.co/docs/reference/beats/filebeat/filebeat-input-filestream#reading-gzip-files)
for instructions on how to enable this.

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
