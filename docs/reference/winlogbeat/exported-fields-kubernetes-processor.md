---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/winlogbeat/current/exported-fields-kubernetes-processor.html
---

# Kubernetes fields [exported-fields-kubernetes-processor]

Kubernetes metadata added by the kubernetes processor

**`kubernetes.pod.name`**
:   Kubernetes pod name

type: keyword


**`kubernetes.pod.uid`**
:   Kubernetes Pod UID

type: keyword


**`kubernetes.pod.ip`**
:   Kubernetes Pod IP

type: ip


**`kubernetes.namespace`**
:   Kubernetes namespace

type: keyword


**`kubernetes.node.name`**
:   Kubernetes node name

type: keyword


**`kubernetes.node.hostname`**
:   Kubernetes hostname as reported by the node’s kernel

type: keyword


**`kubernetes.labels.*`**
:   Kubernetes labels map

type: object


**`kubernetes.annotations.*`**
:   Kubernetes annotations map

type: object


**`kubernetes.selectors.*`**
:   Kubernetes selectors map

type: object


**`kubernetes.replicaset.name`**
:   Kubernetes replicaset name

type: keyword


**`kubernetes.deployment.name`**
:   Kubernetes deployment name

type: keyword


**`kubernetes.statefulset.name`**
:   Kubernetes statefulset name

type: keyword


**`kubernetes.container.name`**
:   Kubernetes container name (different than the name from the runtime)

type: keyword


