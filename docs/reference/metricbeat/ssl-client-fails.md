---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/ssl-client-fails.html
---

# SSL client fails to connect to Logstash [ssl-client-fails]

The host running {{ls}} might be unreachable or the certificate may not be valid. To resolve your issue:

* Make sure that {{ls}} is running and you can connect to it. First, try to ping the {{ls}} host to verify that you can reach it from the host running Metricbeat. Then use either `nc` or `telnet` to make sure that the port is available. For example:

    ```shell
    ping <hostname or IP>
    telnet <hostname or IP> 5044
    ```

* Verify that the certificate is valid and that the hostname and IP match.

    ::::{tip}
    For testing purposes only, you can set `verification_mode: none` to disable hostname checking.
    ::::

* Use OpenSSL to test connectivity to the {{ls}} server and diagnose problems. See the [OpenSSL documentation](https://www.openssl.org/docs/manmaster/man1/openssl-s_client.md) for more info.
* Make sure that you have enabled SSL (set `ssl => true`) when configuring the [Beats input plugin for {{ls}}](logstash-docs-md://lsr/plugins-inputs-beats.md).

## Common SSL-Related Errors and Resolutions [_common_ssl_related_errors_and_resolutions]

Here are some common errors and ways to fix them:

* [tls: failed to parse private key](#failed-to-parse-private-key)
* [x509: cannot validate certificate](#cannot-validate-certificate)
* [getsockopt: no route to host](#getsockopt-no-route-to-host)
* [getsockopt: connection refused](#getsockopt-connection-refused)
* [No connection could be made because the target machine actively refused it](#target-machine-refused-connection)

### tls: failed to parse private key [failed-to-parse-private-key]

This might occur for a few reasons:

* The encrypted file is not recognized as an encrypted PEM block. Metricbeat tries to use the encrypted content as the final key, which fails.
* The file is correctly encrypted in a PEM block, but the decrypted content is not a key in a format that Metricbeat recognizes. The key must be encoded as PEM format.
* The passphrase is missing or has an error.


### x509: cannot validate certificate for <IP address> because it doesn’t contain any IP SANs [cannot-validate-certificate]

This happens because your certificate is only valid for the hostname present in the Subject field.

To resolve this problem, try one of these solutions:

* Create a DNS entry for the hostname mapping it to the server’s IP.
* Create an entry in `/etc/hosts` for the hostname. Or on Windows add an entry to `C:\Windows\System32\drivers\etc\hosts`.
* Re-create the server certificate and add a SubjectAltName (SAN) for the IP address of the server. This makes the server’s certificate valid for both the hostname and the IP address.


### getsockopt: no route to host [getsockopt-no-route-to-host]

This is not a SSL problem. It’s a networking problem. Make sure the two hosts can communicate.


### getsockopt: connection refused [getsockopt-connection-refused]

This is not a SSL problem. Make sure that {{ls}} is running and that there is no firewall blocking the traffic.


### No connection could be made because the target machine actively refused it [target-machine-refused-connection]

A firewall is refusing the connection. Check if a firewall is blocking the traffic on the client, the network, or the destination host.



