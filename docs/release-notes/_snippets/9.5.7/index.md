## 9.5.7 [beats-release-notes-9.5.7]



### Features and enhancements [beats-9.5.7-features-enhancements]


**All**

* Build the input status report of Beat receivers in time linear in the number of inputs. [#53723](https://github.com/elastic/beats/pull/53723)

**Metricbeat**

* Add the `tier_stats` metricset to the `autoops_es` module to collect the number and the duration of the requests of the search and indexing tiers of an Elasticsearch Serverless project. [#54117](https://github.com/elastic/beats/pull/54117)


### Fixes [beats-9.5.7-fixes]


**All**

* Update Brotli to 1.2.7 to fix a decoder panic on malformed input. [#54234](https://github.com/elastic/beats/pull/54234)
* Reduce retries and repeated error logs from Beat receivers while the output is unreachable. [#54271](https://github.com/elastic/beats/pull/54271)
* Fix "ApiVersionsRequest V3" errors on Kafka broker connections by updating the Sarama fork. [#54546](https://github.com/elastic/beats/pull/54546)

**Auditbeat**

* Fix the `auditd` module failing or dropping events when `immutable: true` is set. [#53707](https://github.com/elastic/beats/pull/53707)
* Stop Auditbeat `add_session_metadata` from stalling audit event reading after falling behind. [#53853](https://github.com/elastic/beats/pull/53853)
* Fix the `file_integrity` module's `kprobes` backend failing on kernels that list `.cold` symbols first. [#54106](https://github.com/elastic/beats/pull/54106)
* Fix Auditbeat `file_integrity` dropping all realtime events on macOS when `recursive` is `false`. [#54279](https://github.com/elastic/beats/pull/54279) [#54257](https://github.com/elastic/beats/issues/54257)

**Filebeat**

* Fix the `aws-s3` input rejecting `non_aws_bucket_name` configurations that set only `default_region`. [#54143](https://github.com/elastic/beats/pull/54143)
* Fix processors being applied twice to Elastic Agent filestream inputs configured without streams. [#54416](https://github.com/elastic/beats/pull/54416)
* Update mito to v1.27.1 to fix failures when following redirects with a response body, and to stop the CEL lazy stream decoders from losing read or decoding errors. [#54430](https://github.com/elastic/beats/pull/54430)
* Write Azure Event Hub v2 checkpoints once per message, after all of its records are processed. [#54503](https://github.com/elastic/beats/pull/54503)
* Fix the `azure-blob-storage` and `gcs` inputs re-reading files on every poll when file names have mixed case. [#54512](https://github.com/elastic/beats/pull/54512)
* Fix Filebeat receivers that share a registry failing to start together on Windows. [#54530](https://github.com/elastic/beats/pull/54530)
* Use the Elasticsearch state store for the `aws-cloudwatch` input in managed integration deployments. [#54573](https://github.com/elastic/beats/pull/54573)

**Heartbeat**

* Stop leaking a thread per run when a browser monitor's synthetics command fails to start. [#52405](https://github.com/elastic/beats/pull/52405)

**Libbeat**

* Encode the `data_stream` field set by the `add_data_stream` processor with lowercase keys in Beat receivers. [#54224](https://github.com/elastic/beats/pull/54224)

**Metricbeat**

* Fix process CPU ticks and percentages being under-reported about 40 times on Apple Silicon macOS. [#54502](https://github.com/elastic/beats/pull/54502)

**Osquerybeat**

* Stop scheduled query profiling from stalling Osquery scheduled results. [#53775](https://github.com/elastic/beats/pull/53775) [#53697](https://github.com/elastic/beats/issues/53697)
* Restart `osqueryd` with backoff when it exits unexpectedly after startup. [#53938](https://github.com/elastic/beats/pull/53938) [#53645](https://github.com/elastic/beats/issues/53645)
* Fix the Osquerybeat receiver hanging when it is shut down right after starting. [#53938](https://github.com/elastic/beats/pull/53938) [#53865](https://github.com/elastic/beats/issues/53865)
* Keep `osqueryd` running when the output is slow by queueing scheduled Osquery results. [#54251](https://github.com/elastic/beats/pull/54251) [#53645](https://github.com/elastic/beats/issues/53645)
* Reject Osquery policies that cannot be applied instead of stopping Osquerybeat. [#54249](https://github.com/elastic/beats/pull/54249)

**Winlogbeat**

* Fix PowerShell module pipeline failure on unexpected or localized ParameterBinding lines. [#54115](https://github.com/elastic/beats/pull/54115)
* Fix PowerShell module pipeline failure on event 4103 with multi-line ContextInfo values. [#54260](https://github.com/elastic/beats/pull/54260)
