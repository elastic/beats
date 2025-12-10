## 9.0.1 [beats-9.0.1-release-notes]

### Features and enhancements [beats-9.0.1-features-enhancements]

* For all Beats: Publish `cloud.availability_zone` by `add_cloud_metadata` processor in Azure environments. [#42601]({{beats-issue}}42601) [#43618]({{beats-pull}}43618)
* Add pagination batch size support to Entity Analytics input's Okta provider in Filebeat. [#43655]({{beats-pull}}43655)
* Update CEL mito extensions version to v1.19.0 in Filebeat. [#44098]({{beats-pull}}44098)
* Upgrade node version to latest LTS v18.20.7 in Heartbeat. [#43511]({{beats-pull}}43511)
* Add `enable_batch_api` option in Azure monitor to allow metrics collection of multiple resources using Azure batch API in Metricbeat. [#41790]({{beats-pull}}41790)

### Fixes [beats-9.0.1-fixes]

* For all Beats: Handle permission errors while collecting data from Windows services and don't interrupt the overall collection by skipping affected services. [#40765]({{beats-issue}}40765) [#43665]({{beats-pull}}43665).
* Fixed WebSocket input panic on sudden network error or server crash in Filebeat. [#44063]({{beats-issue}}44063) [44068]({{beats-pull}}44068).
* [Filestream] Log the "reader closed" message on the debug level to avoid log spam in Filebeat. [#44051]({{beats-pull}}44051)
* Fix links to CEL mito extension functions in input documentation in Filebeat. [#44098]({{beats-pull}}44098)

