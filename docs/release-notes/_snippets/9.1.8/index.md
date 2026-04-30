## 9.1.8 [beats-release-notes-9.1.8]



### Features and enhancements [beats-9.1.8-features-enhancements]


**All**

* Include whether Beat is running from a FIPS distribution in User Agent. [#47409](https://github.com/elastic/beats/pull/47409)

**Filebeat**

* Improve logging of cache processor and add ignore failure option. [#47565](https://github.com/elastic/beats/pull/47565)

### Fixes [beats-9.1.8-fixes]


**All**

* Fix a fatal startup error in Beats Receivers caused by truncation of long UTF-8 hostnames. [#47713](https://github.com/elastic/beats/pull/47713) 

**Filebeat**

* Handle and remove BOM during JSON parsing in Azure Blob Storage and GCS inputs. [#47508](https://github.com/elastic/beats/pull/47508)
* Fix an issue where Filebeat could hang during shutdown when using the filestream input. [#47518](https://github.com/elastic/beats/pull/47518)
* Fix double locking in `translate_ldap_attribute` processor and improve logging. [#47585](https://github.com/elastic/beats/pull/47585)
* Fix possible data corruption in TCP, Syslog and Unix inputs. [#47618](https://github.com/elastic/beats/pull/47618) 
* Skip AWS S3 test events in Filebeat AWS S3 input. [#47635](https://github.com/elastic/beats/pull/47635) 

**Metricbeat**

* [Cloud Connect] Use `cluster.metadata.display_name` as cluster name if set. [#47440](https://github.com/elastic/beats/pull/47440)

