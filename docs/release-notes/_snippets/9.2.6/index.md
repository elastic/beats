## 9.2.6 [beats-release-notes-9.2.6]



### Features and enhancements [beats-9.2.6-features-enhancements]


**Filebeat**

* Azure-eventhub-managed-identity. [#48655](https://github.com/elastic/beats/pull/48655) [#48680](https://github.com/elastic/beats/issues/48680)
* Improve log path sanitization for request trace logging. [#48444](https://github.com/elastic/beats/pull/48444) [#48414](https://github.com/elastic/beats/pull/48414) [#48804](https://github.com/elastic/beats/pull/48804) [#48784](https://github.com/elastic/beats/pull/48784) [#48818](https://github.com/elastic/beats/pull/48818) [#48816](https://github.com/elastic/beats/pull/48816) [#48815](https://github.com/elastic/beats/pull/48815) 
* Don&#39;t print warning about small files on each file system scan. [#48444](https://github.com/elastic/beats/pull/48444) [#48414](https://github.com/elastic/beats/pull/48414) [#48804](https://github.com/elastic/beats/pull/48804) [#48784](https://github.com/elastic/beats/pull/48784) [#48818](https://github.com/elastic/beats/pull/48818) [#48816](https://github.com/elastic/beats/pull/48816) [#48815](https://github.com/elastic/beats/pull/48815) 
* Tighten request trace logging destination path checks in CEL, Entity Analytics, HTTP Endpoint and HTTP JSON inputs. [#48444](https://github.com/elastic/beats/pull/48444) [#48414](https://github.com/elastic/beats/pull/48414) [#48804](https://github.com/elastic/beats/pull/48804) [#48784](https://github.com/elastic/beats/pull/48784) [#48818](https://github.com/elastic/beats/pull/48818) [#48816](https://github.com/elastic/beats/pull/48816) [#48815](https://github.com/elastic/beats/pull/48815) 


### Fixes [beats-9.2.6-fixes]


**Filebeat**

* Enforce region configuration when non_aws_bucket_name is defined for awss3 input. [#48444](https://github.com/elastic/beats/pull/48444) [#48414](https://github.com/elastic/beats/pull/48414) [#48804](https://github.com/elastic/beats/pull/48804) [#48784](https://github.com/elastic/beats/pull/48784) [#48818](https://github.com/elastic/beats/pull/48818) [#48816](https://github.com/elastic/beats/pull/48816) [#48815](https://github.com/elastic/beats/pull/48815) 
* Fix Log to Filestream state migration removing states from non-harvested files. [#48570](https://github.com/elastic/beats/pull/48570) 
* Fix AD entity analytics to resolve nested group membership and escape DN filter values. [#48444](https://github.com/elastic/beats/pull/48444) [#48414](https://github.com/elastic/beats/pull/48414) [#48804](https://github.com/elastic/beats/pull/48804) [#48784](https://github.com/elastic/beats/pull/48784) [#48818](https://github.com/elastic/beats/pull/48818) [#48816](https://github.com/elastic/beats/pull/48816) [#48815](https://github.com/elastic/beats/pull/48815) 

  Use the LDAP_MATCHING_RULE_IN_CHAIN matching rule (OID 1.2.840.113556.1.4.1941)
  in Active Directory entity analytics memberOf filters to resolve nested group
  membership at query time. Also escape DN values in the changed-groups filter
  to prevent malformed queries when group names contain LDAP filter metacharacters.
  
* Fix Entity Analytics Okta OAuth2 token requests ignoring custom TLS/SSL configuration. [#48444](https://github.com/elastic/beats/pull/48444) [#48414](https://github.com/elastic/beats/pull/48414) [#48804](https://github.com/elastic/beats/pull/48804) [#48784](https://github.com/elastic/beats/pull/48784) [#48818](https://github.com/elastic/beats/pull/48818) [#48816](https://github.com/elastic/beats/pull/48816) [#48815](https://github.com/elastic/beats/pull/48815) 

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

**Heartbeat**

* Adds a missing dependency for Synthetics on wolfi docker image. [#48444](https://github.com/elastic/beats/pull/48444) [#48414](https://github.com/elastic/beats/pull/48414) [#48804](https://github.com/elastic/beats/pull/48804) [#48784](https://github.com/elastic/beats/pull/48784) [#48818](https://github.com/elastic/beats/pull/48818) [#48816](https://github.com/elastic/beats/pull/48816) [#48815](https://github.com/elastic/beats/pull/48815) 

**Osquerybeat**

* Fix differential results using wrong data source for removed events. [#48444](https://github.com/elastic/beats/pull/48444) [#48414](https://github.com/elastic/beats/pull/48414) [#48804](https://github.com/elastic/beats/pull/48804) [#48784](https://github.com/elastic/beats/pull/48784) [#48818](https://github.com/elastic/beats/pull/48818) [#48816](https://github.com/elastic/beats/pull/48816) [#48815](https://github.com/elastic/beats/pull/48815) [#48427](https://github.com/elastic/beats/issues/48427)

  Fixed two bugs in osquerybeat&#39;s differential results handling:
  1. &#34;removed&#34; events incorrectly read from DiffResults.Added instead of DiffResults.Removed
  2. Simplified code by removing unnecessary intermediate variable and publishing results directly
  

**Packetbeat**

* Clean int overflows and array access in mysql parsers. [#48444](https://github.com/elastic/beats/pull/48444) [#48414](https://github.com/elastic/beats/pull/48414) [#48804](https://github.com/elastic/beats/pull/48804) [#48784](https://github.com/elastic/beats/pull/48784) [#48818](https://github.com/elastic/beats/pull/48818) [#48816](https://github.com/elastic/beats/pull/48816) [#48815](https://github.com/elastic/beats/pull/48815) 
* Add int overflow checks to http parser. [#48444](https://github.com/elastic/beats/pull/48444) [#48414](https://github.com/elastic/beats/pull/48414) [#48804](https://github.com/elastic/beats/pull/48804) [#48784](https://github.com/elastic/beats/pull/48784) [#48818](https://github.com/elastic/beats/pull/48818) [#48816](https://github.com/elastic/beats/pull/48816) [#48815](https://github.com/elastic/beats/pull/48815) 

