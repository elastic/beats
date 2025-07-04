Firestore metricset fetches metrics from [Firestore](https://cloud.google.com/firestore/) in Google Cloud Platform.

The `firestore` metricset contains all metrics exported from the [GCP Firestore Monitoring API](https://cloud.google.com/monitoring/api/metrics_gcp#gcp-firestore). The field names are aligned to [Beats naming conventions](/extend/event-conventions.md) with minor modifications to their GCP metrics name counterpart.

You can specify a single region to fetch metrics like `us-central1`. Be aware that GCP Storage does not use zones so `us-central1-a` will return nothing. If no region is specified, metrics are returned from all buckets.
