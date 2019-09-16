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

- TODO point to kubernetes tests setup
- TODO point to `ResourceQuota` objects creation
- TODO include here expected kube-state-metrics
- TODO include here expected elastic events









