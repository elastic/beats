## 9.4.1 [beats-release-notes-9.4.1]



### Features and enhancements [beats-9.4.1-features-enhancements]


**Libbeat**

* Cache `add_locale` processor and refresh only when `zone` or `offset` changes. [#50343](https://github.com/elastic/beats/pull/50343) [#50322](https://github.com/elastic/beats/issues/50322)


### Fixes [beats-9.4.1-fixes]


**All**

* Update `go-ntlmssp` to v0.1.1. [#50497](https://github.com/elastic/beats/pull/50497) 
* Fix a deadlock between shutdown and metrics collection in the OpenTelemetry telemetry bridge. [#50528](https://github.com/elastic/beats/pull/50528)
* Fix OTel Beat processor to honor `when` conditions. [#50555](https://github.com/elastic/beats/pull/50555) [#50549](https://github.com/elastic/beats/issues/50549)

**Filebeat**

* Fix a race condition during multiline parser shutdown. [#49980](https://github.com/elastic/beats/pull/49980) 
* Fix Okta entity analytics OAuth2 config unpacking for `jwk_json` and `jwk_pem` fields. [#50406](https://github.com/elastic/beats/pull/50406) 
* Fix Active Directory entity analytics to emit device attributes under `activedirectory.device`. [#50472](https://github.com/elastic/beats/pull/50472) [#50471](https://github.com/elastic/beats/issues/50471)
* Fix handling of OAuth2.0 timeouts in CrowdStrike streaming input. [#50492](https://github.com/elastic/beats/pull/50492) [#49988](https://github.com/elastic/beats/issues/49988)

**Libbeat**

* Fix OTel map conversion for `[]time.Duration` fields to avoid dropping duration slices. [#50486](https://github.com/elastic/beats/pull/50486) [#50474](https://github.com/elastic/beats/issues/50474)

**Winlogbeat**

* Fix `Long.decode` failures in the Painless script for the Windows security ingest pipeline. [#49869](https://github.com/elastic/beats/pull/49869)
* Disable Winlogbeat record ID gap detection when using `xml_query` so filtered queries do not loop on non-contiguous record IDs. [#50443](https://github.com/elastic/beats/pull/50443)

