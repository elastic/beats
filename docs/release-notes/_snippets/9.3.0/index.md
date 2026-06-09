## 9.3.0 [beats-release-notes-9.3.0]

_This release also includes: [Deprecations](/release-notes/deprecations.md#beats-9.3.0-deprecations)._


### Features and enhancements [beats-9.3.0-features-enhancements]


**All**

* Introduce cloud connectors flow. [#47587](https://github.com/elastic/beats/pull/47587)
* Make beats receivers emit status for their subcomponents. [#48015](https://github.com/elastic/beats/pull/48015)
* Add GUID translation, base DN inference, and SSPI authentication to LDAP processor. [#47827](https://github.com/elastic/beats/pull/47827) 
* Add `events.failure_store` metric to track events sent to Elasticsearch failure store. [#48068](https://github.com/elastic/beats/pull/48068) [#47164](https://github.com/elastic/beats/issues/47164)

**Filebeat**

* Add support for direct HTTP request rate limit setting in CEL input. [#46953](https://github.com/elastic/beats/pull/46953)
* Allow adding file owner and group to the events&#39; meta fields on Unix systems. [#47331](https://github.com/elastic/beats/pull/47331) [#43226](https://github.com/elastic/beats/issues/43226)
* Add AWS auth method for CEL and HTTP JSON inputs. [#47260](https://github.com/elastic/beats/pull/47260)
* Add client secret authentication method for Azure Event Hub and storage in Filebeat. [#47256](https://github.com/elastic/beats/pull/47256)
* Add client address and name to submitted Redis slowlogs. [#41507](https://github.com/elastic/beats/pull/41507)
* Log unpublished event count and exit publish loop on input context cancellation. [#47730](https://github.com/elastic/beats/pull/47730)
* Upgrade CEL mito library version to v1.24.0. [#47762](https://github.com/elastic/beats/pull/47762)
* Add OTEL metrics to CEL inputs. [#47014](https://github.com/elastic/beats/pull/47014)
* Improve input error reporting to Elastic Agent, specially when pipeline configurations are incorrect. [#47905](https://github.com/elastic/beats/pull/47905) [#45649](https://github.com/elastic/beats/issues/45649)
* Azure EventHub input v2 - Add support for AMQP over WebSocket and HTTPS proxy. [#47956](https://github.com/elastic/beats/pull/47956) [#47823](https://github.com/elastic/beats/issues/47823)
* The Journald input now supports setting a chroot to use when calling
the journalctl binary, thus allowing the journald input to be used
with the wolfi container variant and in environments where the
host&#39;s Journald is not compatible with the `journalctl` version
shipped with the container.
. [#48008](https://github.com/elastic/beats/pull/48008) [#47164](https://github.com/elastic/beats/issues/47164)

  Add support in the journald inpur for using chroot when calling
  `journalctl`. In a container environment this allows to mount the host
  file system into the container and use its `journalctl`, which
  prevents any sort of incompatibility between the `journalctl` in the
  container image and the host Journald. Allows using the journald input with Wolfi based Docker containers.
  
* GZIP support is GA and always enabled on filestream. [#47893](https://github.com/elastic/beats/pull/47893) [#47880](https://github.com/elastic/beats/issues/47880)

  Ingesting GZIP-compressed files is now GA. The `gzip_experimental` configuration option has been deprecated. Users should use `compression` instead. Refer to the [documentation](https://www.elastic.co/docs/reference/beats/filebeat/filebeat-input-filestream#reading-gzip-files) for more details.
  
* Filebeat deploy on kubernetes examples use `compression` instead of `gzip_experimental`. [#48079](https://github.com/elastic/beats/pull/48079) [#47882](https://github.com/elastic/beats/issues/47882)
* Add file-based auth provider for CEL and HTTP JSON inputs. [#47507](https://github.com/elastic/beats/pull/47507) [#47506](https://github.com/elastic/beats/issues/47506)

  The CEL and HTTP JSON inputs now support reading authentication tokens from
  files, enabling integration with various secret providers like Vault,
  Kubernetes secret projections, etc. Tokens are automatically refreshed based on
  a configurable interval without requiring restarts.
  

**Metricbeat**

* Change calculation of CPU/Memory for Kubernetes to allocatable values. [#47815](https://github.com/elastic/beats/pull/47815)

  Update kubernetes cpu and memory metrics to use allocatable values instead of capacity values.
* Add extra debug logging to simplify troubleshooting in prometheus module. [#47477](https://github.com/elastic/beats/pull/47477) [#15693](https://github.com/elastic/beats/issues/15693)
* Add resource pool id to vsphere cluster metricset. [#47883](https://github.com/elastic/beats/pull/47883)
* Add `last_terminated_exitcode` to `kubernetes.container.status`. [#47968](https://github.com/elastic/beats/pull/47968)
* Report memory pressure stall information (PSI) for cgroup v2. [#48054](https://github.com/elastic/beats/pull/48054)

  Add memory PSI metrics to system.process.cgroup, complementing existing CPU and IO pressure metrics for cgroupv2.

**Osquerybeat**

* Add browser_history table to Osquery extension for cross-platform browser history analysis. [#47117](https://github.com/elastic/beats/pull/47117)
* Add amcache hive support for osquery extension in Windows. [#46996](https://github.com/elastic/beats/pull/46996)
* Add status reporting for osquerybeat lifecycle and osqueryd management. [#47472](https://github.com/elastic/beats/pull/47472)
* Add osqueryd process health monitoring with metrics exposed via beats monitoring endpoint. [#47474](https://github.com/elastic/beats/pull/47474)
* Upgrade osquery version to 5.19.0. [#48040](https://github.com/elastic/beats/pull/48040)
* Add record filtering/scoping support to the osquery extension. [#47396](https://github.com/elastic/beats/pull/47396)
* Update documentation for the amcache tables in the osquery extension. [#47748](https://github.com/elastic/beats/pull/47748)
* Add marshalling support for embedded structs in the osquery extension. [#47746](https://github.com/elastic/beats/pull/47746)
* Update column definition encoding to support embedded structs. [#47758](https://github.com/elastic/beats/pull/47758)

**Packetbeat**

* Add status reporter interface to packetbeat. [#45732](https://github.com/elastic/beats/pull/45732)

**Winlogbeat**

* Add &#39;process.args_count&#39; to winlogbeat windows security ingest pipeline. [#47266](https://github.com/elastic/beats/pull/47266)


### Fixes [beats-9.3.0-fixes]


**All**

* Add msync syscall to seccomp whitelist for BadgerDB persistent cache. [#48229](https://github.com/elastic/beats/pull/48229)
* Fix windows install script to properly migrate legacy state data. [#48293](https://github.com/elastic/beats/pull/48293)
* Remove use of github.com/elastic/elastic-agent-client from OSS Beats. [#48353](https://github.com/elastic/beats/pull/48353) 

**Filebeat**

* Fix an issue in the initialization of exporters for input metrics that could cause an unexpected number of exporter connections to be created. [#48321](https://github.com/elastic/beats/pull/48321)
* Prevent panic during startup if dissect processor has invalid field name in tokenizer. [#47839](https://github.com/elastic/beats/pull/47839)
* Fix AD Entity Analytics failing to fetch users when Base DN contains a group CN. [#48395](https://github.com/elastic/beats/pull/48395) 
* Fix filebeat goroutine leak when using harvester_limit. [#48445](https://github.com/elastic/beats/pull/48445)

**Metricbeat**

* Improve defensive checks to prevent panics in meraki module. [#47950](https://github.com/elastic/beats/pull/47950)
* Remove GCP Billing timestamp functions. [#47963](https://github.com/elastic/beats/pull/47963)
* Harden Prometheus metrics parser against panics caused by malformed input data. [#47914](https://github.com/elastic/beats/pull/47914)
* Add bounds checking to Zookeeper server module to prevent index-out-of-range panics. [#47915](https://github.com/elastic/beats/pull/47915)
* Fix panic in graphite server metricset when metric has fewer parts than template expects. [#47916](https://github.com/elastic/beats/pull/47916)
* Skip regions with no permission to query for AWS CloudWatch metrics. [#48135](https://github.com/elastic/beats/pull/48135)
* Enforce configurable size limits on incoming requests for remote_write metricset (max_compressed_body_bytes, max_decoded_body_bytes). [#48218](https://github.com/elastic/beats/pull/48218)
* Autoops agent to shutdown when it can&#39;t recover from the http errors. [#48292](https://github.com/elastic/beats/pull/48292)
* Add missing vector metrics in Autoops agent. [#48365](https://github.com/elastic/beats/pull/48365)
* Stack Monitoring now trims trailing slashes from host URLs for simplicity. [#48430](https://github.com/elastic/beats/pull/48430) [#48426](https://github.com/elastic/beats/issues/48426)
* Flatten AutoOps Cluster Settings to avoid unnecessary nesting and information. [#48454](https://github.com/elastic/beats/pull/48454) [#48453](https://github.com/elastic/beats/issues/48453)

**Osquerybeat**

* Fix zero time encoding for Unix timestamps. [#47970](https://github.com/elastic/beats/pull/47970)

**Packetbeat**

* RPC fragment bounds checking and sanitization. [#47803](https://github.com/elastic/beats/pull/47803) 
* Add check for incorrect length values in PostgreSQL datarow parser. [#47872](https://github.com/elastic/beats/pull/47872)
* Verify and cap memcache UDP fragment counts. [#47874](https://github.com/elastic/beats/pull/47874)
* Fix bounds checking in MongoDB protocol parser to prevent panics. [#47925](https://github.com/elastic/beats/pull/47925)

