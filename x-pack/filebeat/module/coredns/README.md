# Coredns Module

This is a filebeat module for coredns. It supports both standalone coredns deployment and 
coredns deployment in Kubernetes.

## Caveats

* Module is to be considered _beta_.

## Download and install Filebeat

Grab the filebeat binary from elastic.co, and install it by following the instructions.

## Deployment Scenario #1: coredns native deployment

Make sure to update coredns configuration to enable log plugin. This module assumes that coredns log
entries will be written to /var/log/coredns.log. Should it be not the case, please point the module 
log path to the path of the log file. 

Update filebeat.yml to point to Elasticsearch and Kibana. 
Setup Filebeat.
```
./filebeat setup --modules coredns -e
```

Enable the Filebeat coredns module
```
./filebeat modules enable coredns
```

Start Filebeat
```
./filebeat -e
```

Now, the Coredns logs and dashboard should appear in Kibana.


## Deployment Scenario #2: coredns for kubernetes 

For Kubernetes deployment, the filebeat daemon-set yaml file needs to be deployed to the 
Kubernetes cluster. A sample configuration file is provided under the `beats/deploy` directory - 
filebeat-autodiscover-k8s.yaml.
```
kubectl apply -f filebeat-autodiscover-k8s.yaml
```

#### Note the following section in the yaml file
```
filebeat.autodiscover:
      providers:
        - type: kubernetes
          hints.enabled: true
          default.disable: true
```

This enables auto-discovery and hints for filebeat. When default.disable is set to true (default value is false), it will disable log harvesting for the pod/container, unless it has specific annotations enabled. This gives users more granular control on kubernetes log ingestion.

### Note that you probably need to update the coredns configmap to enable logging, and coredns deployment to add proper annotations. 

Sample configmap for coredns:

```
apiVersion: v1
data:
  Corefile: |
    .:53 {
        log
        errors
        health
        kubernetes cluster.local in-addr.arpa ip6.arpa {
           pods verified
           endpoint_pod_names
           upstream
           fallthrough in-addr.arpa ip6.arpa
        }
        prometheus :9153
        proxy . /etc/resolv.conf
        cache 30
        loop
        reload
        loadbalance
    }
kind: ConfigMap
metadata:
  creationTimestamp: "2019-01-31T21:02:57Z"
  name: coredns
  namespace: kube-system
  resourceVersion: "185717"
  selfLink: /api/v1/namespaces/kube-system/configmaps/coredns
  uid: 95a5d5cb-259b-11e9-8e5d-080027971f3c
```

Sample deployment for coredns:

```
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: coredns
spec:
  replicas: 2
  template:
    metadata:
      annotations:
        "co.elastic.logs/module": "coredns"
        "co.elastic.logs/fileset": "kubernetes"
        "co.elastic.logs/disable": "false"
      labels:
        k8s-app: coredns
    spec:
      <snipped>
```

