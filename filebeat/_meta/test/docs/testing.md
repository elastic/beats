# Testing Filebeat

## Testing on Kubernetes

### Prerequisites
- create kubernetes cluster using kind, check [here](https://github.com/elastic/beats/blob/main/metricbeat/module/kubernetes/_meta/test/docs/README.md) for details
- deploy ELK stack, check [here](https://github.com/elastic/beats/blob/main/metricbeat/module/kubernetes/_meta/test/docs/README.md) for details

## Playground Filebeat Pod

A slightly modified (comparing to beats/deploy/kubernetes/filebeat-kubernetes.yaml) all-in-one filebeat manifest resides under `01_playground` directory.
Modifications:
- the daemonset executes an infinite sleep command instead of starting filebeat.
- variables `ELASTICSEARCH_HOST`, `ELASTICSEARCH_PORT`, `ELASTICSEARCH_USERNAME`, `ELASTICSEARCH_PASSWORD` variables are set according to local kind EK stack.

> Note: In case of using Elastic Cloud deployment configure the variables `ELASTIC_CLOUD_ID` and `ELASTIC_CLOUD_AUTH` accordingly.

Deploy filebeat:
```bash
cd filebeat/_meta/test/docs
kubectl apply -f 01_playground/filebeat.yaml
```

## Build and launch filebeat process

1. Build filebeat binary and copy it in the running filebeat pod.
Under beats/filebeat execute:
```bash
# Build filebeat
GOOS=linux GOARCH=amd64 go build

# Copy binary in pod
kubectl cp ./filebeat `kubectl get pod -n kube-system -l k8s-app=filebeat -o jsonpath='{.items[].metadata.name}'`:/usr/share/filebeat/ -n kube-system
````
The above command only copies filebeat binary.

2. For configuration files updates modify `filebeat.yml`, defined in the `ConfigMap` and execute again:
```
kubectl apply -f 01_playground/filebeat.yaml
```
> Note: Filebeat on kubernetes can be running with one of the configurations:
> 1. Filebeat use [container input](https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-input-container.html) to read container logs under the given
> path in combination with [`add_kubernetes_metadata` processor](https://www.elastic.co/guide/en/beats/filebeat/current/add-kubernetes-metadata.html) to enrich logs
> with relevant Kubernetes pod metadata.
> 2. Filebeat use autodiscover [kubernetes provider](https://www.elastic.co/guide/en/beats/filebeat/current/configuration-autodiscover.html#_kubernetes).

3. Start filebeat.
```bash
# Exec in the container and launch filebeat
kubectl exec `kubectl get pod -n kube-system -l k8s-app=filebeat -o jsonpath='{.items[].metadata.name}'` -n kube-system -- bash -c "filebeat -e -c /etc/filebeat.yml"
```
Filebeat will launch and the process logs will appear in the terminal.

You can as well exec in filebeat pod with bash command and then run filebeat.
This gives the flexibility to easily start and stop the process.
