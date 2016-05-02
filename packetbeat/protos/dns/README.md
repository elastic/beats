#### UDP

**Parsing**

1. Attempt to decode each UDP packet.
2. If it succeeds, a transaction is sent.

**Error management**
* Debug information is printed if:
  * A packet fails to decode.

* Error Notes are published if:
  * Never

#### TCP

**Parsing**

1. Fetch the first two bytes of a message containing the length of the message ([RFC 1035](https://www.ietf.org/rfc/rfc1035.txt)).
2. Fill the buffer ```DnsStream.rawData``` with each new ```Parse```.
3. Once the buffer has the expected length (first two bytes), it is decoded and the message is published.

**Error management**
* Debug information is printed if:
  * A message has an unexpected length at any point of the transmission (```Parse```, ```GapInStream```, ```ReceivedFin```).
  * A message fails to decode.

* Error Notes are published if:
  * A response following a request (```dnsConnectionData.prevRequest```) fails to decode.
  * A response following a request (```dnsConnectionData.prevRequest```) has an unexpected length at any point of the transmission (```Parse```, ```GapInStream```, ```ReceivedFin```).

When response error Notes are linked to the previous request, the transaction is then published and removed from the cache (see ```publishResponseError()```).

#### TODO

**General**
* Publish an event with Notes when a Query or a lone Response cannot be decoded.
* Consider adding ICMP support to
     - correlate ICMP type 3, code 4 (datagram too big) with DNS messages,
     - correlate ICMP type 3, code 13 (administratively prohibited) or
       ICMP type 3, code 3 (port unreachable) with blocked DNS messages.
