---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/exported-fields-dns.html
---

# DNS fields [exported-fields-dns]

DNS-specific event fields.

**`dns.flags.authoritative`**
:   A DNS flag specifying that the responding server is an authority for the domain name used in the question.

type: boolean


**`dns.flags.recursion_available`**
:   A DNS flag specifying whether recursive query support is available in the name server.

type: boolean


**`dns.flags.recursion_desired`**
:   A DNS flag specifying that the client directs the server to pursue a query recursively. Recursive query support is optional.

type: boolean


**`dns.flags.authentic_data`**
:   A DNS flag specifying that the recursive server considers the response authentic.

type: boolean


**`dns.flags.checking_disabled`**
:   A DNS flag specifying that the client disables the server signature validation of the query.

type: boolean


**`dns.flags.truncated_response`**
:   A DNS flag specifying that only the first 512 bytes of the reply were returned.

type: boolean


**`dns.question.etld_plus_one`**
:   The effective top-level domain (eTLD) plus one more label. For example, the eTLD+1 for "foo.bar.golang.org." is "golang.org.". The data for determining the eTLD comes from an embedded copy of the data from [http://publicsuffix.org](http://publicsuffix.org).

example: amazon.co.uk.


**`dns.answers_count`**
:   The number of resource records contained in the `dns.answers` field.

type: long


**`dns.authorities`**
:   An array containing a dictionary for each authority section from the answer.

type: object


**`dns.authorities_count`**
:   The number of resource records contained in the `dns.authorities` field. The `dns.authorities` field may or may not be included depending on the configuration of Packetbeat.

type: long


**`dns.authorities.name`**
:   The domain name to which this resource record pertains.

example: example.com.


**`dns.authorities.type`**
:   The type of data contained in this resource record.

example: NS


**`dns.authorities.class`**
:   The class of DNS data contained in this resource record.

example: IN


**`dns.additionals`**
:   An array containing a dictionary for each additional section from the answer.

type: object


**`dns.additionals_count`**
:   The number of resource records contained in the `dns.additionals` field. The `dns.additionals` field may or may not be included depending on the configuration of Packetbeat.

type: long


**`dns.additionals.name`**
:   The domain name to which this resource record pertains.

example: example.com.


**`dns.additionals.type`**
:   The type of data contained in this resource record.

example: NS


**`dns.additionals.class`**
:   The class of DNS data contained in this resource record.

example: IN


**`dns.additionals.ttl`**
:   The time interval in seconds that this resource record may be cached before it should be discarded. Zero values mean that the data should not be cached.

type: long


**`dns.additionals.data`**
:   The data describing the resource. The meaning of this data depends on the type and class of the resource record.


**`dns.opt.version`**
:   The EDNS version.

example: 0


**`dns.opt.do`**
:   If set, the transaction uses DNSSEC.

type: boolean


**`dns.opt.ext_rcode`**
:   Extended response code field.

example: BADVERS


**`dns.opt.udp_size`**
:   Requestor’s UDP payload size (in bytes).

type: long


