## 9.5.5 [beats-release-notes-9.5.5]



### Features and enhancements [beats-9.5.5-features-enhancements]


**All**

* Update Go to 1.26.8. [#53026](https://github.com/elastic/beats/pull/53026) 

**Filebeat**

* Add api_endpoint configuration option to gcp-pubsub input to allow overriding the default Pub/Sub API endpoint. [#53238](https://github.com/elastic/beats/pull/53238) [#53419](https://github.com/elastic/beats/pull/53419) [#53544](https://github.com/elastic/beats/pull/53544) [#53521](https://github.com/elastic/beats/pull/53521) [#53330](https://github.com/elastic/beats/issues/53330)
* Filestream take_over supports from_any_id to migrate from any previous filestream input. [#53238](https://github.com/elastic/beats/pull/53238) [#53419](https://github.com/elastic/beats/pull/53419) [#53544](https://github.com/elastic/beats/pull/53544) [#53521](https://github.com/elastic/beats/pull/53521) [#51307](https://github.com/elastic/beats/issues/51307)
* Reuse the UDP and unix datagram read buffer so each packet allocates only its own length. [#53238](https://github.com/elastic/beats/pull/53238) [#53419](https://github.com/elastic/beats/pull/53419) [#53544](https://github.com/elastic/beats/pull/53544) [#53521](https://github.com/elastic/beats/pull/53521) [#53330](https://github.com/elastic/beats/issues/53330)

**Libbeat**

* Add `append_fields` option to `add_kubernetes_metadata` processor to merge metadata into events that already have a `kubernetes` field without overwriting existing keys. [#53238](https://github.com/elastic/beats/pull/53238) [#53419](https://github.com/elastic/beats/pull/53419) [#53544](https://github.com/elastic/beats/pull/53544) [#53521](https://github.com/elastic/beats/pull/53521) [#53330](https://github.com/elastic/beats/issues/53330)
* Reduce per-event allocations in the add_agent_metadata processor by prebuilding the static metadata maps once at construction. [#53238](https://github.com/elastic/beats/pull/53238) [#53419](https://github.com/elastic/beats/pull/53419) [#53544](https://github.com/elastic/beats/pull/53544) [#53521](https://github.com/elastic/beats/pull/53521) [#53330](https://github.com/elastic/beats/issues/53330)

**Metricbeat**

* Persist additional cluster settings in the autoops_es cluster_settings metricset. [#53238](https://github.com/elastic/beats/pull/53238) [#53419](https://github.com/elastic/beats/pull/53419) [#53544](https://github.com/elastic/beats/pull/53544) [#53521](https://github.com/elastic/beats/pull/53521) [#53330](https://github.com/elastic/beats/issues/53330)
* Report archived cluster setting names in the autoops_es cluster_settings metricset. [#53238](https://github.com/elastic/beats/pull/53238) [#53419](https://github.com/elastic/beats/pull/53419) [#53544](https://github.com/elastic/beats/pull/53544) [#53521](https://github.com/elastic/beats/pull/53521) [#53330](https://github.com/elastic/beats/issues/53330)


### Fixes [beats-9.5.5-fixes]


**Auditbeat**

* Fix auditbeat file_integrity kprobe leak that could prevent FIM from starting when the running kernel does not match the first candidate BTF spec. [#53238](https://github.com/elastic/beats/pull/53238) [#53419](https://github.com/elastic/beats/pull/53419) [#53544](https://github.com/elastic/beats/pull/53544) [#53521](https://github.com/elastic/beats/pull/53521) [#53330](https://github.com/elastic/beats/issues/53330)
* Fix process misattribution in system/socket where TCP/UDP flows were attributed to wrong processes. [#53238](https://github.com/elastic/beats/pull/53238) [#53419](https://github.com/elastic/beats/pull/53419) [#53544](https://github.com/elastic/beats/pull/53544) [#53521](https://github.com/elastic/beats/pull/53521) [#53330](https://github.com/elastic/beats/issues/53330)

**Filebeat**

* Fix agentless Elasticsearch state store writing to the wrong input index. [#53238](https://github.com/elastic/beats/pull/53238) [#53419](https://github.com/elastic/beats/pull/53419) [#53544](https://github.com/elastic/beats/pull/53544) [#53521](https://github.com/elastic/beats/pull/53521) [#53330](https://github.com/elastic/beats/issues/53330)

  Inputs affected by the bug re-ingest once, from their configured initial lookback, the first time they read their own index.
  
* Reconnect the ETW input when its real-time session is stopped and restarted. [#53238](https://github.com/elastic/beats/pull/53238) [#53419](https://github.com/elastic/beats/pull/53419) [#53544](https://github.com/elastic/beats/pull/53544) [#53521](https://github.com/elastic/beats/pull/53521) [#49984](https://github.com/elastic/beats/issues/49984)

  When another controller stopped the real-time ETW session, for example
  `logman stop &lt;session&gt; -ets`, Windows reports the trace as finished and the
  input exited silently, staying down until Filebeat was restarted. The input
  now treats this as a lost session: it reports Degraded, retries creating or
  attaching to the session with a backoff of up to 30 seconds, and reports
  Running again once it has reconnected. Only a session that is not running
  is retried; an attempt that fails for a reason waiting cannot fix, such as
  insufficient privileges, reports Failed. A session that was created but
  could not be fully configured is now stopped rather than left running.
  Reading from an .etl file is not affected. A new `reconnects_total` metric
  counts the reconnection attempts.
  The changes to the shared ETW reader that make a session safe to reuse are
  listed under libbeat.
  
* Fix race condition in Kafka input when multiple inputs are configured. [#53238](https://github.com/elastic/beats/pull/53238) [#53419](https://github.com/elastic/beats/pull/53419) [#53544](https://github.com/elastic/beats/pull/53544) [#53521](https://github.com/elastic/beats/pull/53521) [#34919](https://github.com/elastic/beats/issues/34919)
* Fix Azure AD entity analytics accumulating stale registered owners and users on devices. [#53238](https://github.com/elastic/beats/pull/53238) [#53419](https://github.com/elastic/beats/pull/53419) [#53544](https://github.com/elastic/beats/pull/53544) [#53521](https://github.com/elastic/beats/pull/53521) [#53330](https://github.com/elastic/beats/issues/53330)

  Device.Merge was accumulating RegisteredOwners and RegisteredUsers across syncs
  instead of replacing them. Because the Graph API always returns the complete
  current list, the correct behaviour is replacement. Stale entries persisted
  indefinitely, eventually bloating the kvstore.
  
  Additionally, registered owner and user lookups were issued for deleted devices,
  causing noisy 404 errors from the Graph API.
  
* Fix Active Directory entity analytics panic when ssl.certificate_authorities is set. [#53238](https://github.com/elastic/beats/pull/53238) [#53419](https://github.com/elastic/beats/pull/53419) [#53544](https://github.com/elastic/beats/pull/53544) [#53521](https://github.com/elastic/beats/pull/53521) [#53330](https://github.com/elastic/beats/issues/53330)
* Fix EntraID entity analytics collection behavior for deleted devices when using minimal state mode. [#53238](https://github.com/elastic/beats/pull/53238) [#53419](https://github.com/elastic/beats/pull/53419) [#53544](https://github.com/elastic/beats/pull/53544) [#53521](https://github.com/elastic/beats/pull/53521) [#20](https://github.com/elastic/entcollect/issues/20)
* Respond to http_endpoint requests that contain no events instead of closing the connection. [#53238](https://github.com/elastic/beats/pull/53238) [#53419](https://github.com/elastic/beats/pull/53419) [#53544](https://github.com/elastic/beats/pull/53544) [#53521](https://github.com/elastic/beats/pull/53521) [#53330](https://github.com/elastic/beats/issues/53330)

  Since 8.9 the http_endpoint input wrote a zero HTTP status code when a
  request decoded to no events. net/http rejects status codes below 100, so
  the response panicked and the connection was closed with nothing written:
  clients saw an empty reply and reverse proxies in front of the endpoint
  turned it into a 502. Requests that hold no events are now answered with
  the configured response_code and response_body, as they were before 8.9.
  This covers an empty or nested-empty JSON array, an array holding only
  non-object elements, a whitespace-only body, and a program that returns an
  empty array. A request with no body at all is still rejected with 406, and
  that rejection no longer depends on whether max_body_bytes,
  max_in_flight_bytes or tracer is configured. Invalid response_code and
  options_response_code values are now rejected at startup.
  
* Fix httpjson rate limiter marking input Degraded when rate-limit headers are absent. [#53238](https://github.com/elastic/beats/pull/53238) [#53419](https://github.com/elastic/beats/pull/53419) [#53544](https://github.com/elastic/beats/pull/53544) [#53521](https://github.com/elastic/beats/pull/53521) [#53330](https://github.com/elastic/beats/issues/53330)
* Scope the aws-s3 lexicographical polling tail to the input&#39;s bucket and key prefix. [#53238](https://github.com/elastic/beats/pull/53238) [#53419](https://github.com/elastic/beats/pull/53419) [#53544](https://github.com/elastic/beats/pull/53544) [#53521](https://github.com/elastic/beats/pull/53521) [#53195](https://github.com/elastic/beats/issues/53195)

  All aws-s3 inputs of a process share one persistent state store. In
  lexicographical ordering mode the tail (the key passed as StartAfter to
  ListObjectsV2) was persisted under a single store key, so inputs polling
  different buckets or key prefixes read and overwrote each other&#39;s tail. An
  input could start listing after a key from another bucket and skip its own
  objects. The tail is now persisted under a key derived from the input&#39;s
  bucket and key prefix. The old unscoped key is removed on startup and the
  tail is reseeded from the input&#39;s own persisted states.
  

**Libbeat**

* Add_kubernetes_metadata: wait_for_metadata now prevents unenriched events at startup. [#53238](https://github.com/elastic/beats/pull/53238) [#53419](https://github.com/elastic/beats/pull/53419) [#53544](https://github.com/elastic/beats/pull/53544) [#53521](https://github.com/elastic/beats/pull/53521) [#53090](https://github.com/elastic/beats/issues/53090)
* Make an ETW reader session safe to reuse and stop, and fix a lost stop that could hang shutdown. [#53238](https://github.com/elastic/beats/pull/53238) [#53419](https://github.com/elastic/beats/pull/53419) [#53544](https://github.com/elastic/beats/pull/53544) [#53521](https://github.com/elastic/beats/pull/53521) [#49984](https://github.com/elastic/beats/issues/49984)

  The shared ETW reader, used by the filebeat etw input and the auditbeat
  file_integrity module, was changed so that a session can be used more than
  once and stopped from another goroutine safely:
  - `Session.Reset` clears the handles and rebuilds the session properties
    left behind by a previous run, so the same session can be created or
    attached to again without re-registering its event callback. This is what
    lets the etw input reconnect after its session is stopped and restarted.
  - `StopSession` no longer reports an error when the session it was asked to
    stop has already been stopped by another controller. It is a no-op when no
    session was created or attached to, so it can be used to clean up after a
    failed connection, and it only waits for flushed events to be processed
    when the flush succeeded and a consumer is running.
  - A stop issued before the trace was open is no longer lost: `StartConsumer`
    closes the trace itself instead of blocking in `ProcessTrace` on a session
    that nobody will end, which could hang shutdown. The trace handle is now
    guarded against concurrent access between the consumer and the stopper.
  The auditbeat file_integrity module inherits the last two fixes; its events
  and state are unchanged.
  
* Management: fix healthy input units stuck in CONFIGURING when a sibling unit fails config validation. [#53238](https://github.com/elastic/beats/pull/53238) [#53419](https://github.com/elastic/beats/pull/53419) [#53544](https://github.com/elastic/beats/pull/53544) [#53521](https://github.com/elastic/beats/pull/53521) [#53292](https://github.com/elastic/beats/issues/53292)

**Metricbeat**

* Stop losing a day of azure/billing data when Cost Management returns HTTP 429. [#53238](https://github.com/elastic/beats/pull/53238) [#53419](https://github.com/elastic/beats/pull/53419) [#53544](https://github.com/elastic/beats/pull/53544) [#53521](https://github.com/elastic/beats/pull/53521) [#53330](https://github.com/elastic/beats/issues/53330)

  The azure/billing metricset now honours the throttling hints Azure Cost Management and Consumption return on HTTP 429. Those hints arrive in &#34;x-ms-ratelimit-microsoft.*-retry-after&#34; headers, which the Azure SDK retry policy ignores, so a throttled fetch used to fail after a few seconds of backoff and, with no cursor and a 1440m period, the whole day of billing data was lost. The longest wait advertised by those hints or by the standard Retry-After headers is now honoured before retrying, the retry budget was raised so a per-minute quota window can clear (up to 5 retries, 10s base delay, 5m cap), and every request carries a ClientType header so the calls no longer share the default Cost Management quota pool with all unidentified callers in the tenant.
  

**Osquerybeat**

* Bound the osqueryd startup check and report startup failures through Elastic Agent. [#53238](https://github.com/elastic/beats/pull/53238) [#53419](https://github.com/elastic/beats/pull/53419) [#53544](https://github.com/elastic/beats/pull/53544) [#53521](https://github.com/elastic/beats/pull/53521) [#53330](https://github.com/elastic/beats/issues/53330)

  osquerybeat now bounds the osqueryd --version startup check (default 15s) so a hung validation cannot leave the unit stuck in STARTING. The deadline is configurable via `osquery.elastic_options.check_timeout` (Go duration, for example `30s`).
  
* Clamp native schedule metadata and history timestamps when endpoint clock skew precedes the configured start date. [#53238](https://github.com/elastic/beats/pull/53238) [#53419](https://github.com/elastic/beats/pull/53419) [#53544](https://github.com/elastic/beats/pull/53544) [#53521](https://github.com/elastic/beats/pull/53521) [#53330](https://github.com/elastic/beats/issues/53330)

**Packetbeat**

* Fix rendering of TLS certificate serial numbers so each byte is always represented as two hex digits, matching OpenSSL output. [#53238](https://github.com/elastic/beats/pull/53238) [#53419](https://github.com/elastic/beats/pull/53419) [#53544](https://github.com/elastic/beats/pull/53544) [#53521](https://github.com/elastic/beats/pull/53521) [#53330](https://github.com/elastic/beats/issues/53330)
* Fix panic in DHCPv4 parser when OptionTimeOffset value is shorter than 4 bytes. [#53238](https://github.com/elastic/beats/pull/53238) [#53419](https://github.com/elastic/beats/pull/53419) [#53544](https://github.com/elastic/beats/pull/53544) [#53521](https://github.com/elastic/beats/pull/53521) [#53330](https://github.com/elastic/beats/issues/53330)

**Winlogbeat**

* Fix binary event data fields being overread into subsequent buffer contents. [#53238](https://github.com/elastic/beats/pull/53238) [#53419](https://github.com/elastic/beats/pull/53419) [#53544](https://github.com/elastic/beats/pull/53544) [#53521](https://github.com/elastic/beats/pull/53521) [#53330](https://github.com/elastic/beats/issues/53330)

