---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/exported-fields-tls_detailed.html
---

# Detailed TLS fields [exported-fields-tls_detailed]

Detailed TLS-specific event fields.

**`tls.client.x509.version`**
:   Version of x509 format.

type: keyword

example: 3


**`tls.client.x509.issuer.province`**
:   Province or region within country.

type: keyword


**`tls.client.x509.subject.province`**
:   Province or region within country.

type: keyword


**`tls.server.x509.version`**
:   Version of x509 format.

type: keyword

example: 3


**`tls.server.x509.issuer.province`**
:   Province or region within country.

type: keyword


**`tls.server.x509.subject.province`**
:   Province or region within country.

type: keyword


**`tls.detailed.version`**
:   The version of the TLS protocol used.

type: keyword

example: TLS 1.3


**`tls.detailed.resumption_method`**
:   If the session has been resumed, the underlying method used. One of "id" for TLS session ID or "ticket" for TLS ticket extension.

type: keyword


**`tls.detailed.client_certificate_requested`**
:   Whether the server has requested the client to authenticate itself using a client certificate.

type: boolean


**`tls.detailed.ocsp_response`**
:   The result of an OCSP request.

type: keyword


**`tls.detailed.client_hello.version`**
:   The version of the TLS protocol by which the client wishes to communicate during this session.

type: keyword


**`tls.detailed.client_hello.random`**
:   Random data used by the TLS protocol to generate the encryption key.

type: keyword


**`tls.detailed.client_hello.session_id`**
:   Unique number to identify the session for the corresponding connection with the client.

type: keyword


**`tls.detailed.client_hello.supported_compression_methods`**
:   The list of compression methods the client supports. See [https://www.iana.org/assignments/comp-meth-ids/comp-meth-ids.xhtml](https://www.iana.org/assignments/comp-meth-ids/comp-meth-ids.xhtml)

type: keyword



## extensions [_extensions]

The hello extensions provided by the client.

**`tls.detailed.client_hello.extensions.server_name_indication`**
:   List of hostnames

type: keyword


**`tls.detailed.client_hello.extensions.application_layer_protocol_negotiation`**
:   List of application-layer protocols the client is willing to use.

type: keyword


**`tls.detailed.client_hello.extensions.session_ticket`**
:   Length of the session ticket, if provided, or an empty string to advertise support for tickets.

type: keyword


**`tls.detailed.client_hello.extensions.supported_versions`**
:   List of TLS versions that the client is willing to use.

type: keyword


**`tls.detailed.client_hello.extensions.supported_groups`**
:   List of Elliptic Curve Cryptography (ECC) curve groups supported by the client.

type: keyword


**`tls.detailed.client_hello.extensions.signature_algorithms`**
:   List of signature algorithms that may be use in digital signatures.

type: keyword


**`tls.detailed.client_hello.extensions.ec_points_formats`**
:   List of Elliptic Curve (EC) point formats. Indicates the set of point formats that the client can parse.

type: keyword



## status_request [_status_request]

Status request made to the server.

**`tls.detailed.client_hello.extensions.status_request.type`**
:   The type of the status request. Always "ocsp" if present.

type: keyword


**`tls.detailed.client_hello.extensions.status_request.responder_id_list_length`**
:   The length of the list of trusted responders.

type: short


**`tls.detailed.client_hello.extensions.status_request.request_extensions`**
:   The number of certificate extensions for the request.

type: short


**`tls.detailed.client_hello.extensions._unparsed_`**
:   List of extensions that were left unparsed by Packetbeat.

type: keyword


**`tls.detailed.server_hello.version`**
:   The version of the TLS protocol that is used for this session. It is the highest version supported by the server not exceeding the version requested in the client hello.

type: keyword


**`tls.detailed.server_hello.random`**
:   Random data used by the TLS protocol to generate the encryption key.

type: keyword


**`tls.detailed.server_hello.selected_compression_method`**
:   The compression method selected by the server from the list provided in the client hello.

type: keyword


**`tls.detailed.server_hello.session_id`**
:   Unique number to identify the session for the corresponding connection with the client.

type: keyword



## extensions [_extensions_2]

The hello extensions provided by the server.

**`tls.detailed.server_hello.extensions.application_layer_protocol_negotiation`**
:   Negotiated application layer protocol

type: keyword


**`tls.detailed.server_hello.extensions.session_ticket`**
:   Used to announce that a session ticket will be provided by the server. Always an empty string.

type: keyword


**`tls.detailed.server_hello.extensions.supported_versions`**
:   Negotiated TLS version to be used.

type: keyword


**`tls.detailed.server_hello.extensions.ec_points_formats`**
:   List of Elliptic Curve (EC) point formats. Indicates the set of point formats that the server can parse.

type: keyword



## status_request [_status_request_2]

Status request made to the server.

**`tls.detailed.server_hello.extensions.status_request.response`**
:   Whether a certificate status request response was made.

type: boolean


**`tls.detailed.server_hello.extensions._unparsed_`**
:   List of extensions that were left unparsed by Packetbeat.

type: keyword


**`tls.detailed.server_certificate_chain`**
:   Chain of trust for the server certificate.

type: array


**`tls.detailed.client_certificate_chain`**
:   Chain of trust for the client certificate.

type: array


**`tls.detailed.alert_types`**
:   An array containing the TLS alert type for every alert received.

type: keyword


