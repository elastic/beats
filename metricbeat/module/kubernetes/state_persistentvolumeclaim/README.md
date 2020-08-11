# Kube-state-metrics/PersistentVolumeClaim

This metricset connects to kube-state-metrics endpoint to retrieve and report Persistent Volume Claim metrics.

## Version history

- November 2019, first release using kube-state-metrics `v1.8.0`.

## Configuration

See the metricset documentation for the configuration reference.

## Manual testing

Create Persistent Volume Claims.
- Use non existent storage classes to stuck them pending
```
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: claim2
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 8Gi
  storageClassName: notexisting
  volumeMode: Filesystem
```

- Add labels
```
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  labels:
    category: disposable
    team: observability
  name: claim1
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 80Gi
  storageClassName: standard
  volumeMode: Filesystem
```

