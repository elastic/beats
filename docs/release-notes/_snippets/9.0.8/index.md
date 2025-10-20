## 9.0.8 [beats-9.0.8-release-notes]

_This release also includes: [Breaking changes](/release-notes/breaking-changes.md#beats-9.0.8-breaking-changes)_

### Features and enhancements [beats-9.0.8-features-enhancements]

**Metricbeat**

- Upgrade github.com/microsoft/go-mssqldb version from v1.7.2 to v1.8.2. [44990]({{beats-pull}}44990)
- Add SSL support for SQL modules: drivers Mysql, PostgreSQL, and MSSQL. [44748]({{beats-pull}}44748)
- Add NTP response validation for system/ntp module. [46184]({{beats-pull}}46184)
- Add vertexai_logs metricset to GCP for prompt response collection from VertexAI service. [46383]({{beats-pull}}46383)
- Add default timegrain to Azure Storage Account metricset. [46786]({{beats-pull}}46786)

### Fixes [beats-9.0.8-fixes]

**Affecting all Beats**

- Update github.com/docker/docker to v28.3.3 [46334]({{beats-pull}}46334)
- Fixed a panic in the Kafka output that could occur when shutting down while final events were being published. [46109]({{beats-issue}}46109) [46446]({{beats-pull}}46446)

**Filebeat**

- The UDP input now fails if it cannot bind to the configured port and its status is set to failed when running under Elastic Agent. [37216]({{beats-issue}}37216) [46302]({{beats-pull}}46302)
- The Unix input now fails on errors listening to the socket and its status is set to failed when running under Elastic Agent. [46302]({{beats-pull}}46302)
- [Journald input] Fix reading all files in a folder and watching for new ones. [46657]({{beats-issue}}46657) [46682]({{beats-pull}}46682)
- [azure-eventhub] Fix handling of connection strings with entity path. [43715]({{beats-issue}}43715) [43716]({{beats-pull}}43716)

**Metricbeat**

- Do not log an error if metadata enrichment is disabled for K8's module. [46536]({{beats-pull}}46536)
- Fix Azure Monitor wildcard metrics names timegrain issue by using the first, smallest timegrain; fix nil pointer issue. [46145]({{beats-pull}}46145)

**Winlogbeat**

- Fix EventLog reset logic to not close renderers. [46376]({{beats-pull}}46376) {issue}45750{45750}
