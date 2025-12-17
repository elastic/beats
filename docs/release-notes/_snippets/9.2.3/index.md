## 9.2.3 [beats-release-notes-9.2.3]



### Features and enhancements [beats-9.2.3-features-enhancements]


**All**

* Make beats receivers emit status for their subcomponents. [#47256](https://github.com/elastic/beats/pull/47256) [#47956](https://github.com/elastic/beats/pull/47956) [#48115](https://github.com/elastic/beats/pull/48115) [#47823](https://github.com/elastic/beats/issues/47823)
* Add GUID translation, base DN inference, and SSPI authentication to LDAP processor. [#47827](https://github.com/elastic/beats/pull/47827) 

**Filebeat**

* Log unpublished event count and exit publish loop on input context cancellation. [#47256](https://github.com/elastic/beats/pull/47256) [#47956](https://github.com/elastic/beats/pull/47956) [#48115](https://github.com/elastic/beats/pull/48115) [#47823](https://github.com/elastic/beats/issues/47823)
* Improving input error reporting to Elastic Agent, specially when pipeline configurations are incorrect. [#47905](https://github.com/elastic/beats/pull/47905) [#45649](https://github.com/elastic/beats/issues/45649)

**Metricbeat**

* K8s_container_allocatable. [#47256](https://github.com/elastic/beats/pull/47256) [#47956](https://github.com/elastic/beats/pull/47956) [#48115](https://github.com/elastic/beats/pull/48115) [#47823](https://github.com/elastic/beats/issues/47823)

  Updates kubernetes cpu and memory metrics to use allocatable values instead of capacity values.
* Add resource pool id to vsphere cluster metricset. [#47256](https://github.com/elastic/beats/pull/47256) [#47956](https://github.com/elastic/beats/pull/47956) [#48115](https://github.com/elastic/beats/pull/48115) [#47823](https://github.com/elastic/beats/issues/47823)

**Packetbeat**

* Ipfrag2. [#47256](https://github.com/elastic/beats/pull/47256) [#47956](https://github.com/elastic/beats/pull/47956) [#48115](https://github.com/elastic/beats/pull/48115) [#47823](https://github.com/elastic/beats/issues/47823)


### Fixes [beats-9.2.3-fixes]


**Filebeat**

* Prevent panic during startup if dissect processor has invalid field name in tokenizer. [#47256](https://github.com/elastic/beats/pull/47256) [#47956](https://github.com/elastic/beats/pull/47956) [#48115](https://github.com/elastic/beats/pull/48115) [#47823](https://github.com/elastic/beats/issues/47823)

**Metricbeat**

* Improve defensive checks to prevent panics in meraki module. [#47256](https://github.com/elastic/beats/pull/47256) [#47956](https://github.com/elastic/beats/pull/47956) [#48115](https://github.com/elastic/beats/pull/48115) [#47823](https://github.com/elastic/beats/issues/47823)
* Remove GCP Billing timestamp functions. [#47256](https://github.com/elastic/beats/pull/47256) [#47956](https://github.com/elastic/beats/pull/47956) [#48115](https://github.com/elastic/beats/pull/48115) [#47823](https://github.com/elastic/beats/issues/47823)

**Packetbeat**

* Rpc_fragment_sanitization. [#47803](https://github.com/elastic/beats/pull/47803) 
* Verify and cap memcache udp fragment counts. [#47256](https://github.com/elastic/beats/pull/47256) [#47956](https://github.com/elastic/beats/pull/47956) [#48115](https://github.com/elastic/beats/pull/48115) [#47823](https://github.com/elastic/beats/issues/47823)

