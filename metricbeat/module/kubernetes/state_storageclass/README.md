### Version history

**January 2023**: Kube state metrics versions 2.4.2-2.7.0

### Resources

- [State storage class metrics](https://github.com/kubernetes/kube-state-metrics/blob/main/internal/store/storageclass.go):
  declaration and description

### Metrics insight

All metrics have the labels:
- namespace
- pod
- uid

Additionally:
- kube_storageclass_info
  - provisioner
  - reclaim_policy
  - volume_binding_mode
- kube_storageclass_labels
- kube_storageclass_created

### Setup environment for manual tests
Go to `metricbeat/module/kubernetes/_meta/test/docs`.


























# Kube-state-metrics/StorageClass

This metricset connects to kube-state-metrics endpoint to retrieve and report Storage Class metrics.

Interestingly enough kube-state-metrics does not repport annotations, we are unable to inform which storage class is default. We can consider enriching adding that info, or contributing back to kube-state-metrics to add annotations.

## Version history

- February 2020, first release using kube-state-metrics `v1.8.0`.

## Configuration

See the metricset documentation for the configuration reference.

## Manual testing

Probably your kubernetes cluster already has a storage class. You can add extra SCs:

Example:

```bash
kubectl apply -f - << EOF
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: beats-test-sc1
  labels:
    testl1: value1
    testl2: value2
provisioner: kubernetes.io/non-existing1
reclaimPolicy: Retain
volumeBindingMode: WaitForFirstConsumer
---
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: beats-test-sc2
  labels:
    testl3: value3
    testl4: value4
provisioner: kubernetes.io/non-existing2
reclaimPolicy: Delete
volumeBindingMode: Immediate
EOF
```

Then run metricbeat pointing to the kube-state-metrics endpoint.
