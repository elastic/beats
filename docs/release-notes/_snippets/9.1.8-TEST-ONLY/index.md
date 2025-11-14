## 9.1.8-TEST-ONLY [beats-release-notes-9.1.8-TEST-ONLY]



### Features and enhancements [beats-9.1.8-TEST-ONLY-features-enhancements]


**All**

* Include whether Beat is running from a FIPS distribution in User Agent. [#47644](https://github.com/elastic/beats/pull/47644) [#47645](https://github.com/elastic/beats/pull/47645) [#47618](https://github.com/elastic/beats/pull/47618) [#47550](https://github.com/elastic/beats/issues/47550)

**Filebeat**

* Improve logging of cache processor and add ignore failure option. [#47644](https://github.com/elastic/beats/pull/47644) [#47645](https://github.com/elastic/beats/pull/47645) [#47618](https://github.com/elastic/beats/pull/47618) [#47550](https://github.com/elastic/beats/issues/47550)


### Fixes [beats-9.1.8-TEST-ONLY-fixes]


**Filebeat**

* Handle and remove BOM during JSON parsing in azureblobstorage and gcs inputs. [#47644](https://github.com/elastic/beats/pull/47644) [#47645](https://github.com/elastic/beats/pull/47645) [#47618](https://github.com/elastic/beats/pull/47618) [#47550](https://github.com/elastic/beats/issues/47550)
* Fixed an issue where filebeat could hang during shutdown when using the filestream input. [#47644](https://github.com/elastic/beats/pull/47644) [#47645](https://github.com/elastic/beats/pull/47645) [#47618](https://github.com/elastic/beats/pull/47618) [#47550](https://github.com/elastic/beats/issues/47550)
* Fix double locking in translate_ldap_attribute processor and improve logging. [#47644](https://github.com/elastic/beats/pull/47644) [#47645](https://github.com/elastic/beats/pull/47645) [#47618](https://github.com/elastic/beats/pull/47618) [#47550](https://github.com/elastic/beats/issues/47550)
* Fix possible data corruption in tcp, syslog and unix inputs. [#47644](https://github.com/elastic/beats/pull/47644) [#47645](https://github.com/elastic/beats/pull/47645) [#47618](https://github.com/elastic/beats/pull/47618) [#47550](https://github.com/elastic/beats/issues/47550)

**Metricbeat**

* [Cloud Connect] Use cluster.metadata.display_name as cluster name if set. [#47644](https://github.com/elastic/beats/pull/47644) [#47645](https://github.com/elastic/beats/pull/47645) [#47618](https://github.com/elastic/beats/pull/47618) [#47550](https://github.com/elastic/beats/issues/47550)

