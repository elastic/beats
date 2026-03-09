## 9.1.1 [beats-9.1.1-release-notes]

### Features and enhancements [beats-9.1.1-features-enhancements]

**Filebeat**

- Log CEL single object evaluation results as ECS compliant documents where possible. [45254]({{beats-issue}}45254) [45399]({{beats-pull}}45399)
- Enhanced HTTPJSON input error logging with structured error metadata conforming to Elastic Common Schema (ECS) conventions. [45653]({{beats-pull}}45653)

### Fixes [beats-9.1.1-fixes]

**Filebeat**

- Fix a panic in the winlog input that prevented it from starting. [45693]({{beats-issue}}45693) [45730]({{beats-pull}}45730)

**Metricbeat**

- Improve error messages in AWS Health [45408]({{beats-pull}}45408)
- Fix URL construction to handle query parameters properly in GET requests for Jolokia [45620]({{beats-pull}}45620)

