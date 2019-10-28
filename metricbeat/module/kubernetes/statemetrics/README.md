# Kube-state-metrics metricset

This metricset connects to kube-state-metrics endpoint to retrieve and report its metrics.

## Version history

- November 2019, first release using kube-state-metrics `v1.7.0`. Coexisting with other `state_*` metrics at the kubernetes module

## Kube State Metrics

Setup documentation can be found at the projects repo:
https://github.com/kubernetes/kube-state-metrics

Test environments use minikube, but any kubernetes provisioner should be ok for this metricset.

## Configuration

See the metricset documentation for the configuration reference.

## Resources

### Certificate signing request

- Make sure kube controller manager uses `--cluster-signing-cert` and `--cluster-signing-key`.
- Create a CSR with your tool of choice. Base64 encode the file and remove carriage return.
- Create the CSR object at kubernetes:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: certificates.k8s.io/v1beta1
kind: CertificateSigningRequest
metadata:
  name: mycert
  labels:
    env: testing
spec:
  request: $(cat myserver.csr)
  usages:
  - digital signature
  - key encipherment
  - server auth
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

- There is only one event type being pushed to elasticsearch which includes creation date, namespace, name and resource version
