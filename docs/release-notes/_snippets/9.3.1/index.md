## 9.3.1 [beats-release-notes-9.3.1]



### Features and enhancements [beats-9.3.1-features-enhancements]


**Filebeat**

* Add support for managed identity authentication to the `azure-eventhub` input. [#48655](https://github.com/elastic/beats/pull/48655) [#48680](https://github.com/elastic/beats/issues/48680)
* Improve log path sanitization for request trace logging. [#48719](https://github.com/elastic/beats/pull/48719) 
* Add descriptions and units to CEL input OpenTelemetry metrics. [#48684](https://github.com/elastic/beats/pull/48684) 
* Don't print warning about small files on each file system scan. [#48704](https://github.com/elastic/beats/pull/48704) [#45642](https://github.com/elastic/beats/issues/45642)
* Allow the configuration of OTLP histogram aggregation through the `OTEL_EXPORTER_OTLP_METRICS_DEFAULT_HISTOGRAM_AGGREGATION` environment variable in CEL input. [#48731](https://github.com/elastic/beats/pull/48731) [#48730](https://github.com/elastic/beats/issues/48730)
* Tighten request trace logging destination path checks in CEL, Entity Analytics, HTTP Endpoint and HTTP JSON inputs. [#48863](https://github.com/elastic/beats/pull/48863) 


### Fixes [beats-9.3.1-fixes]


**All**

* Updates the `translate_ldap_attribute` processor discovery to try both LDAP and LDAPS per host, starting with LDAPS. [#48818](https://github.com/elastic/beats/pull/48818) 

**Elastic Agent**

* Fix a bug which could report an invalid number of active `otelconsumer` events. [#48720](https://github.com/elastic/beats/pull/48720) [#12515](https://github.com/elastic/elastic-agent/issues/12515)

**Filebeat**

* Enforce region configuration when `non_aws_bucket_name` is defined for the `awss3` input. [#48534](https://github.com/elastic/beats/pull/48534) [#47847](https://github.com/elastic/beats/issues/47847)
* Fix Log to Filestream state migration removing states from non-harvested files. [#48570](https://github.com/elastic/beats/pull/48570) 
* Fix CEL input incorrectly counting degraded program runs as successful executions in OpenTelemetry metrics. [#48734](https://github.com/elastic/beats/pull/48734) [#48714](https://github.com/elastic/beats/issues/48714)
* Fix Active Directory Entity Analytics to resolve nested group membership and escape Base DN filter values. [#48815](https://github.com/elastic/beats/pull/48815) 
* Fix Entity Analytics Okta OAuth2 token requests ignoring custom TLS/SSL configuration. [#48866](https://github.com/elastic/beats/pull/48866) 
* Fix an issue where the `azure-blob-storage` input was failing with the Storage Blob Data Reader RBAC role. [#48886](https://github.com/elastic/beats/pull/48886) [#48890](https://github.com/elastic/beats/issues/48890)

**Filebeat, Metricbeat**

* Add 30s metric logging to Beat receivers. [#48541](https://github.com/elastic/beats/pull/48541) 

**Heartbeat**

* Add a missing dependency for Synthetics on Wolfi Docker image. [#48569](https://github.com/elastic/beats/pull/48569) 

**Libbeat**

* Add SSPI bind timeout and document Windows account requirements for the `translate_ldap_attribute` processor. [#48444](https://github.com/elastic/beats/pull/48444) 
* Fix `otelconsumer` logging hundreds of errors per second when queue is full. [#48807](https://github.com/elastic/beats/pull/48807) [#48803](https://github.com/elastic/beats/issues/48803)

**Osquerybeat**

* Fix differential results using wrong data source for removed events. [#48438](https://github.com/elastic/beats/pull/48438) [#48427](https://github.com/elastic/beats/issues/48427)

**Packetbeat**

* Refactor the DHCPv4 parsers and fix parsing issues. The DHCP `router` field is now a list, as is specified in RFC2132. [#48414](https://github.com/elastic/beats/pull/48414) 
* Fix procfs network parsers. [#48428](https://github.com/elastic/beats/pull/48428) 
* Fix a panic in the Thrift struct parser triggered by malformed packets. [#48498](https://github.com/elastic/beats/pull/48498) 
* Add array access checks to SIP parser. [#48514](https://github.com/elastic/beats/pull/48514) 
* Fix potential array access panics and infinite loops in PostgreSQL parser. [#48528](https://github.com/elastic/beats/pull/48528) 
* Clean up `int` overflows and array access issues in MySQL parsers. [#48543](https://github.com/elastic/beats/pull/48543) 
* Add `int` overflow checks to the `http` parser. [#48563](https://github.com/elastic/beats/pull/48563) 

