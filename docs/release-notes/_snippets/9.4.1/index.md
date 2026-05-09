## 9.4.1 [beats-release-notes-9.4.1]



### Features and enhancements [beats-9.4.1-features-enhancements]


**Libbeat**

* Cache add_locale processor and refresh only when zone or offset changes. [#50555](https://github.com/elastic/beats/pull/50555) [#50567](https://github.com/elastic/beats/pull/50567) [#50568](https://github.com/elastic/beats/pull/50568) [#50569](https://github.com/elastic/beats/pull/50569) [#50563](https://github.com/elastic/beats/pull/50563) [#50549](https://github.com/elastic/beats/issues/50549)


### Fixes [beats-9.4.1-fixes]


**All**

* Update github.com/Azure/go-ntlmssp to v0.1.1. [#50555](https://github.com/elastic/beats/pull/50555) [#50567](https://github.com/elastic/beats/pull/50567) [#50568](https://github.com/elastic/beats/pull/50568) [#50569](https://github.com/elastic/beats/pull/50569) [#50563](https://github.com/elastic/beats/pull/50563) [#50549](https://github.com/elastic/beats/issues/50549)
* Fix a deadlock in beat otel receiver shutdown. [#50555](https://github.com/elastic/beats/pull/50555) [#50567](https://github.com/elastic/beats/pull/50567) [#50568](https://github.com/elastic/beats/pull/50568) [#50569](https://github.com/elastic/beats/pull/50569) [#50563](https://github.com/elastic/beats/pull/50563) [#50549](https://github.com/elastic/beats/issues/50549)
* Fix OTel Beat processor to honor `when` conditions. [#50555](https://github.com/elastic/beats/pull/50555) [#50567](https://github.com/elastic/beats/pull/50567) [#50568](https://github.com/elastic/beats/pull/50568) [#50569](https://github.com/elastic/beats/pull/50569) [#50563](https://github.com/elastic/beats/pull/50563) [#50549](https://github.com/elastic/beats/issues/50549)

**Filebeat**

* Fix race condition during multiline parser shutdown. [#50555](https://github.com/elastic/beats/pull/50555) [#50567](https://github.com/elastic/beats/pull/50567) [#50568](https://github.com/elastic/beats/pull/50568) [#50569](https://github.com/elastic/beats/pull/50569) [#50563](https://github.com/elastic/beats/pull/50563) [#50549](https://github.com/elastic/beats/issues/50549)
* Fix Okta entity analytics OAuth2 config unpacking for jwk_json and jwk_pem fields. [#50555](https://github.com/elastic/beats/pull/50555) [#50567](https://github.com/elastic/beats/pull/50567) [#50568](https://github.com/elastic/beats/pull/50568) [#50569](https://github.com/elastic/beats/pull/50569) [#50563](https://github.com/elastic/beats/pull/50563) [#50549](https://github.com/elastic/beats/issues/50549)
* Fix Active Directory entity analytics to emit device attributes under activedirectory.device. [#50555](https://github.com/elastic/beats/pull/50555) [#50567](https://github.com/elastic/beats/pull/50567) [#50568](https://github.com/elastic/beats/pull/50568) [#50569](https://github.com/elastic/beats/pull/50569) [#50563](https://github.com/elastic/beats/pull/50563) [#50471](https://github.com/elastic/beats/issues/50471)
* Fix handling of OAuth2.0 timeouts in CrowdStrike streaming input. [#50555](https://github.com/elastic/beats/pull/50555) [#50567](https://github.com/elastic/beats/pull/50567) [#50568](https://github.com/elastic/beats/pull/50568) [#50569](https://github.com/elastic/beats/pull/50569) [#50563](https://github.com/elastic/beats/pull/50563) [#50549](https://github.com/elastic/beats/issues/50549)

**Libbeat**

* Fix OTel map conversion for []time.Duration fields to avoid dropping duration slices. [#50555](https://github.com/elastic/beats/pull/50555) [#50567](https://github.com/elastic/beats/pull/50567) [#50568](https://github.com/elastic/beats/pull/50568) [#50569](https://github.com/elastic/beats/pull/50569) [#50563](https://github.com/elastic/beats/pull/50563) [#50549](https://github.com/elastic/beats/issues/50549)

**Winlogbeat**

* Fix Long decoding error in painless script for windows ingest pipeline. [#50555](https://github.com/elastic/beats/pull/50555) [#50567](https://github.com/elastic/beats/pull/50567) [#50568](https://github.com/elastic/beats/pull/50568) [#50569](https://github.com/elastic/beats/pull/50569) [#50563](https://github.com/elastic/beats/pull/50563) [#50549](https://github.com/elastic/beats/issues/50549)
* Disable Winlogbeat record ID gap detection when using xml_query so filtered queries do not loop on non-contiguous record IDs. [#50555](https://github.com/elastic/beats/pull/50555) [#50567](https://github.com/elastic/beats/pull/50567) [#50568](https://github.com/elastic/beats/pull/50568) [#50569](https://github.com/elastic/beats/pull/50569) [#50563](https://github.com/elastic/beats/pull/50563) [#50549](https://github.com/elastic/beats/issues/50549)

