Storage metricset fetches metrics from [Storage](https://cloud.google.com/storage/) in Google Cloud Platform.

The `storage` metricset contains all metrics exported from the [GCP Storage Monitoring API](https://cloud.google.com/monitoring/api/metrics_gcp#gcp-storage). The field names are aligned to [Beats naming conventions](/extend/event-conventions.md) with minor modifications to their GCP metrics name counterpart.

You can specify a single region to fetch metrics like `us-central1`. Be aware that GCP Storage does not use zones so `us-central1-a` will return nothing. If no region is specified, metrics are returned from all buckets.
