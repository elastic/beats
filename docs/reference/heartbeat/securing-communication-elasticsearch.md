---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/heartbeat/current/securing-communication-elasticsearch.html
---

# Secure communication with Elasticsearch [securing-communication-elasticsearch]

When sending data to a secured cluster through the `elasticsearch` output, Heartbeat can use any of the following authentication methods:

* Basic authentication credentials (username and password).
* Token-based API authentication.
* A client certificate.

Authentication is specified in the Heartbeat configuration file:

* To use **basic authentication**, specify the `username` and `password` settings under `output.elasticsearch`. For example:

    ```yaml
    output.elasticsearch:
      hosts: ["https://myEShost:9200"]
      username: "heartbeat_writer" <1>
      password: "YOUR_PASSWORD" <2>
    ```

    1. This user needs the privileges required to publish events to {{es}}. To create a user like this, see [Create a *publishing* user](/reference/heartbeat/privileges-to-publish-events.md).
    2. This example shows a hard-coded password, but you should store sensitive values in the [secrets keystore](/reference/heartbeat/keystore.md).

* To use token-based **API key authentication**, specify the `api_key` under `output.elasticsearch`. For example:

    ```yaml
    output.elasticsearch:
      hosts: ["https://myEShost:9200"]
      api_key: "ZCV7VnwBgnX0T19fN8Qe:KnR6yE41RrSowb0kQ0HWoA" <1>
    ```

    1. This API key must have the privileges required to publish events to {{es}}. To create an API key like this, see [*Grant access using API keys*](/reference/heartbeat/beats-api-keys.md).


* To use **Public Key Infrastructure (PKI) certificates** to authenticate users, specify the `certificate` and `key` settings under `output.elasticsearch`. For example:

    ```yaml
    output.elasticsearch:
      hosts: ["https://myEShost:9200"]
      ssl.certificate: "/etc/pki/client/cert.pem" <1>
      ssl.key: "/etc/pki/client/cert.key" <2>
    ```

    1. The path to the certificate for SSL client authentication
    2. The client certificate key


    These settings assume that the distinguished name (DN) in the certificate is mapped to the appropriate roles in the `role_mapping.yml` file on each node in the {{es}} cluster. For more information, see [Using role mapping files](docs-content://deploy-manage/users-roles/cluster-or-deployment-auth/mapping-users-groups-to-roles.md#mapping-roles-file).

    By default, Heartbeat uses the list of trusted certificate authorities (CA) from the operating system where Heartbeat is running. If the certificate authority that signed your node certificates is not in the host system’s trusted certificate authorities list, you need to add the path to the `.pem` file that contains your CA’s certificate to the Heartbeat configuration. This will configure Heartbeat to use a specific list of CA certificates instead of the default list from the OS.

    Here is an example configuration:

    ```yaml
    output.elasticsearch:
      hosts: ["https://myEShost:9200"]
      ssl.certificate_authorities: <1>
        - /etc/pki/my_root_ca.pem
        - /etc/pki/my_other_ca.pem
      ssl.certificate: "/etc/pki/client.pem" <2>
      ssl.key: "/etc/pki/key.pem" <3>
    ```

    1. Specify the path to the local `.pem` file that contains your Certificate Authority’s certificate. This is needed if you use your own CA to sign your node certificates.
    2. The path to the certificate for SSL client authentication
    3. The client certificate key


    ::::{note}
    For any given connection, the SSL/TLS certificates must have a subject that matches the value specified for `hosts`, or the SSL handshake fails. For example, if you specify `hosts: ["foobar:9200"]`, the certificate MUST include `foobar` in the subject (`CN=foobar`) or as a subject alternative name (SAN). Make sure the hostname resolves to the correct IP address. If no DNS is available, then you can associate the IP address with your hostname in `/etc/hosts` (on Unix) or `C:\Windows\System32\drivers\etc\hosts` (on Windows).
    ::::




## Learn more about secure communication [securing-communication-learn-more]

More information on sending data to a secured cluster is available in the configuration reference:

* [Elasticsearch](/reference/heartbeat/elasticsearch-output.md)
* [SSL](/reference/heartbeat/configuration-ssl.md)

