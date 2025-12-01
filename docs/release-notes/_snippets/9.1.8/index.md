## 9.1.8 [beats-release-notes-9.1.8]



### Features and enhancements [beats-9.1.8-features-enhancements]


**All**

* Include whether Beat is running from a FIPS distribution in User Agent. [#47830](https://github.com/elastic/beats/pull/47830) [#47753](https://github.com/elastic/beats/pull/47753) [#46437](https://github.com/elastic/beats/issues/46437)

**Filebeat**

* Improve logging of cache processor and add ignore failure option. [#47830](https://github.com/elastic/beats/pull/47830) [#47753](https://github.com/elastic/beats/pull/47753) [#46437](https://github.com/elastic/beats/issues/46437)


### Fixes [beats-9.1.8-fixes]


**All**

* Fix a fatal startup error in Beats Receivers caused by truncation of long UTF-8 hostnames. [#47830](https://github.com/elastic/beats/pull/47830) [#47753](https://github.com/elastic/beats/pull/47753) [#46437](https://github.com/elastic/beats/issues/46437)

**Filebeat**

* Handle and remove BOM during JSON parsing in azureblobstorage and gcs inputs. [#47830](https://github.com/elastic/beats/pull/47830) [#47753](https://github.com/elastic/beats/pull/47753) [#46437](https://github.com/elastic/beats/issues/46437)
* Fixed an issue where filebeat could hang during shutdown when using the filestream input. [#47830](https://github.com/elastic/beats/pull/47830) [#47753](https://github.com/elastic/beats/pull/47753) [#46437](https://github.com/elastic/beats/issues/46437)
* Fix double locking in translate_ldap_attribute processor and improve logging. [#47830](https://github.com/elastic/beats/pull/47830) [#47753](https://github.com/elastic/beats/pull/47753) [#46437](https://github.com/elastic/beats/issues/46437)
* Fix possible data corruption in tcp, syslog and unix inputs. [#47830](https://github.com/elastic/beats/pull/47830) [#47753](https://github.com/elastic/beats/pull/47753) [#46437](https://github.com/elastic/beats/issues/46437)
* Skip s3 test events in filebeat s3 input. [#47830](https://github.com/elastic/beats/pull/47830) [#47753](https://github.com/elastic/beats/pull/47753) [#46437](https://github.com/elastic/beats/issues/46437)

**Metricbeat**

* [Cloud Connect] Use cluster.metadata.display_name as cluster name if set. [#47830](https://github.com/elastic/beats/pull/47830) [#47753](https://github.com/elastic/beats/pull/47753) [#46437](https://github.com/elastic/beats/issues/46437)

