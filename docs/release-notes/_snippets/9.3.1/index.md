## 9.3.1 [beats-release-notes-9.3.1]



### Features and enhancements [beats-9.3.1-features-enhancements]


**Filebeat**

* Azure-eventhub-managed-identity. [#48655](https://github.com/elastic/beats/pull/48655) [#48680](https://github.com/elastic/beats/issues/48680)
* Improve log path sanitization for request trace logging. [#48719](https://github.com/elastic/beats/pull/48719) 
* Add descriptions and units to CEL input OpenTelemetry metrics. [#48684](https://github.com/elastic/beats/pull/48684) 
* Don&#39;t print warning about small files on each file system scan. [#48704](https://github.com/elastic/beats/pull/48704) [#45642](https://github.com/elastic/beats/issues/45642)
* Allow configuration of OTLP histogram aggregation via OTEL_EXPORTER_OTLP_METRICS_DEFAULT_HISTOGRAM_AGGREGATION environment variable in CEL input. [#48731](https://github.com/elastic/beats/pull/48731) [#48730](https://github.com/elastic/beats/issues/48730)
* Tighten request trace logging destination path checks in CEL, Entity Analytics, HTTP Endpoint and HTTP JSON inputs. [#48863](https://github.com/elastic/beats/pull/48863) 


### Fixes [beats-9.3.1-fixes]


**All**

* Translate_ldap_attribute discovery tries both LDAP and LDAPS per host, LDAPS first. [#48818](https://github.com/elastic/beats/pull/48818) 

  When the translate_ldap_attribute processor discovers LDAP servers (via DNS SRV
  or LOGONSERVER), it now adds the alternate scheme for each discovered address:
  if LDAP is found it also tries LDAPS for that host, and if LDAPS is found it
  also tries LDAP. For each host, LDAPS is tried before LDAP to prefer TLS.
  

**Elastic agent**

* Fix a bug that could report an invalid number of active &#34;otelconsumer&#34; events. [#48720](https://github.com/elastic/beats/pull/48720) [#12515](https://github.com/elastic/elastic-agent/issues/12515)

**Filebeat**

* Enforce region configuration when non_aws_bucket_name is defined for awss3 input. [#48534](https://github.com/elastic/beats/pull/48534) [#47847](https://github.com/elastic/beats/issues/47847)
* Fix Log to Filestream state migration removing states from non-harvested files. [#48570](https://github.com/elastic/beats/pull/48570) 
* Fix CEL input incorrectly counting degraded program runs as successful executions in OpenTelemetry metrics. [#48734](https://github.com/elastic/beats/pull/48734) [#48714](https://github.com/elastic/beats/issues/48714)
* Fix AD entity analytics to resolve nested group membership and escape DN filter values. [#48815](https://github.com/elastic/beats/pull/48815) 

  Use the LDAP_MATCHING_RULE_IN_CHAIN matching rule (OID 1.2.840.113556.1.4.1941)
  in Active Directory entity analytics memberOf filters to resolve nested group
  membership at query time. Also escape DN values in the changed-groups filter
  to prevent malformed queries when group names contain LDAP filter metacharacters.
  
* Fix Entity Analytics Okta OAuth2 token requests ignoring custom TLS/SSL configuration. [#48866](https://github.com/elastic/beats/pull/48866) 

  The OAuth2 authentication flow in the Okta entity analytics provider
  was ignoring the user-configured HTTP client. Instead, it was using
  Go&#39;s default HTTP client for all token-related requests
  (initial token exchange, token refresh, and API calls).
  
  This meant that any custom TLS/SSL or proxy settings configured by
  the user were silently discarded, causing connection failures in
  environments that rely on custom certificates or proxies.
  
  This fix ensures the configured HTTP client is propagated through
  all OAuth2 token operations, so that outgoing requests correctly
  use the user&#39;s transport configuration.
  
* Fix azure-blob-storage input failing with Storage Blob Data Reader RBAC role. [#48886](https://github.com/elastic/beats/pull/48886) [#48890](https://github.com/elastic/beats/issues/48890)

**Filebeat, metricbeat**

* Add 30s metric logging to beat receivers. [#48541](https://github.com/elastic/beats/pull/48541) 

**Heartbeat**

* Adds a missing dependency for Synthetics on wolfi docker image. [#48569](https://github.com/elastic/beats/pull/48569) 

**Libbeat**

* Add SSPI bind timeout and document Windows account requirements for translate_ldap_attribute processor. [#48444](https://github.com/elastic/beats/pull/48444) 

  The translate_ldap_attribute processor SSPI bind could hang indefinitely when
  running under a local user account (which cannot obtain Kerberos credentials).
  This fix adds a 10-second timeout to prevent the hang and updates documentation
  to clearly explain which Windows account types support SSPI authentication:
  Local System, Network Service, domain users, and gMSA accounts work; local
  user accounts do not.
  
* Fix otelconsumer logging hundreds of errors per second when queue is full. [#48807](https://github.com/elastic/beats/pull/48807) [#48803](https://github.com/elastic/beats/issues/48803)

**Osquerybeat**

* Fix differential results using wrong data source for removed events. [#48438](https://github.com/elastic/beats/pull/48438) [#48427](https://github.com/elastic/beats/issues/48427)

  Fixed two bugs in osquerybeat&#39;s differential results handling:
  1. &#34;removed&#34; events incorrectly read from DiffResults.Added instead of DiffResults.Removed
  2. Simplified code by removing unnecessary intermediate variable and publishing results directly
  

**Packetbeat**

* Refactor dhcpv4 parsers, fix numerous parsing bugs. The DHCP &#34;router&#34; field is now a list, as is specified in RFC2132. [#48414](https://github.com/elastic/beats/pull/48414) 
* Fix procfs network parsers. [#48428](https://github.com/elastic/beats/pull/48428) 
* Fix Thrift struct parser oob bug. [#48498](https://github.com/elastic/beats/pull/48498) 
* Clean up and add checks to SIP parser. [#48514](https://github.com/elastic/beats/pull/48514) 
* Fix potential array access panics &amp; infinite loops in postgres parser. [#48528](https://github.com/elastic/beats/pull/48528) 
* Clean int overflows and array access in mysql parsers. [#48543](https://github.com/elastic/beats/pull/48543) 
* Add int overflow checks to http parser. [#48563](https://github.com/elastic/beats/pull/48563) 

