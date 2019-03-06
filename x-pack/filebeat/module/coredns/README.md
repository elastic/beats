# Coredns Module

This is a filebeat module for coredns. It supports both standalone coredns deployment and 
coredns deployment in Kubernetes.

## Caveats

* Module is to be considered _beta_.

## Download and install Filebeat

Grab the filebeat binary from elastic.co, and install it by following the instructions.



## Enable coredns module for kubernetes by deploying the daemon-set yaml file 
```
kubectl apply -f k8s-ingest.yaml
```
### Note the following section in the yaml file

```
filebeat.autodiscover:
      providers:
        - type: kubernetes
          hints.enabled: true
          default.disable: true
```

This enables auto-discovery and hints for filebeat. When default.disable is set to true (default value false), it will disable log harvesting for the pod/container, unless it has specific annotations enabled. This gives users more granular control on kubernetes log ingestion.

### Sample kubernetes deployment configuration with annotations, and disable set to false, which enables log harvesting for the pods
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
        "co.elastic.logs/fileset": "log"
        "co.elastic.logs/disable": "false"
      labels:
        k8s-app: coredns
```

