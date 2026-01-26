## 9.1.3 [beats-9.1.3-release-notes]

### Features and enhancements [beats-9.1.3-features-enhancements]

**Affecting all Beats**

- Update Go version to 1.24.5. [45403]({{beats-pull}}45403)
- Improve trimming of BOM from UTF-8 data in the libbeat `reader/readfile.EncoderReader`. [45742]({{beats-pull}}45742)

**Filebeat**

- Add mechanism to allow HTTP JSON templates to terminate without logging an error. [45664]({{beats-issue}}45664) [45810]({{beats-pull}}45810)
- Add status reporting support for AWS S3 input. [45748]({{beats-pull}}45748)

### Fixes [beats-9.1.3-fixes]

**Affecting all Beats**

- Fixed case where Beats would silently fail due to invalid input configuration, now the error is correctly reported. [43118]({{beats-issue}}43118) [45733]({{beats-pull}}45733)

**Filebeat**

- Fix handling of unnecessary BOM in UTF-8 text received by o365audit input. [44327]({{beats-issue}}44327) [45739]({{beats-pull}}45739)
- Fix reading journald messages with more than 4kb. [45511]({{beats-issue}}45511) [46017]({{beats-pull}}46017)
- Restore the Streaming input on Windows. [46031]({{beats-pull}}46031)
- Fix termination of input on API errors. [45999]({{beats-pull}}45999)

**Metricbeat**

- Changed Kafka protocol version from 3.6.0 to 2.1.0 to fix compatibility with Kafka 2.x brokers. [45761]({{beats-pull}}45761)
- Enhance behavior of `sanitizeError`: replace sensitive info even if it is escaped and add pattern-based sanitization. [45857]({{beats-pull}}45857)

