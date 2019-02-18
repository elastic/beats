# kubernetes module

This module enables log ingestion for kubernetes, specifically envoy and coredns filesets. 

## Caveats

* Module is to be considered _beta_.

## Download and install Filebeat

Grab the filebeat binary from elastic.co, and install it by following the instructions.

## Enable kubernetes module by deploying the daemon set yaml file 
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
  name: ambassador
spec:
  replicas: 3
  template:
    metadata:
      annotations:
        "co.elastic.logs/module": "kubernetes"
        "co.elastic.logs/fileset": "envoy"
        "co.elastic.logs/disable": "false"
      labels:
        service: ambassador
```

