Monitoring needs to be enabled in the cluster:

PUT _cluster/settings
{
  "persistent": {
    "xpack.monitoring.collection.enabled": true
  }
}


Get cluster UUID

GET /
