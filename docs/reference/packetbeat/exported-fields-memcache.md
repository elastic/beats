---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/exported-fields-memcache.html
---

# Memcache fields [exported-fields-memcache]

Memcached-specific event fields

**`memcache.protocol_type`**
:   The memcache protocol implementation. The value can be "binary" for binary-based, "text" for text-based, or "unknown" for an unknown memcache protocol type.

type: keyword


**`memcache.request.line`**
:   The raw command line for unknown commands ONLY.

type: keyword


**`memcache.request.command`**
:   The memcache command being requested in the memcache text protocol. For example "set" or "get". The binary protocol opcodes are translated into memcache text protocol commands.

type: keyword


**`memcache.response.command`**
:   Either the text based protocol response message type or the name of the originating request if binary protocol is used.

type: keyword


**`memcache.request.type`**
:   The memcache command classification. This value can be "UNKNOWN", "Load", "Store", "Delete", "Counter", "Info", "SlabCtrl", "LRUCrawler", "Stats", "Success", "Fail", or "Auth".

type: keyword


**`memcache.response.type`**
:   The memcache command classification. This value can be "UNKNOWN", "Load", "Store", "Delete", "Counter", "Info", "SlabCtrl", "LRUCrawler", "Stats", "Success", "Fail", or "Auth". The text based protocol will employ any of these, whereas the binary based protocol will mirror the request commands only (see `memcache.response.status` for binary protocol).

type: keyword


**`memcache.response.error_msg`**
:   The optional error message in the memcache response (text based protocol only).

type: keyword


**`memcache.request.opcode`**
:   The binary protocol message opcode name.

type: keyword


**`memcache.response.opcode`**
:   The binary protocol message opcode name.

type: keyword


**`memcache.request.opcode_value`**
:   The binary protocol message opcode value.

type: long


**`memcache.response.opcode_value`**
:   The binary protocol message opcode value.

type: long


**`memcache.request.opaque`**
:   The binary protocol opaque header value used for correlating request with response messages.

type: long


**`memcache.response.opaque`**
:   The binary protocol opaque header value used for correlating request with response messages.

type: long


**`memcache.request.vbucket`**
:   The vbucket index sent in the binary message.

type: long


**`memcache.response.status`**
:   The textual representation of the response error code (binary protocol only).

type: keyword


**`memcache.response.status_code`**
:   The status code value returned in the response (binary protocol only).

type: long


**`memcache.request.keys`**
:   The list of keys sent in the store or load commands.

type: array


**`memcache.response.keys`**
:   The list of keys returned for the load command (if present).

type: array


**`memcache.request.count_values`**
:   The number of values found in the memcache request message. If the command does not send any data, this field is missing.

type: long


**`memcache.response.count_values`**
:   The number of values found in the memcache response message. If the command does not send any data, this field is missing.

type: long


**`memcache.request.values`**
:   The list of base64 encoded values sent with the request (if present).

type: array


**`memcache.response.values`**
:   The list of base64 encoded values sent with the response (if present).

type: array


**`memcache.request.bytes`**
:   The byte count of the values being transferred.

type: long

format: bytes


**`memcache.response.bytes`**
:   The byte count of the values being transferred.

type: long

format: bytes


**`memcache.request.delta`**
:   The counter increment/decrement delta value.

type: long


**`memcache.request.initial`**
:   The counter increment/decrement initial value parameter (binary protocol only).

type: long


**`memcache.request.verbosity`**
:   The value of the memcache "verbosity" command.

type: long


**`memcache.request.raw_args`**
:   The text protocol raw arguments for the "stats …​" and "lru crawl …​" commands.

type: keyword


**`memcache.request.source_class`**
:   The source class id in *slab reassign* command.

type: long


**`memcache.request.dest_class`**
:   The destination class id in *slab reassign* command.

type: long


**`memcache.request.automove`**
:   The automove mode in the *slab automove* command expressed as a string. This value can be "standby"(=0), "slow"(=1), "aggressive"(=2), or the raw value if the value is unknown.

type: keyword


**`memcache.request.flags`**
:   The memcache command flags sent in the request (if present).

type: long


**`memcache.response.flags`**
:   The memcache message flags sent in the response (if present).

type: long


**`memcache.request.exptime`**
:   The data expiry time in seconds sent with the memcache command (if present). If the value is <30 days, the expiry time is relative to "now", or else it is an absolute Unix time in seconds (32-bit).

type: long


**`memcache.request.sleep_us`**
:   The sleep setting in microseconds for the *lru_crawler sleep* command.

type: long


**`memcache.response.value`**
:   The counter value returned by a counter operation.

type: long


**`memcache.request.noreply`**
:   Set to true if noreply was set in the request. The `memcache.response` field will be missing.

type: boolean


**`memcache.request.quiet`**
:   Set to true if the binary protocol message is to be treated as a quiet message.

type: boolean


**`memcache.request.cas_unique`**
:   The CAS (compare-and-swap) identifier if present.

type: long


**`memcache.response.cas_unique`**
:   The CAS (compare-and-swap) identifier to be used with CAS-based updates (if present).

type: long


**`memcache.response.stats`**
:   The list of statistic values returned. Each entry is a dictionary with the fields "name" and "value".

type: array


**`memcache.response.version`**
:   The returned memcache version string.

type: keyword


