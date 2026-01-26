## 9.1.0 [beats-9.1.0-release-notes]

### Features and enhancements [beats-9.1.0-features-enhancements]

**Affecting all Beats**

- Added the `now` processor, which will populate the specified target field with the current timestamp. [44795]({{beats-pull}}44795)

**Filebeat**

- Refactor & cleanup with updates to default values and documentation. [41834]({{beats-pull}}41834)
- Add support for SSL and Proxy configurations for websocket type in streaming input. [41934]({{beats-pull}}41934)
- Filestream take over now supports taking over states from other Filestream inputs and dynamic loading of inputs (autodiscover and Elastic Agent). There is a new syntax for the configuration, but the previous one can still be used. [42472]({{beats-issue}}42472) [42884]({{beats-issue}}42884) [42624]({{beats-pull}}42624)
- Refactor & cleanup with updates to default values and documentation. [41834]({{beats-pull}}41834)
- Segregated `max_workers` from `batch_size` in the GCS input. [44311]({{beats-issue}}44311) [44333]({{beats-pull}}44333)
- Add milliseconds to document timestamp from awscloudwatch Filebeat input [44306]({{beats-pull}}44306)
- Added support for specifying custom content-types and encodings in azureblobstorage input. [44330]({{beats-issue}}44330) [44402]({{beats-pull}}44402)
- Introduce lastSync start position to AWS CloudWatch input backed by state registry. [43251]({{beats-pull}}43251)
- Add proxy support to GCP Pub/Sub input. [44892]({{beats-pull}}44892)
- Segregated `max_workers` from `batch_size` in the azure-blob-storage input. [44491]({{beats-issue}}44491) [44992]({{beats-pull}}44992)
- Add support for relationship expansion to EntraID entity analytics provider. [43324]({{beats-issue}}43324) [44761]({{beats-pull}}44761)
- Update CEL mito extensions to v1.22.0. [45245]({{beats-pull}}45245)
- Add support for generalized token authentication to CEL input. [45359]({{beats-pull}}45359)

**Metricbeat**

- Add new metricset wmi for the windows module. [42017]({{beats-pull}}42017)
- Changed the Elasticsearch module behavior to only pull settings from non-system indices. [43243]({{beats-pull}}43243)
- Exclude dotted indices from settings pull in Elasticsearch module. [43306]({{beats-pull}}43306)
- Add a `jetstream` metricset to the NATS module [43310]({{beats-pull}}43310)
- Update NATS module compatibility. Oldest version supported is now 2.2.6 [43310]({{beats-pull}}43310)
- Upgrade Prometheus Library to v0.300.1. [43540]({{beats-pull}}43540)
- Add GCP Dataproc metadata collector in GCP module. [43518]({{beats-pull}}43518)
- Updated list of supported vSphere versions in the documentation. [43642]({{beats-pull}}43642)
- Add SSL support for sql module: drivers mysql, postgres, and mssql. [44748]({{beats-pull}}44748)
- Add VPN metrics to meraki module [44851]({{beats-pull}}44851)
- Add GCP cache for metadata collectors. [44432]({{beats-pull}}44432)

### Fixes [beats-9.1.0-fixes]

**Auditbeat**

- Fix potential data loss in add_session_metadata. [42795]({{beats-pull}}42795)
- auditbeat/fim: Fix FIM@ebpfevents for new kernels #44371. [44371]({{beats-pull}}44371)

**Filebeat**

- Log bad handshake details when websocket connection fails [41300]({{beats-pull}}41300)
- Fix aws region in aws-s3 input s3 polling mode.  [41572]({{beats-pull}}41572)
- Fix a logging regression that ignored to_files and logged to stdout. [44573]({{beats-pull}}44573)
- Fixed issue for "Root level readerConfig no longer respected" in azureblobstorage input. [44812]({{beats-issue}}44812) [44873]({{beats-pull}}44873)
- Fixed password authentication for ACL users in the Redis input of Filebeat. [44137]({{beats-pull}}44137)
- The data and logs path has changed on Windows to `$env:ProgramFiles`. See the breaking changes page for more details.

**Heartbeat**

- Added maintenance windows support for Heartbeat. [41508]({{beats-pull}}41508)

