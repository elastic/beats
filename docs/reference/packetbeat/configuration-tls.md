---
navigation_title: "TLS"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/configuration-tls.html
---

# Capture TLS traffic [configuration-tls]


TLS is a cryptographic protocol that provides secure communications on top of an existing application protocol, like HTTP or MySQL.

Packetbeat intercepts the initial handshake in a TLS connection and extracts useful information that helps operators diagnose problems and strengthen the security of their network and systems. It does not decrypt any information from the encapsulated protocol, nor does it reveal any sensitive information such as cryptographic keys. TLS versions 1.0 to 1.3 are supported.

It works by intercepting the client and server "hello" messages, which contain the negotiated parameters for the connection such as cryptographic ciphers and protocol versions. It can also intercept TLS alerts, which are sent by one of the parties to signal a problem with the negotiation, such as an expired certificate or a cryptographic error.

An example of indexed event:

```json
"tls": {
    "client": {
      "supported_ciphers": [
        "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
        "TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384",
        "TLS_RSA_WITH_3DES_EDE_CBC_SHA",
        "TLS_EMPTY_RENEGOTIATION_INFO_SCSV"
      ],
      "ja3": "e6573e91e6eb777c0933c5b8f97f10cd",
      "server_name": "example.net"
    },
    "server": {
      "subject": "CN=www.example.org,OU=Technology,O=Internet Corporation for Assigned Names and Numbers,L=Los Angeles,ST=California,C=US",
      "issuer": "CN=DigiCert SHA2 Secure Server CA,O=DigiCert Inc,C=US",
      "not_before": "2018-11-28T00:00:00.000Z",
      "not_after": "2020-12-02T12:00:00.000Z",
      "hash": {
        "sha1": "7BB698386970363D2919CC5772846984FFD4A889"
      }
    },
    "version": "1.2",
    "version_protocol": "tls",
    "cipher": "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
    "established": true,
    "next_protocol": "h2",
    "detailed": {
      "server_certificate": {
        "subject": {
          "common_name": "www.example.org",
          "country": "US",
          "organization": "Internet Corporation for Assigned Names and Numbers",
          "organizational_unit": "Technology",
          "locality": "Los Angeles",
          "province": "California"
        },
        "not_after": "2020-12-02T12:00:00.000Z",
        "public_key_size": 2048,
        "alternative_names": [
          "www.example.org",
          "example.com",
          "example.edu",
          "example.net",
          "example.org",
          "www.example.com",
          "www.example.edu",
          "www.example.net"
        ],
        "signature_algorithm": "SHA256-RSA",
        "version": 3,
        "issuer": {
          "organization": "DigiCert Inc",
          "common_name": "DigiCert SHA2 Secure Server CA",
          "country": "US"
        },
        "not_before": "2018-11-28T00:00:00.000Z",
        "public_key_algorithm": "RSA",
        "serial_number": "21020869104500376438182461249190639870"
      },
      "server_certificate_chain": [
        {
          "public_key_algorithm": "RSA",
          "not_before": "2013-03-08T12:00:00.000Z",
          "not_after": "2023-03-08T12:00:00.000Z",
          "version": 3,
          "serial_number": "2646203786665923649276728595390119057",
          "issuer": {
            "organizational_unit": "www.digicert.com",
            "common_name": "DigiCert Global Root CA",
            "country": "US",
            "organization": "DigiCert Inc"
          },
          "subject": {
            "country": "US",
            "organization": "DigiCert Inc",
            "common_name": "DigiCert SHA2 Secure Server CA"
          },
          "public_key_size": 2048,
          "signature_algorithm": "SHA256-RSA"
        },
        {
          "public_key_algorithm": "RSA",
          "subject": {
            "common_name": "DigiCert Global Root CA",
            "country": "US",
            "organization": "DigiCert Inc",
            "organizational_unit": "www.digicert.com"
          },
          "issuer": {
            "country": "US",
            "organization": "DigiCert Inc",
            "organizational_unit": "www.digicert.com",
            "common_name": "DigiCert Global Root CA"
          },
          "signature_algorithm": "SHA1-RSA",
          "serial_number": "10944719598952040374951832963794454346",
          "not_before": "2006-11-10T00:00:00.000Z",
          "not_after": "2031-11-10T00:00:00.000Z",
          "public_key_size": 2048,
          "version": 3
        }
      ],
      "client_certificate_requested": false,
      "version": "TLS 1.2",
      "client_hello": {
        "version": "3.3",
        "supported_compression_methods": [
          "NULL"
        ],
        "extensions": {
          "ec_points_formats": [
            "uncompressed"
          ],
          "supported_groups": [
            "x25519",
            "secp256r1",
            "secp384r1"
          ],
          "signature_algorithms": [
            "rsa_pkcs1_sha512",
            "ecdsa_secp521r1_sha512",
            "(unknown:0xefef)",
            "rsa_pkcs1_sha384",
            "ecdsa_secp384r1_sha384",
            "rsa_pkcs1_sha256",
            "ecdsa_secp256r1_sha256",
            "(unknown:0xeeee)",
            "(unknown:0xeded)",
            "(unknown:0x0301)",
            "(unknown:0x0303)",
            "rsa_pkcs1_sha1",
            "ecdsa_sha1"
          ],
          "application_layer_protocol_negotiation": [
            "h2",
            "http/1.1"
          ],
          "server_name_indication": [
            "example.net"
          ]
        }
      },
      "server_hello": {
        "version": "3.3",
        "session_id": "23bb2aed5d215e1228220b0a51d7aa220785e9e4b83b4f430229117971e9913f",
        "selected_compression_method": "NULL",
        "extensions": {
          "application_layer_protocol_negotiation": [
            "h2"
          ],
          "_unparsed_": [
            "renegotiation_info",
            "server_name_indication"
          ],
          "ec_points_formats": [
            "uncompressed",
            "ansiX962_compressed_prime",
            "ansiX962_compressed_char2"
          ]
        }
      }
    }
  }
```

