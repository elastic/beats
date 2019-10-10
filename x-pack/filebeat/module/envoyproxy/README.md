# Envoyproxy Module

This is a filebeat module for Envoy proxy access log. 

## Caveats

* Module is to be considered _beta_.

## Download and install Filebeat

Grab the filebeat binary from elastic.co, and install it by following the instructions.

## Deployment Scenario #1: envoy native deployment

This module assumes that envoy log entries will be written to /var/log/envoy.log. Should it be not the case, please point the module log path to the path of the log file. 

Update filebeat.yml to point to Elasticsearch and Kibana. 
Setup Filebeat.
```
./filebeat setup --modules envoyproxy -e
```

Enable the Filebeat envoyproxy module
```
./filebeat modules enable envoyproxy
```

Start Filebeat
```
./filebeat -e
```

Now, the Envoy logs and dashboard should appear in Kibana.


## Deployment Scenario #2: envoy for kubernetes 

For Kubernetes deployment, the filebeat daemon-set yaml file needs to be deployed to the Kubernetes cluster. Sample configuration files is provided under the `beats/deploy/filebeat` directory (https://github.com/elastic/beats/tree/master/deploy/kubernetes/filebeat), and can be deployed by doing the following:
```
kubectl apply -f filebeat
```

#### Note the following section in the ConfigMap, make changes to the yaml file if necessary
```
  filebeat.autodiscover:
    providers:
      - type: kubernetes
        hints.enabled: true
        hints.default_config.enabled: false

  processors:
    - add_kubernetes_metadata: ~
```

This enables auto-discovery and hints for filebeat. When default.disable is set to true (default value is false), it will disable log harvesting for the pod/container, unless it has specific annotations enabled. This gives users more granular control on kubernetes log ingestion. The `add_kubernetes_metadata` processor will add enrichment data for Kubernetes to the ingest logs.

#### Note the following section in the DaemonSet, make changes to the yaml file if necessary
```
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: filebeat
  namespace: kube-system
  labels:
    k8s-app: filebeat
spec:
  selector:
    matchLabels:
      k8s-app: filebeat
  template:
    metadata:
      labels:
        k8s-app: filebeat
    spec:
      serviceAccountName: filebeat
      terminationGracePeriodSeconds: 30
      containers:
      - name: filebeat
        image: docker.elastic.co/beats/filebeat:%VERSION%
        args: [
          "sh", "-c", "filebeat setup -e --modules envoyproxy -c /etc/filebeat.yml && filebeat -e -c /etc/filebeat.yml"
        ]
        env:
        # Edit the following values to reflect your setup accordingly
        - name: ELASTICSEARCH_HOST
          value: 192.168.99.1
        - name: ELASTICSEARCH_USERNAME
          value: elastic
        - name: ELASTICSEARCH_PASSWORD
          value: changeme
        - name: KIBANA_HOST
          value: 192.168.99.1
```

The module setup step can also be done separately without Kubernetes if applicable, and in that case, the args can be simplified to:
```
        args: [
          "sh", "-c", "filebeat -e -c /etc/filebeat.yml"
        ]
```

#### Sample Deployment for envoy, using ambassador as an example. Note the annotations.

```
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ambassador
spec:
  replicas: 3
  selector:
    matchLabels:
      service: ambassador
  template:
    metadata:
      annotations:
        "co.elastic.logs/module": "envoyproxy"
        "co.elastic.logs/fileset": "log"
        "co.elastic.logs/disable": "false"
      labels:
        service: ambassador
    spec:
      serviceAccountName: ambassador
      containers:
      - name: ambassador
        image: quay.io/datawire/ambassador:0.50.0
      <snipped>
```

