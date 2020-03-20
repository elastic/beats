# Kube-state-metrics/Service


This metricset connects to kube-state-metrics endpoint to retrieve and report Service metrics.

## Version history

- December 2019, first release using kube-state-metrics `v1.8.0`.

## Configuration

See the metricset documentation for the configuration reference.

## Manual testing

Create a service. Try different types as:

Example:
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

Then run metricbeat pointing to the kube-state-metrics endpoint.
