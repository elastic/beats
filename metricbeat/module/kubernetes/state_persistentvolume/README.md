# Kube-state-metrics/PersistentVolume


This metricset connects to kube-state-metrics endpoint to retrieve and report Persistent Volume metrics.

## Version history

- November 2019, first release using kube-state-metrics `v1.8.0`.

## Configuration

See the metricset documentation for the configuration reference.

## Manual testing

Running a minikube cluster allows you to create persistent volume using the `/data/` directory at hostpath

```yaml
apiVersion: v1
kind: PersistentVolume
metadata:
  name: pv0001
spec:
  accessModes:
    - ReadWriteOnce
  capacity:
    storage: 5Gi
  hostPath:
    path: /data/pv0001/
```

Try adding labels, and creating volumes from pods using PVC.