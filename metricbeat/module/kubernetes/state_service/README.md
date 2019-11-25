# Kube-state-metrics/Service


This metricset connects to kube-state-metrics endpoint to retrieve and report Service metrics.

## Version history

## Version history

- November 2019, first release using kube-state-metrics `v1.8.0`.

## Configuration

See the metricset documentation for the configuration reference.

## Service


- Make sure kube controller manager uses `--cluster-signing-cert` and `--cluster-signing-key`.
- Create a CSR with your tool of choice. Base64 encode the file and remove carriage return.
- Create the CSR object at kubernetes:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Service
metadata:
  name: elastic-test-svc
  labels:
    test1: value1
    test2: value2
spec:
  selector:
    app: elastic-test-app
  ports:
    - name: port80
      protocol: TCP
      port: 80
      targetPort: 9080
    - name: port90
      protocol: TCP
      port: 90
      targetPort: 9090
---
apiVersion: v1
kind: Service
metadata:
  name: elastic-external-svc
  labels:
    test-external1: value1
    test-external2: value2
spec:
  type: ExternalName
  externalName: elastic.resource
EOF
```

- Operate on the CSR object (approve, deny)

```bash
kubectl certificate approve testcert
```

- Create a number of CSRs (pending, approved, denied, labeled ...)
- Add labels to the CSR (they will be reported by the metricset)
- Launch metricbeat enabling this metricset

### Configmap

- Create configmap objects at kubernetes at different namespaces

```bash
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  namespace: kube-system
  name: myconfig
data:
  set1: /
    item1.1=one
    item1.2=two
  set2: /
    item2.1=uno
    item2.2=dos
EOF
```
