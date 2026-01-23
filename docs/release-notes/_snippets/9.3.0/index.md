## 9.3.0 [beats-release-notes-9.3.0]

_This release also includes: [Deprecations](/release-notes/deprecations.md#beats-9.3.0-deprecations)._


### Features and enhancements [beats-9.3.0-features-enhancements]


**All**

* Introduce cloud connectors flow. [#48508](https://github.com/elastic/beats/pull/48508) [#48480](https://github.com/elastic/beats/pull/48480) 
* Make beats receivers emit status for their subcomponents. [#48508](https://github.com/elastic/beats/pull/48508) [#48480](https://github.com/elastic/beats/pull/48480) 
* Add GUID translation, base DN inference, and SSPI authentication to LDAP processor. [#47827](https://github.com/elastic/beats/pull/47827) 
* Add `events.failure_store` metric to track events sent to Elasticsearch failure store. [#48068](https://github.com/elastic/beats/pull/48068) [#47164](https://github.com/elastic/beats/issues/47164)

**Filebeat**

* Add support for direct HTTP request rate limit setting in CEL input. [#48508](https://github.com/elastic/beats/pull/48508) [#48480](https://github.com/elastic/beats/pull/48480) 
* Allow adding file owner and group to the events&#39; meta fields on Unix systems. [#47331](https://github.com/elastic/beats/pull/47331) [#43226](https://github.com/elastic/beats/issues/43226)
* Add AWS auth method for CEL and HTTP JSON inputs. [#48508](https://github.com/elastic/beats/pull/48508) [#48480](https://github.com/elastic/beats/pull/48480) 
* Add client secret authentication method for Azure Event Hub and storage in Filebeat. [#48508](https://github.com/elastic/beats/pull/48508) [#48480](https://github.com/elastic/beats/pull/48480) 
* Add client address and name to submitted Redis slowlogs. [#48508](https://github.com/elastic/beats/pull/48508) [#48480](https://github.com/elastic/beats/pull/48480) 
* Log unpublished event count and exit publish loop on input context cancellation. [#48508](https://github.com/elastic/beats/pull/48508) [#48480](https://github.com/elastic/beats/pull/48480) 
* Upgrade CEL mito library to v1.24.0. [#48508](https://github.com/elastic/beats/pull/48508) [#48480](https://github.com/elastic/beats/pull/48480) 
* Add OTEL metrics to CEL inputs. [#48508](https://github.com/elastic/beats/pull/48508) [#48480](https://github.com/elastic/beats/pull/48480) 
* Improving input error reporting to Elastic Agent, specially when pipeline configurations are incorrect. [#47905](https://github.com/elastic/beats/pull/47905) [#45649](https://github.com/elastic/beats/issues/45649)
* Azure-eventhub-v2-add-support-for-amqp-over-websocket-and-https-proxy. [#47956](https://github.com/elastic/beats/pull/47956) [#47823](https://github.com/elastic/beats/issues/47823)
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
* Add file-based auth provider for CEL and HTTP JSON inputs. [#48508](https://github.com/elastic/beats/pull/48508) [#48480](https://github.com/elastic/beats/pull/48480) [#47506](https://github.com/elastic/beats/issues/47506)

  The CEL and HTTP JSON inputs now support reading authentication tokens from
  files, enabling integration with various secret providers like Vault,
  Kubernetes secret projections, etc. Tokens are automatically refreshed based on
  a configurable interval without requiring restarts.
  

**Metricbeat**

* K8s_container_allocatable. [#48508](https://github.com/elastic/beats/pull/48508) [#48480](https://github.com/elastic/beats/pull/48480) 

  Updates kubernetes cpu and memory metrics to use allocatable values instead of capacity values.
* Add extra debug logging to simplify troubleshooting in prometheus module. [#47477](https://github.com/elastic/beats/pull/47477) [#15693](https://github.com/elastic/beats/issues/15693)
* Add resource pool id to vsphere cluster metricset. [#48508](https://github.com/elastic/beats/pull/48508) [#48480](https://github.com/elastic/beats/pull/48480) 
* Add `last_terminated_exitcode` to `kubernetes.container.status`. [#48508](https://github.com/elastic/beats/pull/48508) [#48480](https://github.com/elastic/beats/pull/48480) 
* Report memory pressure stall information (PSI) for cgroup v2. [#48508](https://github.com/elastic/beats/pull/48508) [#48480](https://github.com/elastic/beats/pull/48480) 

  Add memory PSI metrics to system.process.cgroup, complementing existing CPU and IO pressure metrics for cgroupv2

**Osquerybeat**

* Add browser_history table to Osquery extension for cross-platform browser history analysis. [#48508](https://github.com/elastic/beats/pull/48508) [#48480](https://github.com/elastic/beats/pull/48480) 
* Add amcache hive support for osquery extension in Windows. [#48508](https://github.com/elastic/beats/pull/48508) [#48480](https://github.com/elastic/beats/pull/48480) 
* Add status reporting for osquerybeat lifecycle and osqueryd management. [#48508](https://github.com/elastic/beats/pull/48508) [#48480](https://github.com/elastic/beats/pull/48480) 
* Add osqueryd process health monitoring with metrics exposed via beats monitoring endpoint. [#48508](https://github.com/elastic/beats/pull/48508) [#48480](https://github.com/elastic/beats/pull/48480) 
* Upgrade osquery to 5.19.0. [#48508](https://github.com/elastic/beats/pull/48508) [#48480](https://github.com/elastic/beats/pull/48480) 
* Add record filtering/scoping support to the osquery extension. [#48508](https://github.com/elastic/beats/pull/48508) [#48480](https://github.com/elastic/beats/pull/48480) 
* Updates documentation for the amcache tables in the osquery extension. [#48508](https://github.com/elastic/beats/pull/48508) [#48480](https://github.com/elastic/beats/pull/48480) 
* Add marshalling support for embedded structs in the osquery extension. [#48508](https://github.com/elastic/beats/pull/48508) [#48480](https://github.com/elastic/beats/pull/48480) 
* Update column definition encoding to support embedded structs. [#48508](https://github.com/elastic/beats/pull/48508) [#48480](https://github.com/elastic/beats/pull/48480) 

**Packetbeat**

* Add status reporter interface to packetbeat. [#48508](https://github.com/elastic/beats/pull/48508) [#48480](https://github.com/elastic/beats/pull/48480) 
* Ipfrag2. [#48508](https://github.com/elastic/beats/pull/48508) [#48480](https://github.com/elastic/beats/pull/48480) 

**Winlogbeat**

* Adds &#39;process.args_count&#39; to winlogbeat windows security ingest pipeline. [#48508](https://github.com/elastic/beats/pull/48508) [#48480](https://github.com/elastic/beats/pull/48480) 


### Fixes [beats-9.3.0-fixes]


**All**

* Add msync syscall to seccomp whitelist for BadgerDB persistent cache. [#48508](https://github.com/elastic/beats/pull/48508) [#48480](https://github.com/elastic/beats/pull/48480) 
* Fix windows install script to properly migrate legacy state data. [#48508](https://github.com/elastic/beats/pull/48508) [#48480](https://github.com/elastic/beats/pull/48480) 
* Remove use of github.com/elastic/elastic-agent-client from OSS Beats. [#48353](https://github.com/elastic/beats/pull/48353) 

**Filebeat**

* Fixed an issue in the initialization of exporters for input metrics that could cause an unexpected number of exporter connections to be created. [#48508](https://github.com/elastic/beats/pull/48508) [#48480](https://github.com/elastic/beats/pull/48480) 
* Prevent panic during startup if dissect processor has invalid field name in tokenizer. [#48508](https://github.com/elastic/beats/pull/48508) [#48480](https://github.com/elastic/beats/pull/48480) 
* Fix AD Entity Analytics failing to fetch users when Base DN contains a group CN. [#48395](https://github.com/elastic/beats/pull/48395) 
* Fix filebeat goroutine leak when using harvester_limit. [#48508](https://github.com/elastic/beats/pull/48508) [#48480](https://github.com/elastic/beats/pull/48480) 

**Metricbeat**

* Improve defensive checks to prevent panics in meraki module. [#48508](https://github.com/elastic/beats/pull/48508) [#48480](https://github.com/elastic/beats/pull/48480) 
* Remove GCP Billing timestamp functions. [#48508](https://github.com/elastic/beats/pull/48508) [#48480](https://github.com/elastic/beats/pull/48480) 
* Harden Prometheus metrics parser against panics caused by malformed input data. [#48508](https://github.com/elastic/beats/pull/48508) [#48480](https://github.com/elastic/beats/pull/48480) 
* Add bounds checking to Zookeeper server module to prevent index-out-of-range panics. [#48508](https://github.com/elastic/beats/pull/48508) [#48480](https://github.com/elastic/beats/pull/48480) 
* Fix panic in graphite server metricset when metric has fewer parts than template expects. [#48508](https://github.com/elastic/beats/pull/48508) [#48480](https://github.com/elastic/beats/pull/48480) 
* Skip regions with no permission to query for AWS CloudWatch metrics. [#48508](https://github.com/elastic/beats/pull/48508) [#48480](https://github.com/elastic/beats/pull/48480) 
* Enforces configurable size limits on incoming requests for remote_write metricset (max_compressed_body_bytes, max_decoded_body_bytes). [#48508](https://github.com/elastic/beats/pull/48508) [#48480](https://github.com/elastic/beats/pull/48480) 
* Autoops agent to shutdown when it can&#39;t recover from the http errors. [#48508](https://github.com/elastic/beats/pull/48508) [#48480](https://github.com/elastic/beats/pull/48480) 
* Added missing vector metrics in autoops agent. [#48508](https://github.com/elastic/beats/pull/48508) [#48480](https://github.com/elastic/beats/pull/48480) 
* Stack Monitoring now trims trailing slashes from host URLs for simplicity. [#48508](https://github.com/elastic/beats/pull/48508) [#48480](https://github.com/elastic/beats/pull/48480) [#48426](https://github.com/elastic/beats/issues/48426)
* Flatten AutoOps Cluster Settings to avoid unnecessary nesting and information. [#48508](https://github.com/elastic/beats/pull/48508) [#48480](https://github.com/elastic/beats/pull/48480) [#48453](https://github.com/elastic/beats/issues/48453)

**Osquerybeat**

* Fixes zero time encoding for unix timestamps. [#48508](https://github.com/elastic/beats/pull/48508) [#48480](https://github.com/elastic/beats/pull/48480) 

**Packetbeat**

* Rpc_fragment_sanitization. [#47803](https://github.com/elastic/beats/pull/47803) 
* Add check for incorrect length values in postgres datarow parser. [#48508](https://github.com/elastic/beats/pull/48508) [#48480](https://github.com/elastic/beats/pull/48480) 
* Verify and cap memcache udp fragment counts. [#48508](https://github.com/elastic/beats/pull/48508) [#48480](https://github.com/elastic/beats/pull/48480) 
* Fix bounds checking in MongoDB protocol parser to prevent panics. [#48508](https://github.com/elastic/beats/pull/48508) [#48480](https://github.com/elastic/beats/pull/48480) 

