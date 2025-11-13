## 9.2.2-TEST-ONLY [beats-release-notes-9.2.2-TEST-ONLY]



### Features and enhancements [beats-9.2.2-TEST-ONLY-features-enhancements]


**Filebeat**

* Add support for DPoP authentication for the CEL and HTTP JSON inputs. [#47622](https://github.com/elastic/beats/pull/47622) 
* Improve logging of cache processor and add ignore failure option. [#47622](https://github.com/elastic/beats/pull/47622) 


### Fixes [beats-9.2.2-TEST-ONLY-fixes]


**Filebeat**

* Handle and remove BOM during JSON parsing in azureblobstorage and gcs inputs. [#47622](https://github.com/elastic/beats/pull/47622) 
* Fixed an issue where filebeat could hang during shutdown when using the filestream input. [#47622](https://github.com/elastic/beats/pull/47622) 
* Fix double locking in translate_ldap_attribute processor and improve logging. [#47622](https://github.com/elastic/beats/pull/47622) 

**Metricbeat**

* [Cloud Connect] Use cluster.metadata.display_name as cluster name if set. [#47622](https://github.com/elastic/beats/pull/47622) 

