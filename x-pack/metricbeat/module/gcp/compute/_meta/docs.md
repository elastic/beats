Compute metricset to fetch metrics from [Compute Engine](https://cloud.google.com/compute/) Virtual Machines in Google Cloud Platform. No Monitoring or Logging agent is required in your instances to use this metricset.

The `compute` metricset contains all metrics exported from the [Stackdriver API](https://cloud.google.com/monitoring/api/metrics_gcp#gcp-compute). The field names are aligned to [Beats naming conventions](/extend/event-conventions.md) with minor modifications to their GCP metrics name counterpart.

Extra labels and metadata are also extracted using the [Compute API](https://cloud.google.com/compute/docs/reference/rest/v1/instances/get). This is enough to get most of the info associated with a metric like Compute labels and metadata and metric specific Labels.


## Labels [_labels]

Here is a list of labels collected by `compute` metricset depending on the type of metric being collected:

* `instance_name`: The name of the VM instance. Collected with:

    * `gcp.instance.firewall.*`
    * `gcp.instance.cpu.*`
    * `gcp.instance.disk.*`
    * `gcp.instance.memory.*`
    * `gcp.instance.network.*`
    * `gcp.instance.uptime`

* `device_name`: The name of the disk device. Collected with:

    * `gcp.instance.disk.*`

* `storage_type`: The storage type: `pd-standard`, `pd-ssd`, or `local-ssd`. Collected with:

    * `gcp.instance.disk.*`

* `device_type`: The disk type: `ephemeral` or `permanent`. Collected with:

    * `gcp.instance.disk.*`

* `loadBalanced`: Whether traffic was sent from an L3 loadbalanced IP address assigned to the VM. Traffic that is externally routed from the VM’s standard internal or external IP address, such as L7 loadbalanced traffic, is not considered to be loadbalanced in this metric. Collected with:

    * `gcp.instance.network.*`
