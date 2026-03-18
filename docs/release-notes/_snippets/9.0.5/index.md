## 9.0.5 [beats-9.0.5-release-notes]

### Features and enhancements [beats-9.0.5-features-enhancements]

**Filebeat**

- Enhanced HTTPJSON input error logging with structured error metadata conforming to Elastic Common Schema (ECS) conventions. [45653]({{beats-pull}}45653)

**Metricbeat**

- Improve error messages in AWS Health. [45408]({{beats-pull}}45408)

### Fixes [beats-9.0.5-fixes]

**Auditbeat**

- Auditd: Request status from a separate socket to avoid data congestion. [41207]({{beats-pull}}41207)
- Fix potential data loss in `add_session_metadata`. [42795]({{beats-pull}}42795)

**Metricbeat**

- Fix URL construction to handle query parameters properly in GET requests for Jolokia. [45620]({{beats-pull}}45620)
