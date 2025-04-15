---
navigation_title: "add_cloudfoundry_metadata"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/add-cloudfoundry-metadata.html
---

# Add Cloud Foundry metadata [add-cloudfoundry-metadata]


The `add_cloudfoundry_metadata` processor annotates each event with relevant metadata from Cloud Foundry applications. The events are annotated with Cloud Foundry metadata, only if the event contains a reference to a Cloud Foundry application (using field `cloudfoundry.app.id`) and the configured Cloud Foundry client is able to retrieve information for the application.

Each event is annotated with:

* Application Name
* Space ID
* Space Name
* Organization ID
* Organization Name

::::{note}
Pivotal Application Service and Tanzu Application Service include this metadata in all events from the firehose since version 2.8. In these cases the metadata in the events is used, and `add_cloudfoundry_metadata` processor doesn’t modify these fields.
::::


For efficient annotation, application metadata retrieved by the Cloud Foundry client is stored in a persistent cache on the filesystem under the `path.data` directory. This is done so the metadata can persist across restarts of Metricbeat. For control over this cache, use the `cache_duration` and `cache_retry_delay` settings.

```yaml
processors:
  - add_cloudfoundry_metadata:
      api_address: https://api.dev.cfdev.sh
      client_id: uaa-filebeat
      client_secret: verysecret
      ssl:
        verification_mode: none
      # To connect to Cloud Foundry over verified TLS you can specify a client and CA certificate.
      #ssl:
      #  certificate_authorities: ["/etc/pki/cf/ca.pem"]
      #  certificate:              "/etc/pki/cf/cert.pem"
      #  key:                      "/etc/pki/cf/cert.key"
```

It has the following settings:

`api_address`
:   (Optional) The URL of the Cloud Foundry API. It uses `http://api.bosh-lite.com` by default.

`doppler_address`
:   (Optional) The URL of the Cloud Foundry Doppler Websocket. It uses value from ${api_address}/v2/info by default.

`uaa_address`
:   (Optional) The URL of the Cloud Foundry UAA API. It uses value from ${api_address}/v2/info by default.

`rlp_address`
:   (Optional) The URL of the Cloud Foundry RLP Gateway. It uses value from ${api_address}/v2/info by default.

`client_id`
:   Client ID to authenticate with Cloud Foundry.

`client_secret`
:   Client Secret to authenticate with Cloud Foundry.

`cache_duration`
:   (Optional) Maximum amount of time to cache an application’s metadata. Defaults to 120 seconds.

`cache_retry_delay`
:   (Optional) Time to wait before trying to obtain an application’s metadata again in case of error. Defaults to 20 seconds.

`ssl`
:   (Optional) SSL configuration to use when connecting to Cloud Foundry.

