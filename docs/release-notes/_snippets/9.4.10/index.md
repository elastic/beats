## 9.4.10 [beats-release-notes-9.4.10]



### Features and enhancements [beats-9.4.10-features-enhancements]


**All**

* Build the input status report of Beat receivers in time linear in the number of inputs. [#53723](https://github.com/elastic/beats/pull/53723) [#53730](https://github.com/elastic/beats/issues/53730)

**Metricbeat**

* Add the `tier_stats` metricset to the `autoops_es` module to collect the number and the duration of the requests of the search and indexing tiers of an Elasticsearch Serverless project. [#54117](https://github.com/elastic/beats/pull/54117)


### Fixes [beats-9.4.10-fixes]


**All**

* Fix quadratic cost of building the receiver config for units with many streams. [#53726](https://github.com/elastic/beats/pull/53726) [#53731](https://github.com/elastic/beats/issues/53731)
* Update Brotli to 1.2.7 to fix a decoder panic on malformed input. [#54234](https://github.com/elastic/beats/pull/54234)
* Fix "ApiVersionsRequest V3" errors on Kafka broker connections by updating the Sarama fork. [#54546](https://github.com/elastic/beats/pull/54546)

**Auditbeat**

* Fix the `auditd` module failing or dropping events when `immutable: true` is set. [#53707](https://github.com/elastic/beats/pull/53707) [#50189](https://github.com/elastic/beats/issues/50189)
* Stop Auditbeat `add_session_metadata` from stalling audit event reading after falling behind. [#53853](https://github.com/elastic/beats/pull/53853)
* Fix the `file_integrity` module's `kprobes` backend failing on kernels that list `.cold` symbols first. [#54106](https://github.com/elastic/beats/pull/54106)
* Fix Auditbeat `file_integrity` dropping all realtime events on macOS when `recursive` is `false`. [#54279](https://github.com/elastic/beats/pull/54279) [#54257](https://github.com/elastic/beats/issues/54257)

**Filebeat**

* Fix processors being applied twice to Elastic Agent filestream inputs configured without streams. [#54416](https://github.com/elastic/beats/pull/54416)
* Fix the `azure-blob-storage` and `gcs` inputs re-reading files on every poll when file names have mixed case. [#54512](https://github.com/elastic/beats/pull/54512)
* Fix Filebeat receivers that share a registry failing to start together on Windows. [#54530](https://github.com/elastic/beats/pull/54530)

**Heartbeat**

* Stop leaking a thread per run when a browser monitor's synthetics command fails to start. [#52405](https://github.com/elastic/beats/pull/52405)

**Libbeat**

* Encode the `data_stream` field set by the `add_data_stream` processor with lowercase keys in Beat receivers. [#54224](https://github.com/elastic/beats/pull/54224)
* Accept client certificates signed by CAs added to `certificate_authorities` while a TLS server is running. [#54749](https://github.com/elastic/beats/pull/54749) [#54742](https://github.com/elastic/beats/issues/54742)

**Metricbeat**

* Fix process CPU ticks and percentages being under-reported about 40 times on Apple Silicon macOS. [#54502](https://github.com/elastic/beats/pull/54502)

**Winlogbeat**

* Fix PowerShell module pipeline failure on unexpected or localized ParameterBinding lines. [#54115](https://github.com/elastic/beats/pull/54115)
* Fix PowerShell module pipeline failure on event 4103 with multi-line ContextInfo values. [#54260](https://github.com/elastic/beats/pull/54260)

