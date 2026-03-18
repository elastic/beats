PubSub metricsetf fetches metrics from [Pub/Sub](https://cloud.google.com/pubsub/) topics and subscriptions in Google Cloud Platform.

The `pubsub` metricset contains all GA stage metrics exported from the [Stackdriver API](https://cloud.google.com/monitoring/api/metrics_gcp#gcp-pubsub). The field names are aligned to [Beats naming conventions](/extend/event-conventions.md) with minor modifications to their GCP metrics name counterpart.

No special permissions are needed apart from the ones detailed in the module section of the docs.
