## 9.1.2 [beats-9.1.2-release-notes]

### Features and enhancements [beats-9.1.2-features-enhancements]

**Filebeat**

- Add status reporting support for AWS CloudWatch input. [45679]({{beats-pull}}45679)

**Winlogbeat**

- Render data values in XML renderer. [44132]({{beats-pull}}44132)

### Fixes [beats-9.1.2-fixes]

**Filebeat**

- Fix error handling in ABS input when both root level `max_workers` and `batch_size` are empty. [45680]({{beats-issue}}45680) [45743]({{beats-pull}}45743)

