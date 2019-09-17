# Kube-state-metrics/ResourceQuota

## Version history

- September 2019, `v1.7.0`

## Resources

Docs for 1.7 release of `kube-state-metrics` ResourceQuota can be found here:
https://github.com/kubernetes/kube-state-metrics/blob/release-1.7/docs/resourcequota-metrics.md


## Metrics insight

    - kube_resourcequota{namespace,resourcequota,resource,type} Gauge

        Info about existing `ResourceQuota` and current status        

    - kube_resourcequota_created{namespace,resourcequota} Gauge

        Creation time for `ResourceQuota`

## Setup environment for manual tests

- Setup kubernetes environment for beats testing

https://github.com/elastic/beats/tree/master/metricbeat/module/kubernetes/_meta/test

- Install `kube-state-metrics`

As part of the referred document above, follow these instructions

https://github.com/elastic/beats/tree/master/metricbeat/module/kubernetes/_meta/test#testing-kubernetes-loads

- Create `ResourceQuota` objects

The manifest are found at this location, not only creates the `ResourceQuota` objects, but also other resources that will fail because of the existence of the quota at the namespace:

https://github.com/elastic/beats/tree/master/metricbeat/module/kubernetes/_meta/test/docs/02_objects/resourcequota.yaml

It will create

- named `rqtest` namespace, which will be assigned the resource quotas
- named `resources` resource quota, which will limit the ammount of CPU and memory that can be assigned to the namespace. (This settings won't be put to test) 
- `objects` resource quota, which will limit the quantity of objects that can be created at this namespace:
  - 3 Pods
  - 1 Configmap
  - 0 PersistentVolumeClaims
  - 1 ReplicaController
  - 1 Secret
  - 2 Services
  - 1 Service type LoadBalancer

- It will also create regular objects at that same namespace
  - 1 Service type LoadBalancer, that will succeed
  - 1 Service type LoadBalancer, that **will fail** due to exceeding Quota

- Copy binary and metricbeat assets to the playground pod. The module file targeting `ResourceQuota` should look like this:

```yaml
- module: kubernetes
  enabled: true
  metricsets:
    - state_resourcequota
  period: 10s
  hosts: ["kube-state-metrics.kube-system:8080"]
  in_cluster: true
```

- Execute metricbeat from the playground

You should see at elasticsearch/kibana:

Events that indicate a hard limit on services of type LoadBalancer

- `dataset`: `kubernetes.resourcequota`
- `kubernetes.resourcequota.name`:  `objects`
- `kubernetes.resourcequota.resource`: `services.loadbalancers`
- `kubernetes.resourcequota.quota`: 1
- `kubernetes.resourcequota.type`: `hard`

Events that indicate the number of service type LoadBalancer used

- `dataset`: `kubernetes.resourcequota`
- `kubernetes.resourcequota.name`:  `objects`
- `kubernetes.resourcequota.resource`: `services.loadbalancers`
- `kubernetes.resourcequota.quota`: 1
- `kubernetes.resourcequota.type`: `used`
