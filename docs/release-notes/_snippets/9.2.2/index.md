## 9.2.2 [beats-release-notes-9.2.2]

_This release also includes: [Breaking changes](/release-notes/breaking-changes.md#beats-9.2.2-breaking-changes)._


### Features and enhancements [beats-9.2.2-features-enhancements]


**All**

* Include whether Beat is running from a FIPS distribution in User Agent. [#47763](https://github.com/elastic/beats/pull/47763) 

**Filebeat**

* Add support for DPoP authentication for the CEL and HTTP JSON inputs. [#47763](https://github.com/elastic/beats/pull/47763) 
* Improve logging of cache processor and add ignore failure option. [#47763](https://github.com/elastic/beats/pull/47763) 


### Fixes [beats-9.2.2-fixes]


**All**

* Fix a fatal startup error in Beats Receivers caused by truncation of long UTF-8 hostnames. [#47763](https://github.com/elastic/beats/pull/47763) 
* Not being able to start the add_docker_metadata processor is now consistently a non-fatal error when Docker is not available. [#47763](https://github.com/elastic/beats/pull/47763) 

**Filebeat**

* [Filestream] ensure harvester always restarts if the file has not been fully ingested. [#47107](https://github.com/elastic/beats/pull/47107) [#46923](https://github.com/elastic/beats/issues/46923)
* Handle and remove BOM during JSON parsing in azureblobstorage and gcs inputs. [#47763](https://github.com/elastic/beats/pull/47763) 
* Fixed an issue where filebeat could hang during shutdown when using the filestream input. [#47763](https://github.com/elastic/beats/pull/47763) 
* Fix double locking in translate_ldap_attribute processor and improve logging. [#47763](https://github.com/elastic/beats/pull/47763) 
* Fix possible data corruption in tcp, syslog and unix inputs. [#47763](https://github.com/elastic/beats/pull/47763) 
* Skip s3 test events in filebeat s3 input. [#47763](https://github.com/elastic/beats/pull/47763) 

**Metricbeat**

* [Cloud Connect] Use cluster.metadata.display_name as cluster name if set. [#47763](https://github.com/elastic/beats/pull/47763) 

