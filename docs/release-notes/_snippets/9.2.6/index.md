## 9.2.6 [beats-release-notes-9.2.6]



### Features and enhancements [beats-9.2.6-features-enhancements]


**Filebeat**

* Add support for managed identity authentication to the `azure-eventhub` input. [#48655](https://github.com/elastic/beats/pull/48655) [#48680](https://github.com/elastic/beats/issues/48680)
* Improve log path sanitization for request trace logging. [#48719](https://github.com/elastic/beats/pull/48719) 
* Don't print warning about small files on each file system scan. [#48704](https://github.com/elastic/beats/pull/48704) [#45642](https://github.com/elastic/beats/issues/45642)
* Tighten request trace logging destination path checks in CEL, Entity Analytics, HTTP Endpoint and HTTP JSON inputs. [#48863](https://github.com/elastic/beats/pull/48863) 


### Fixes [beats-9.2.6-fixes]


**Filebeat**

* Enforce region configuration when `non_aws_bucket_name` is defined for the `awss3` input. [#48534](https://github.com/elastic/beats/pull/48534) [#47847](https://github.com/elastic/beats/issues/47847)
* Fix Log to Filestream state migration removing states from non-harvested files. [#48570](https://github.com/elastic/beats/pull/48570) 
* Fix Active Directory Entity Analytics to resolve nested group membership and escape Base DN filter values. [#48395](https://github.com/elastic/beats/pull/48395) 
* Fix Entity Analytics Okta OAuth2 token requests ignoring custom TLS/SSL configuration. [#48866](https://github.com/elastic/beats/pull/48866) 
* Fix an issue where the `azure-blob-storage` input was failing with the Storage Blob Data Reader RBAC role. [#48886](https://github.com/elastic/beats/pull/48886) [#48890](https://github.com/elastic/beats/issues/48890)

**Heartbeat**

* Add a missing dependency for Synthetics on Wolfi Docker image. [#48569](https://github.com/elastic/beats/pull/48569) 

**Osquerybeat**

* Fix differential results using wrong data source for removed events. [#48438](https://github.com/elastic/beats/pull/48438) [#48427](https://github.com/elastic/beats/issues/48427)

**Packetbeat**

* Clean up `int` overflows and array access issues in MySQL parsers. [#48543](https://github.com/elastic/beats/pull/48543) 
* Add `int` overflow checks to the `http` parser. [#48563](https://github.com/elastic/beats/pull/48563) 

