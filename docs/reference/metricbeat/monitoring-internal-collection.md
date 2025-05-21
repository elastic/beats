---
navigation_title: "Use internal collection"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/monitoring-internal-collection.html
---

# Use internal collection to send monitoring data [monitoring-internal-collection]


Use internal collectors to send {{beats}} monitoring data directly to your monitoring cluster. Or as an alternative to internal collection, use [Use {{metricbeat}} collection](/reference/metricbeat/monitoring-metricbeat-collection.md). The benefit of using internal collection instead of {{metricbeat}} is that you have fewer pieces of software to install and maintain.

1. Create an API key or user that has appropriate authority to send system-level monitoring data to {{es}}. For example, you can use the built-in `beats_system` user or assign the built-in `beats_system` role to another user. For more information on the required privileges, see [Create a *monitoring* user](/reference/metricbeat/privileges-to-publish-monitoring.md). For more information on how to use API keys, see [*Grant access using API keys*](/reference/metricbeat/beats-api-keys.md).
2. Add the `monitoring` settings in the Metricbeat configuration file. If you configured the {{es}} output and want to send Metricbeat monitoring events to the same {{es}} cluster, specify the following minimal configuration:

    ```yaml
    monitoring:
      enabled: true
      elasticsearch:
        api_key:  id:api_key <1>
        username: beats_system
        password: somepassword
    ```

    1. Specify one of `api_key` or `username`/`password`.


    If you want to send monitoring events to an [{{ecloud}}](https://cloud.elastic.co/) monitoring cluster, you can use two simpler settings. When defined, these settings overwrite settings from other parts in the configuration. For example:

    ```yaml
    monitoring:
      enabled: true
      cloud.id: 'staging:dXMtZWFzdC0xLmF3cy5mb3VuZC5pbyRjZWM2ZjI2MWE3NGJmMjRjZTMzYmI4ODExYjg0Mjk0ZiRjNmMyY2E2ZDA0MjI0OWFmMGNjN2Q3YTllOTYyNTc0Mw=='
      cloud.auth: 'elastic:YOUR_PASSWORD'
    ```

    If you configured a different output, such as {{ls}} or you want to send Metricbeat monitoring events to a separate {{es}} cluster (referred to as the *monitoring cluster*), you must specify additional configuration options. For example:

    ```yaml
    monitoring:
      enabled: true
      cluster_uuid: PRODUCTION_ES_CLUSTER_UUID <1>
      elasticsearch:
        hosts: ["https://example.com:9200", "https://example2.com:9200"] <2>
        api_key:  id:api_key <3>
        username: beats_system
        password: somepassword
    ```

    1. This setting identifies the {{es}} cluster under which the monitoring data for this Metricbeat instance will appear in the Stack Monitoring UI. To get a cluster’s `cluster_uuid`, call the `GET /` API against that production cluster.
    2. This setting identifies the hosts and port numbers of {{es}} nodes that are part of the monitoring cluster.
    3. Specify one of `api_key` or `username`/`password`.


    If you want to use PKI authentication to send monitoring events to {{es}}, you must specify a different set of configuration options. For example:

    ```yaml
    monitoring:
      enabled: true
      cluster_uuid: PRODUCTION_ES_CLUSTER_UUID
      elasticsearch:
        hosts: ["https://example.com:9200", "https://example2.com:9200"]
        username: ""
        ssl.certificate_authorities: ["/etc/pki/root/ca.pem"]
        ssl.certificate: "/etc/pki/client/cert.pem"
        ssl.key: "/etc/pki/client/cert.key"
    ```

    You must specify the `username` as `""` explicitly so that the username from the client certificate (`CN`) is used. See [SSL](/reference/metricbeat/configuration-ssl.md) for more information about SSL settings.

3. Start Metricbeat.
4. [View the monitoring data in {{kib}}](docs-content://deploy-manage/monitor/stack-monitoring/kibana-monitoring-data.md).


