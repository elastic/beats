The OPC UA module reads values from [OPC UA](https://opcfoundation.org/about/opc-technologies/opc-ua/) servers, such as PLCs, gateways and industrial edge devices, using the [gopcua](https://github.com/gopcua/opcua) client.

The default metricset is `node`.


## Compatibility [_compatibility_opcua]

The OPC UA module is tested with the [OPC PLC](https://github.com/Azure-Samples/iot-edge-opc-plc) server 2.15.14.


## Security [_security_opcua]

The module supports the security policies None, Basic128Rsa15, Basic256, Basic256Sha256, Aes128_Sha256_RsaOaep and Aes256_Sha256_RsaPss with the security modes None, Sign and SignAndEncrypt. Sign and SignAndEncrypt need a client certificate and key, and the server must trust the client certificate.

Anonymous authentication is used by default. Set `username` and `password` for user name authentication.

The module connects to the endpoint configured in `hosts`, even when the server advertises a different host name in its endpoint descriptions, which is common for servers running in containers.
