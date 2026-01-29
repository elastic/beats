## 9.2.5 [beats-release-notes-9.2.5]





### Fixes [beats-9.2.5-fixes]


**All**

* Fix windows install script to properly migrate legacy state data. [#48498](https://github.com/elastic/beats/pull/48498) [#48444](https://github.com/elastic/beats/pull/48444) [#48600](https://github.com/elastic/beats/pull/48600) [#48601](https://github.com/elastic/beats/pull/48601) [#48602](https://github.com/elastic/beats/pull/48602) [#48563](https://github.com/elastic/beats/pull/48563) [#48543](https://github.com/elastic/beats/pull/48543) 
* Remove use of github.com/elastic/elastic-agent-client from OSS Beats. [#48353](https://github.com/elastic/beats/pull/48353) 

**Filebeat**

* Fix AD Entity Analytics failing to fetch users when Base DN contains a group CN. [#48395](https://github.com/elastic/beats/pull/48395) 
* Fix filebeat goroutine leak when using harvester_limit. [#48498](https://github.com/elastic/beats/pull/48498) [#48444](https://github.com/elastic/beats/pull/48444) [#48600](https://github.com/elastic/beats/pull/48600) [#48601](https://github.com/elastic/beats/pull/48601) [#48602](https://github.com/elastic/beats/pull/48602) [#48563](https://github.com/elastic/beats/pull/48563) [#48543](https://github.com/elastic/beats/pull/48543) 
* Update github.com/elastic/mito to v1.24.1 to fix issue with rate limit calculation. [#48498](https://github.com/elastic/beats/pull/48498) [#48444](https://github.com/elastic/beats/pull/48444) [#48600](https://github.com/elastic/beats/pull/48600) [#48601](https://github.com/elastic/beats/pull/48601) [#48602](https://github.com/elastic/beats/pull/48602) [#48563](https://github.com/elastic/beats/pull/48563) [#48543](https://github.com/elastic/beats/pull/48543) 

**Metricbeat**

* Enforces configurable size limits on incoming requests for remote_write metricset (max_compressed_body_bytes, max_decoded_body_bytes). [#48498](https://github.com/elastic/beats/pull/48498) [#48444](https://github.com/elastic/beats/pull/48444) [#48600](https://github.com/elastic/beats/pull/48600) [#48601](https://github.com/elastic/beats/pull/48601) [#48602](https://github.com/elastic/beats/pull/48602) [#48563](https://github.com/elastic/beats/pull/48563) [#48543](https://github.com/elastic/beats/pull/48543) 
* Autoops agent to shutdown when it can&#39;t recover from the http errors. [#48498](https://github.com/elastic/beats/pull/48498) [#48444](https://github.com/elastic/beats/pull/48444) [#48600](https://github.com/elastic/beats/pull/48600) [#48601](https://github.com/elastic/beats/pull/48601) [#48602](https://github.com/elastic/beats/pull/48602) [#48563](https://github.com/elastic/beats/pull/48563) [#48543](https://github.com/elastic/beats/pull/48543) 
* Added missing vector metrics in autoops agent. [#48498](https://github.com/elastic/beats/pull/48498) [#48444](https://github.com/elastic/beats/pull/48444) [#48600](https://github.com/elastic/beats/pull/48600) [#48601](https://github.com/elastic/beats/pull/48601) [#48602](https://github.com/elastic/beats/pull/48602) [#48563](https://github.com/elastic/beats/pull/48563) [#48543](https://github.com/elastic/beats/pull/48543) 
* Stack Monitoring now trims trailing slashes from host URLs for simplicity. [#48498](https://github.com/elastic/beats/pull/48498) [#48444](https://github.com/elastic/beats/pull/48444) [#48600](https://github.com/elastic/beats/pull/48600) [#48601](https://github.com/elastic/beats/pull/48601) [#48602](https://github.com/elastic/beats/pull/48602) [#48563](https://github.com/elastic/beats/pull/48563) [#48543](https://github.com/elastic/beats/pull/48543) [#48426](https://github.com/elastic/beats/issues/48426)
* Flatten AutoOps Cluster Settings to avoid unnecessary nesting and information. [#48498](https://github.com/elastic/beats/pull/48498) [#48444](https://github.com/elastic/beats/pull/48444) [#48600](https://github.com/elastic/beats/pull/48600) [#48601](https://github.com/elastic/beats/pull/48601) [#48602](https://github.com/elastic/beats/pull/48602) [#48563](https://github.com/elastic/beats/pull/48563) [#48543](https://github.com/elastic/beats/pull/48543) [#48453](https://github.com/elastic/beats/issues/48453)

**Packetbeat**

* Add check for incorrect length values in postgres datarow parser. [#48498](https://github.com/elastic/beats/pull/48498) [#48444](https://github.com/elastic/beats/pull/48444) [#48600](https://github.com/elastic/beats/pull/48600) [#48601](https://github.com/elastic/beats/pull/48601) [#48602](https://github.com/elastic/beats/pull/48602) [#48563](https://github.com/elastic/beats/pull/48563) [#48543](https://github.com/elastic/beats/pull/48543) 
* Fix procfs network parsers. [#48498](https://github.com/elastic/beats/pull/48498) [#48444](https://github.com/elastic/beats/pull/48444) [#48600](https://github.com/elastic/beats/pull/48600) [#48601](https://github.com/elastic/beats/pull/48601) [#48602](https://github.com/elastic/beats/pull/48602) [#48563](https://github.com/elastic/beats/pull/48563) [#48543](https://github.com/elastic/beats/pull/48543) 
* Fix Thrift struct parser oob bug. [#48498](https://github.com/elastic/beats/pull/48498) [#48444](https://github.com/elastic/beats/pull/48444) [#48600](https://github.com/elastic/beats/pull/48600) [#48601](https://github.com/elastic/beats/pull/48601) [#48602](https://github.com/elastic/beats/pull/48602) [#48563](https://github.com/elastic/beats/pull/48563) [#48543](https://github.com/elastic/beats/pull/48543) 
* Clean up and add checks to SIP parser. [#48498](https://github.com/elastic/beats/pull/48498) [#48444](https://github.com/elastic/beats/pull/48444) [#48600](https://github.com/elastic/beats/pull/48600) [#48601](https://github.com/elastic/beats/pull/48601) [#48602](https://github.com/elastic/beats/pull/48602) [#48563](https://github.com/elastic/beats/pull/48563) [#48543](https://github.com/elastic/beats/pull/48543) 
* Fix potential array access panics &amp; infinite loops in postgres parser. [#48498](https://github.com/elastic/beats/pull/48498) [#48444](https://github.com/elastic/beats/pull/48444) [#48600](https://github.com/elastic/beats/pull/48600) [#48601](https://github.com/elastic/beats/pull/48601) [#48602](https://github.com/elastic/beats/pull/48602) [#48563](https://github.com/elastic/beats/pull/48563) [#48543](https://github.com/elastic/beats/pull/48543) 

