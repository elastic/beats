# Heartbeat

## Monitor Kubernetes services uptime

### Kubernetes Deployment

Heartbeat can be deployed to monitor the whole cluster from a single pod.

Everything is deployed under `kube-system` namespace, you can change that by
updating YAML manifests under this folder.

### Settings

We use official [Beats Docker images](https://github.com/elastic/beats-docker),
as they allow external files configuration, a [ConfigMap](https://kubernetes.io/docs/tasks/configure-pod-container/configure-pod-configmap/)
is used for kubernetes specific settings. Check [heartbeat-configmap.yaml](heartbeat-configmap.yaml)
for details.

Also, [heartbeat-deployment.yaml](heartbeat-deployment.yaml) uses a set of environment
variables to configure Elasticsearch output:

Variable | Default | Description
-------- | ------- | -----------
ELASTICSEARCH_HOST | elasticsearch | Elasticsearch host
ELASTICSEARCH_PORT | 9200 | Elasticsearch port
ELASTICSEARCH_USERNAME | elastic | Elasticsearch username for HTTP auth
ELASTICSEARCH_PASSWORD | changeme | Elasticsearch password

If there is an existing `elasticsearch` service in the kubernetes cluster these
defaults will use it.
