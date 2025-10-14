## 9.1.5 [beats-9.1.5-release-notes]

_This release also includes: [Breaking changes](/release-notes/breaking-changes.md#beats-9.1.5-breaking-changes)_

### Features and enhancements [beats-9.1.5-features-enhancements]

**Filebeat**

- Hints based autodiscover now sets `close.on_state_change.removed: false` in the default configuration to avoid missing the last log lines from a container. [34789]({{beats-issue}}34789) [46695]({{beats-pull}}46695)

**Metricbeat**

- Log every 401 response from Kubernetes API Server. [42714]({{beats-pull}}42714)
- Add new metrics to vSphere Virtual Machine dataset (CPU usage percentage, disk average usage, disk read/write rate, number of disk reads/writes, memory usage percentage). [44205]({{beats-pull}}44205)
- Added checks for the Resty response object in all Meraki module API calls to ensure proper handling of nil responses. [44193]({{beats-pull}}44193)
- Add latency config option to Azure Monitor module. [44366]({{beats-pull}}44366)
- Increase default polling period for MongoDB module from 10s to 60s. [44781]({{beats-pull}}44781)
- Upgrade github.com/microsoft/go-mssqldb version from v1.7.2 to v1.8.2. [44990]({{beats-pull}}44990)
- Add NTP response validation for system/ntp module. [46184]({{beats-pull}}46184)
- Add vertexai_logs metricset to GCP for prompt response collection from VertexAI service. [46383]({{beats-pull}}46383)
- Add default timegrain to Azure Storage Account metricset. [46786]({{beats-pull}}46786)

### Fixes [beats-9.1.5-fixes]

**Affecting all Beats**

- Fixed a panic in the Kafka output that could occur when shutting down while final events were being published. [46109]({{beats-issue}}46109) [46446]({{beats-pull}}46446)

**Filebeat**

- [Journald input] Fix reading all files in a folder and watching for new ones. [46657]({{beats-issue}}46657) [46682]({{beats-pull}}46682)
- The UDP input now fails if it cannot bind to the configured port and its status is set to failed when running under Elastic Agent. [37216]({{beats-issue}}37216) [46302]({{beats-pull}}46302)
- The Unix input now fails on errors listening to the socket and its status is set to failed when running under Elastic Agent. [46302]({{beats-pull}}46302)
- In Filestream, setting `clean_inactive: 0` does not re-ingest all files on startup any more. [45601]({{beats-issue}}45601) [46373]({{beats-pull}}46373)
- Fix metrics from TCP & UDP inputs when the port number is > 32767 [46486]({{beats-pull}}46486)
- [azure-eventhub] Fix handling of connection strings with entity path. [43715]({{beats-issue}}43715) [43716]({{beats-pull}}43716)

**Winlogbeat**

- Fix EventLog reset logic to not close renderers. [46376]({{beats-pull}}46376) {issue}45750{45750}
