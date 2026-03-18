## 9.0.3 [beats-9.0.3-release-notes]

### Features and enhancements [beats-9.0.3-features-enhancements]

**Affecting all Beats**

- Update to Go 1.24.4. [44696]({{beats-pull}}44696)

**Filebeat**

- Fix handling of ADC (Application Default Credentials) metadata server credentials in HTTPJSON input. [44349]({{beats-issue}}44349) [44436]({{beats-pull}}44436)
- Fix handling of ADC (Application Default Credentials) metadata server credentials in CEL input. [44349]({{beats-issue}}44349) [44571]({{beats-pull}}44571)
- Filestream now logs at level warn the number of files that are too small to be ingested [44751]({{beats-pull}}44751)

**Metricbeat**

- Add check for http error codes in the Metricbeat's Prometheus query submodule [44493]({{beats-pull}}44493)
- Increase default polling period for MongoDB module from 10s to 60s [44781]({{beats-pull}}44781)

### Fixes [beats-9.0.3-fixes]

**Affecting all Beats**

- Fix `dns` processor to handle IPv6 server addresses properly. [44526]({{beats-pull}}44526)
- Fix an issue where the Kafka output could get stuck if a proxied connection to the Kafka cluster was reset. [44606]({{beats-issue}}44606)
- Use Debian 11 to build linux/arm to match linux/amd64. Upgrades linux/arm64's statically linked glibc from 2.28 to 2.31. [44816]({{beats-issue}}44816)

**Filebeat**

- Handle special values of accountExpires in the Activedirectory Entity Analytics provider. [43364]({{beats-pull}}43364)
- Fix status reporting panic in GCP Pub/Sub input. [44624]({{beats-issue}}44624) [44625]({{beats-pull}}44625)
- If a Filestream input fails to be created, its ID is removed from the list of running input IDs [44697]({{beats-pull}}44697)
- Fix timeout handling by Crowdstrike streaming input. [44720]({{beats-pull}}44720)
- Ensure DEPROVISIONED Okta entities are published by Okta entityanalytics provider. [12658]({{beats-issue}}12658) [44719]({{beats-pull}}44719)
- Fix handling of cursors by the streaming input for Crowdstrike. [44364]({{beats-issue}}44364) [44548]({{beats-pull}}44548)
- Added missing "text/csv" content-type filter support in azureblobsortorage input. [44596]({{beats-issue}}44596) [44824]({{beats-pull}}44824)
- Fix unexpected EOF detection and improve memory usage. [44813]({{beats-pull}}44813)

**Heartbeat**

- Add missing dependencies to ubi9-minimal distro. [44556]({{beats-pull}}44556)

**Metricbeat**

- Fix panic in kafka consumergroup member assignment fetching when there are 0 members in consumer group. [44576]({{beats-pull}}44576)
- Sanitize error messages in Fetch method of SQL module [44577]({{beats-pull}}44577)
- Upgrade `go.mongodb.org/mongo-driver` from `v1.14.0` to `v1.17.4` to fix connection leaks in MongoDB module [44769]({{beats-pull}}44769)

