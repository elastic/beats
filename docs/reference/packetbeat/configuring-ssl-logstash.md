---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/configuring-ssl-logstash.html
---

# Secure communication with Logstash [configuring-ssl-logstash]

You can use SSL mutual authentication to secure connections between Packetbeat and Logstash. This ensures that Packetbeat sends encrypted data to trusted Logstash servers only, and that the Logstash server receives data from trusted Packetbeat clients only.

To use SSL mutual authentication:

1. Create a certificate authority (CA) and use it to sign the certificates that you plan to use for Packetbeat and Logstash. Creating a correct SSL/TLS infrastructure is outside the scope of this document. There are many online resources available that describe how to create certificates.

    ::::{tip}
    If you are using {{security-features}}, you can use the [elasticsearch-certutil tool](elasticsearch://reference/elasticsearch/command-line-tools/certutil.md) to generate certificates.
    ::::

2. Configure Packetbeat to use SSL. In the `packetbeat.yml` config file, specify the following settings under `ssl`:

    * `certificate_authorities`: Configures Packetbeat to trust any certificates signed by the specified CA. If `certificate_authorities` is empty or not set, the trusted certificate authorities of the host system are used.
    * `certificate` and `key`: Specifies the certificate and key that Packetbeat uses to authenticate with Logstash.

        For example:

        ```yaml
        output.logstash:
          hosts: ["logs.mycompany.com:5044"]
          ssl.certificate_authorities: ["/etc/ca.crt"]
          ssl.certificate: "/etc/client.crt"
          ssl.key: "/etc/client.key"
        ```

        For more information about these configuration options, see [SSL](/reference/packetbeat/configuration-ssl.md).

3. Configure Logstash to use SSL. In the Logstash config file, specify the following settings for the [Beats input plugin for Logstash](logstash-docs-md://lsr/plugins-inputs-beats.md):

    * `ssl`: When set to true, enables Logstash to use SSL/TLS.
    * `ssl_certificate_authorities`: Configures Logstash to trust any certificates signed by the specified CA.
    * `ssl_certificate` and `ssl_key`: Specify the certificate and key that Logstash uses to authenticate with the client.
    * `ssl_verify_mode`: Specifies whether the Logstash server verifies the client certificate against the CA. You need to specify either `peer` or `force_peer` to make the server ask for the certificate and validate it. If you specify `force_peer`, and Packetbeat doesn’t provide a certificate, the Logstash connection will be closed. If you choose not to use [certutil](elasticsearch://reference/elasticsearch/command-line-tools/certutil.md), the certificates that you obtain must allow for both `clientAuth` and `serverAuth` if the extended key usage extension is present.

        For example:

        ```json
        input {
          beats {
            port => 5044
            ssl => true
            ssl_certificate_authorities => ["/etc/ca.crt"]
            ssl_certificate => "/etc/server.crt"
            ssl_key => "/etc/server.key"
            ssl_verify_mode => "force_peer"
          }
        }
        ```

        For more information about these options, see the [documentation for the Beats input plugin](logstash-docs-md://lsr/plugins-inputs-beats.md).



## Validate the Logstash server’s certificate [testing-ssl-logstash]

Before running Packetbeat, you should validate the Logstash server’s certificate. You can use `curl` to validate the certificate even though the protocol used to communicate with Logstash is not based on HTTP. For example:

```shell
curl -v --cacert ca.crt https://logs.mycompany.com:5044
```

If the test is successful, you’ll receive an empty response error:

```shell
* Rebuilt URL to: https://logs.mycompany.com:5044/
*   Trying 192.168.99.100...
* Connected to logs.mycompany.com (192.168.99.100) port 5044 (#0)
* TLS 1.2 connection using TLS_DHE_RSA_WITH_AES_256_CBC_SHA
* Server certificate: logs.mycompany.com
* Server certificate: mycompany.com
> GET / HTTP/1.1
> Host: logs.mycompany.com:5044
> User-Agent: curl/7.43.0
> Accept: */*
>
* Empty reply from server
* Connection #0 to host logs.mycompany.com left intact
curl: (52) Empty reply from server
```

The following example uses the IP address rather than the hostname to validate the certificate:

```shell
curl -v --cacert ca.crt https://192.168.99.100:5044
```

Validation for this test fails because the certificate is not valid for the specified IP address. It’s only valid for the `logs.mycompany.com`, the hostname that appears in the Subject field of the certificate.

```shell
* Rebuilt URL to: https://192.168.99.100:5044/
*   Trying 192.168.99.100...
* Connected to 192.168.99.100 (192.168.99.100) port 5044 (#0)
* WARNING: using IP address, SNI is being disabled by the OS.
* SSL: certificate verification failed (result: 5)
* Closing connection 0
curl: (51) SSL: certificate verification failed (result: 5)
```

See the [troubleshooting docs](/reference/packetbeat/ssl-client-fails.md) for info about resolving this issue.


## Test the Packetbeat to Logstash connection [_test_the_packetbeat_to_logstash_connection]

If you have Packetbeat running as a service, first stop the service. Then test your setup by running Packetbeat in the foreground so you can quickly see any errors that occur:

```sh
packetbeat -c packetbeat.yml -e -v
```

Any errors will be printed to the console. See the [troubleshooting docs](/reference/packetbeat/ssl-client-fails.md) for info about resolving common errors.

