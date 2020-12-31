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
