# Kube-state-metrics/Cronjob

## Version history

- July 2019, `v1.7.0`

## Resources

Docs for 1.7 release of `kube-state-metrics` cronjobs can be found here:
https://github.com/kubernetes/kube-state-metrics/blob/release-1.7/docs/cronjob-metrics.md

## Metrics insight

    - kube_cronjob_labels{namespace,cronjob,label_run} Gauge
        
        Need to modify prometheus thing to be able to read this
        *Not added yet!*

    - kube_cronjob_info{namespace,cronjob,schedule=,concurrency_policy} Gauge

    - kube_cronjob_created{namespace,cronjob} Gauge
        
        Unix time

    - kube_cronjob_status_active{namespace,cronjob} Gauge
        
        Contains the number of active pods working for this cronjob, will usually be 0 or 1, but I think this could be more than 1

    - kube_cronjob_status_last_schedule_time{namespace,cronjob} Gauge
        
        Unix time

    - kube_cronjob_spec_suspend{namespace,cronjob} Gauge

    - kube_cronjob_spec_starting_deadline_seconds{namespace,cronjob} Gauge
            
        RE-TEST -- add deadline

    - kube_cronjob_next_schedule_time{namespace,cronjob} Gauge
        
        Unix time

    - kube_cronjob_annotations{namespace="default",cronjob="bye"}
        
        Marked as experimental
        *Not added yet!*

## Setup environment for manual tests


Instructions for `Linux` and `Kind`. If you are using any other environment and need to adapt these instructions, please, update accordingly.

Deploy metricbeat pre-baked pod as recommended by the docs.

Kubernetes YAML chunk should look like:

```yaml
- module: kubernetes
  enabled: true
  metricsets:
    - state_cronjob
  period: 10s
  hosts: ["kube-state-metrics:8080"]
```

Deploy kube-state-metrics. You can find the manifests [here](https://github.com/kubernetes/kube-state-metrics/tree/release-1.7/kubernetes)








