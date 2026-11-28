---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-module-opcua.html
applies_to:
  stack: beta
  serverless: beta
---

% This file is generated! See metricbeat/scripts/mage/docs_collector.go

# OPC UA module [metricbeat-module-opcua]

The OPC UA module reads values from [OPC UA](https://opcfoundation.org/about/opc-technologies/opc-ua/) servers, such as PLCs, gateways and industrial edge devices, using the [gopcua](https://github.com/gopcua/opcua) client.

The default metricset is `node`.


## Compatibility [_compatibility_opcua]

The OPC UA module is tested with the [OPC PLC](https://github.com/Azure-Samples/iot-edge-opc-plc) server 2.15.14.


## Security [_security_opcua]

The module supports the security policies None, Basic128Rsa15, Basic256, Basic256Sha256, Aes128_Sha256_RsaOaep and Aes256_Sha256_RsaPss with the security modes None, Sign and SignAndEncrypt. Sign and SignAndEncrypt need a client certificate and key, and the server must trust the client certificate.

Anonymous authentication is used by default. Set `username` and `password` for user name authentication.

The module connects to the endpoint configured in `hosts`, even when the server advertises a different host name in its endpoint descriptions, which is common for servers running in containers.


## Example configuration [_example_configuration]

The OPC UA module supports the standard configuration options that are described in [Modules](/reference/metricbeat/configuration-metricbeat.md). Here is an example configuration:

```yaml
metricbeat.modules:
- module: opcua
  metricsets: ["node"]
  period: 10s
  # OPC UA endpoints to read from. One session is opened per endpoint.
  hosts: ["opc.tcp://localhost:4840"]
  # Timeout for connecting and for each read request.
  #timeout: 10s

  # Nodes to read. The id uses the OPC UA node id format, for example
  # "ns=3;s=StepUp" or "ns=2;i=1001". The optional label is added to the event.
  nodes:
    - id: "ns=3;s=StepUp"
      label: "step"

  # Security policy: None, Basic128Rsa15, Basic256, Basic256Sha256,
  # Aes128_Sha256_RsaOaep or Aes256_Sha256_RsaPss.
  #security_policy: "None"
  # Security mode: None, Sign or SignAndEncrypt. Sign and SignAndEncrypt need
  # client_certificate and client_key.
  #security_mode: "None"

  # Credentials for user name authentication. Anonymous authentication is used
  # when username is empty.
  #username: ""
  #password: ""

  # PEM or DER encoded client certificate and private key.
  #client_certificate: ""
  #client_key: ""

  # Application URI of the client. It must match the URI in the client
  # certificate.
  #application_uri: "urn:elastic:metricbeat:opcua"
```


## Metricsets [_metricsets]

The following metricsets are available:

* [node](/reference/metricbeat/metricbeat-metricset-opcua-node.md)  {applies_to}`stack: beta`
