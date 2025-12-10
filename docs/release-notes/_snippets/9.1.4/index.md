## 9.1.4 [beats-9.1.4-release-notes]

### Features and enhancements [beats-9.1.4-features-enhancements]

**Filebeat**

- Improve HTTP JSON health status logic for empty template results. [46332]({{beats-pull}}46332)
- Improve CEL input documentation of authentication options. [46253]({{beats-pull}}46253)
- Add status reporting support for Azure Event Hub v2 input. [44846]({{beats-pull}}44846)
- Add documentation for device collection in Entity Analytics Active Directory Filebeat's input. [46363]({{beats-pull}}46363)

**Metricbeat**

- Add support for Kafka 4.0 in the Kafka module. [44723]({{beats-pull}}44723)

### Fixes [beats-9.1.4-fixes]

**Affecting all Beats**

- Fix a race condition during metrics initialization which could cause a panic. [45822]({{beats-issue}}45822) [46054]({{beats-pull}}46054)
- Fixed a panic when the beat restarts itself by adding 'eventfd2' to default seccomp policy [46372]({{beats-issue}}46372)
- Update github.com/go-viper/mapstructure/v2 to v2.4.0 [46335]({{beats-pull}}46335)
- Update Go version to 1.24.7 [46070]({{beats-pull}}46070).
- Update github.com/docker/docker to v28.3.3 [46334]({{beats-pull}}46334)

**Filebeat**

- Fix wrongly emitted missing input ID warning [42969]({{beats-issue}}42969) [45747]({{beats-pull}}45747)
- Fix race condition that could cause Filebeat to hang during shutdown after failing to startup [45034]({{beats-issue}}45034) [46331]({{beats-pull}}46331)
- Fixed hints autodiscover for Docker when the configuration is only `hints.enabled: true`. [45156]({{beats-issue}}45156) [45864]({{beats-pull}}45864)

**Metricbeat**

- Fix an issue where the conntrack metricset entries field reported a count inflated by a factor of the number of CPU cores. [46138]({{beats-issue}}46138) [46140]({{beats-pull}}46140)

**Winlogbeat**

- Fix forwarded event handling and add channel error resilience. [46190]({{beats-pull}}46190)
