Dataproc metricset fetches metrics from [Dataproc](https://cloud.google.com/dataproc/) in Google Cloud Platform.

The `dataproc` metricset contains all metrics exported from the [GCP Dataproc Monitoring API](https://cloud.google.com/monitoring/api/metrics_gcp#gcp-dataproc). The field names are aligned to [Beats naming conventions](/extend/event-conventions.md) with minor modifications to their GCP metrics name counterpart.

You can specify a single region to fetch metrics like `us-central1`. Be aware that GCP Storage does not use zones so `us-central1-a` will return nothing. If no region is specified, metrics are returned from all buckets.
