## 9.2.5 [beats-release-notes-9.2.5]





### Fixes [beats-9.2.5-fixes]


**All**

* Fix windows install script to properly migrate legacy state data. [#48293](https://github.com/elastic/beats/pull/48293)
* Remove use of github.com/elastic/elastic-agent-client from OSS Beats. [#48353](https://github.com/elastic/beats/pull/48353) 

**Filebeat**

* Fix AD Entity Analytics failing to fetch users when Base DN contains a group CN. [#48395](https://github.com/elastic/beats/pull/48395) 
* Fix Filebeat goroutine leak when using harvester_limit. [#48445](https://github.com/elastic/beats/pull/48445)
* Update github.com/elastic/mito version to v1.24.1 to fix issue with rate limit calculation. [#48499](https://github.com/elastic/beats/pull/48499)

**Metricbeat**

* Enforce configurable size limits on incoming requests for remote_write metricset (max_compressed_body_bytes, max_decoded_body_bytes). [#48218](https://github.com/elastic/beats/pull/48218)
* Autoops agent to shutdown when it can&#39;t recover from the http errors. [#48292](https://github.com/elastic/beats/pull/48292)
* Add missing vector metrics in Autoops agent. [#48365](https://github.com/elastic/beats/pull/48365)
* Stack Monitoring now trims trailing slashes from host URLs for simplicity. [#48430](https://github.com/elastic/beats/pull/48430)
* Flatten AutoOps Cluster Settings to avoid unnecessary nesting and information. [#48454](https://github.com/elastic/beats/pull/48454)

**Packetbeat**

* Add check for incorrect length values in PostgreSQL datarow parser. [#47872](https://github.com/elastic/beats/pull/47872)
* Fix procfs network parsers. [#48428](https://github.com/elastic/beats/pull/48428)
* Fix Thrift struct parser oob bug. [#48498](https://github.com/elastic/beats/pull/48498)
* Clean up and add checks to SIP parser. [#48514](https://github.com/elastic/beats/pull/48514)
* Fix potential array access panics &amp; infinite loops in PostgreSQL parser. [#48528](https://github.com/elastic/beats/pull/48528)