The TLS events generated by Packetbeat follow the Elastic Common Schema (ECS) format. See [ECS TLS fields](ecs://reference/ecs-tls.md) for a description of the populated fields.

Detailed information that is not defined in ECS is added under the `tls.detailed` key. The [`include_detailed_fields`](#include_detailed_fields) configuration flag is used to control whether this information is exported.

The fields under `tls.detailed.client_hello` contain the algorithms and extensions supported by the client, as well as the maximum TLS version it supports.

Fields under `tls.detailed.server_hello` contain the final settings for the TLS session: The selected cipher, compression method, TLS version to use and other extensions such as application layer protocol negotiation (ALPN).

See the [*Detailed TLS fields*](/reference/packetbeat/exported-fields-tls_detailed.md) section for more information.

The following settings are specific to the TLS protocol. Here is a sample configuration for the `tls` section of the `packetbeat.yml` config file:

```yaml
packetbeat.protocols:
- type: tls
  send_certificates: true
  include_raw_certificates: false
  include_detailed_fields: true
  fingerprints: [ md5, sha1, sha256 ]
```

## Configuration options [_configuration_options_12]

The `send_certificates` and `include_detailed_fields` settings are useful for limiting the amount of data Packetbeat indexes, as multiple certificates are usually exchanged in a single transaction, and those can take a considerable amount of storage.

Also see [Common protocol options](/reference/packetbeat/common-protocol-options.md).

### `send_certificates` [_send_certificates]

This setting causes information about the certificates presented by the client and server to be included in the detailed fields. The server’s certificate is indexed under `tls.detailed.server_certificate` and its certification chain under `tls.detailed.server_certificate_chain`. For the client, the `client_certificate` and `client_certificate_chain` fields are used. The default is true.


### `include_raw_certificates` [_include_raw_certificates]

You can set `include_raw_certificates` to include the raw certificate chains encoded in PEM format, under the `tls.server.certificate_chain` and `tls.client.certificate_chain` fields. The default is false.


### `include_detailed_fields` [include_detailed_fields]

Controls whether the [*Detailed TLS fields*](/reference/packetbeat/exported-fields-tls_detailed.md) are added to exported documents. When set to `false`, only [ECS TLS fields](ecs://reference/ecs-tls.md) are included. The default is `true`.


### `fingerprints` [_fingerprints]

Defines a list of hash algorithms to calculate the certificate’s fingerprints. Valid values are `sha1`, `sha256` and `md5`.

The default is to output SHA-1 fingerprints.



