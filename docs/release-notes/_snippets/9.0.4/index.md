## 9.0.4 [beats-9.0.4-release-notes]

### Features and enhancements [beats-9.0.4-features-enhancements]

**Filebeat**

- Add Fleet status updating to GCS input. [44273]({{beats-issue}}44273) [44508]({{beats-pull}}44508)
- Add Fleet status update functionality to udp input. [44419]({{beats-issue}}44419) [44785]({{beats-pull}}44785)
- Add Fleet status update functionality to tcp input. [44420]({{beats-issue}}44420) [44786]({{beats-pull}}44786)
- Add Fleet status updating to Azure Blob Storage input. [44268]({{beats-issue}}44268) [44945]({{beats-pull}}44945)
- Add Fleet status updating to HTTP JSON input. [44282]({{beats-issue}}44282) [44365]({{beats-pull}}44365)
- Add input metrics to Azure Blob Storage input. [36641]({{beats-issue}}36641) [43954]({{beats-pull}}43954)
- Add support for websocket keep_alive heartbeat in the streaming input. [42277]({{beats-issue}}42277) [44204]({{beats-pull}}44204)
- Add missing "text/csv" content-type filter support in GCS input. [44922]({{beats-issue}}44922) [44923]({{beats-pull}}44923)

**Heartbeat**

- Upgrade Node version to latest LTS v20.19.3. [45087]({{beats-pull}}45087)
- Add base64 encoding option to inline monitors. [45100]({{beats-pull}}45100)

**Metricbeat**

- Upgrade github.com/microsoft/go-mssqldb version from v1.7.2 to v1.8.2. [44990]({{beats-pull}}44990)

### Fixes [beats-9.0.4-fixes]

**Affecting all Beats**

- The Elasticsearch output now correctly applies exponential backoff when being throttled by 429s ("too many requests") from Elasticsarch. [36926]({{beats-issue}}36926) [45073]({{beats-pull}}45073)

**Winlogbeat**

- Fix EvtVarTypeAnsiString conversion. [44026]({{beats-pull}}44026)
