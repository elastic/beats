## 9.1.9 [beats-release-notes-9.1.9]

_This release also includes: [Breaking changes](/release-notes/breaking-changes.md#beats-9.1.9-breaking-changes)._


### Features and enhancements [beats-9.1.9-features-enhancements]


**Filebeat**

* Log unpublished event count and exit publish loop on input context cancellation. [#47256](https://github.com/elastic/beats/pull/47256) [#47956](https://github.com/elastic/beats/pull/47956) [#48141](https://github.com/elastic/beats/pull/48141) [#48115](https://github.com/elastic/beats/pull/48115) [#47823](https://github.com/elastic/beats/issues/47823)

**Metricbeat**

* Add resource pool id to vsphere cluster metricset. [#47256](https://github.com/elastic/beats/pull/47256) [#47956](https://github.com/elastic/beats/pull/47956) [#48141](https://github.com/elastic/beats/pull/48141) [#48115](https://github.com/elastic/beats/pull/48115) [#47823](https://github.com/elastic/beats/issues/47823)

**Packetbeat**

* Ipfrag2. [#47256](https://github.com/elastic/beats/pull/47256) [#47956](https://github.com/elastic/beats/pull/47956) [#48141](https://github.com/elastic/beats/pull/48141) [#48115](https://github.com/elastic/beats/pull/48115) [#47823](https://github.com/elastic/beats/issues/47823)


### Fixes [beats-9.1.9-fixes]


**Filebeat**

* [Filestream] ensure harvester always restarts if the file has not been fully ingested. [#47107](https://github.com/elastic/beats/pull/47107) [#46923](https://github.com/elastic/beats/issues/46923)
* Prevent panic during startup if dissect processor has invalid field name in tokenizer. [#47256](https://github.com/elastic/beats/pull/47256) [#47956](https://github.com/elastic/beats/pull/47956) [#48141](https://github.com/elastic/beats/pull/48141) [#48115](https://github.com/elastic/beats/pull/48115) [#47823](https://github.com/elastic/beats/issues/47823)

**Metricbeat**

* Improve defensive checks to prevent panics in meraki module. [#47256](https://github.com/elastic/beats/pull/47256) [#47956](https://github.com/elastic/beats/pull/47956) [#48141](https://github.com/elastic/beats/pull/48141) [#48115](https://github.com/elastic/beats/pull/48115) [#47823](https://github.com/elastic/beats/issues/47823)
* Remove GCP Billing timestamp functions. [#47256](https://github.com/elastic/beats/pull/47256) [#47956](https://github.com/elastic/beats/pull/47956) [#48141](https://github.com/elastic/beats/pull/48141) [#48115](https://github.com/elastic/beats/pull/48115) [#47823](https://github.com/elastic/beats/issues/47823)

**Packetbeat**

* Rpc_fragment_sanitization. [#47803](https://github.com/elastic/beats/pull/47803) 
* Verify and cap memcache udp fragment counts. [#47256](https://github.com/elastic/beats/pull/47256) [#47956](https://github.com/elastic/beats/pull/47956) [#48141](https://github.com/elastic/beats/pull/48141) [#48115](https://github.com/elastic/beats/pull/48115) [#47823](https://github.com/elastic/beats/issues/47823)

