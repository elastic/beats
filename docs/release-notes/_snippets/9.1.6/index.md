## 9.1.6 [beats-release-notes-9.1.6]


### Features and enhancements [beats-9.1.6-features-enhancements]

* Improve the Prometheus helper to handle multiple content types including blank and invalid headers. [#47085](https://github.com/elastic/beats/pull/47085) 


### Fixes [beats-9.1.6-fixes]

* Prevent panic in logstash output when trying to send events while shutting down. [#46960](https://github.com/elastic/beats/pull/46960) 
* Prevent panic in replace processor for non-string values. [#47009](https://github.com/elastic/beats/pull/47009) 
* Autodiscover now correctly updates Kubernetes metadata on node and pod label changes. [#47034](https://github.com/elastic/beats/pull/47034) 
* Prevent 3s startup delay when add_cloud_metadata is used with debug logs. [#47058](https://github.com/elastic/beats/pull/47058) 
* Update elastic-agent-system-metrics to v0.13.3. [#47104](https://github.com/elastic/beats/pull/47104) 

  Removes &#34;Accurate CPU counts not available on platform&#34; log spam at the debug log level.
* Allows users to customize their data stream namespace to &#34;generic&#34;. [#47140](https://github.com/elastic/beats/pull/47140) 
* Fix defer usage for stopped status reporting. [#46916](https://github.com/elastic/beats/pull/46916) 
* Fix missing AWS cloudwatch metrics with linked accounts and same dimensions. [#46978](https://github.com/elastic/beats/pull/46978) 
* Add a fix to handle blank content-type headers in HTTP responses for Prometheus. [#47027](https://github.com/elastic/beats/pull/47027) 

