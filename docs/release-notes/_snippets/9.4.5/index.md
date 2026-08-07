## 9.4.5 [beats-release-notes-9.4.5]



### Features and enhancements [beats-9.4.5-features-enhancements]


**Auditbeat**

* Update quark to 0.6.0. [#51767](https://github.com/elastic/beats/pull/51767) 

**Filebeat**

* Aggregate SQS mode health status to reduce false Degraded/Healthy transitions. [#51819](https://github.com/elastic/beats/pull/51819) [#51692](https://github.com/elastic/beats/issues/51692)

  Replace scattered per-event status reporting in the aws-s3 SQS input with a
  centralized health aggregator. The input now stays Running while making forward
  progress despite bounded retryable failures, reports Degraded only for sustained
  conditions needing operator action (persistent receive failures, delete/finalize
  errors, poison-pill messages), filters out context.Canceled from shutdown/reload,
  and clears Degraded only when the specific causing condition resolves.
  
* Add configurable User-Agent header to the CrowdStrike streaming input. [#51822](https://github.com/elastic/beats/pull/51822) 
* Remove Authorization headers from CEL, Entity Analytics, HTTP Endpoint, and HTTP JSON input request trace logs.  


### Fixes [beats-9.4.5-fixes]


**All**

* Upgrade to go 1.26.5.  

**Elastic agent**

* Fix beat receiver Shutdown hanging when Start fails before the beater is launched. [#52106](https://github.com/elastic/beats/pull/52106) [#52009](https://github.com/elastic/beats/issues/52009)

**Filebeat**

* Fix offset commits when using multiline in Journald and Kafka inputs. [#52286](https://github.com/elastic/beats/pull/52286) [#51981](https://github.com/elastic/beats/issues/51981)
* Fix S3 polling input to exclude same-bucket backup objects from listing. [#52170](https://github.com/elastic/beats/pull/52170) 

  When same-bucket backup is configured with the default empty
  bucket_list_prefix, backup objects are listed alongside source objects
  and reprocessed indefinitely. Exclude keys matching the backup prefix
  from listing results when the backup destination is the same bucket.
  
* Honor path_style for non-AWS S3 buckets in the aws-s3 input. [#52003](https://github.com/elastic/beats/pull/52003) 

  When using non_aws_bucket_name with a custom endpoint, the aws-s3 input always marked the endpoint hostname immutable, forcing path-style requests and ignoring path_style. This broke S3-compatible providers that require virtual-hosted addressing, which returned VirtualHostDomainRequired.  The hostname is now kept mutable for non-AWS buckets so path_style is honored.
  
* Use a stable status message for SQS receive errors to prevent update churn during sustained outages.  
* Do not mark the CEL input&#39;s health as degraded when the maximum executions is exceeded.  
* Fix aws-s3 input silently dropping events when a parser clears the message content. [#52077](https://github.com/elastic/beats/pull/52077) 

  A parser can move the decoded data into the event fields and clear the content, for example an ndjson parser configured without a message_key. When such an object was read through the readFile path (any content-type other than application/json or application/x-ndjson), the input dropped every event with no error logged, because it only published when the message content was non-empty. Events are now published when either the content or the fields are populated.
  
* Honor the configured language when rendering Windows events.  [#7332](https://github.com/elastic/sdh-beats/issues/7332)
* Fix journald input facility filters on journalctl older than 245.  

  The journald input passed `--facility` to journalctl unconditionally, but the flag only exists on journalctl &gt;= 245, so on older systems (e.g. RHEL 8) configurations with `facilities` set collected no events. Facilities are now passed as SYSLOG_FACILITY matches, which are supported on every version and are what `--facility` translates to internally.
* The journald input now reports a Degraded status when journalctl repeatedly exits without delivering any data, instead of staying Healthy while collecting no events.  
* Fix TCP and UDP inputs hanging during shutdown.  

**Heartbeat**

* Fix panic with multiple heartbeat receivers.  
* Bake Synthetics browser binaries into the heartbeat Docker image.  [#52439](https://github.com/elastic/beats/issues/52439)

  npm 12 no longer runs a dependency&#39;s `install` lifecycle script during `npm i` by
  default, so the transitive playwright-chromium `install` hook that used to download
  the Playwright browsers into the heartbeat image during build stopped running. As a
  result the image shipped with no browsers and all Synthetics browser monitors failed
  with &#34;browserType.launch: Executable doesn&#39;t exist at
  .../chromium_headless_shell-&lt;rev&gt;/...&#34;. The image build now installs the browsers
  explicitly with the bundled Playwright CLI after `npm i`, independent of npm&#39;s
  install-script policy.
  

**Libbeat**

* Prevent stale autodiscover resources after Kubernetes leadership changes.  

**Metricbeat**

* Fix Azure module authentication on sovereign clouds (Government, China).  

  The Azure module now derives the full cloud configuration (ARM token audience, metrics batch API endpoint and audience) from resource_manager_endpoint. Previously the monitor metricset silently ignored resource_manager_audience, always requesting public-cloud tokens, the billing metricset mutated the SDK&#39;s global cloud configuration, and the metrics batch API endpoint was hardcoded to the public cloud.

**Osquerybeat**

* Reject CPIO entries in macOS .pkg payloads whose paths escape the destination directory, preventing path traversal during extraction (CWE-22). [#51446](https://github.com/elastic/beats/pull/51446) 

