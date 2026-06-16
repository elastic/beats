---
navigation_title: Beats
mapped_pages:
  - https://www.elastic.co/guide/en/beats/libbeat/current/release-notes.html
products:
  - id: beats
applies_to:
  stack: ga
sub:
  product: Beats
---

% Release notes include only features, enhancements, and fixes. Add breaking changes, deprecations, and known issues to the applicable release notes sections.

% ## version.next [beats-versionext-release-notes]

% ### Features and enhancements [beats-versionext-features-enhancements]
* Add support for CouchDB v3 in Metricbeat. [#22743]({{beats-issue}}22743) [#26950]({{beats-pull}}26950)

% ### Fixes [beats-versionext-fixes]

## 9.0.2 [beats-9.0.2-release-notes]

### Features and enhancements [beats-9.0.2-features-enhancements]

**Affecting all Beats**

- Update Go version to v1.24.3. [44270]({{beats-pull}}44270)

**Filebeat**

- Add support for collecting device entities in the Active Directory entity analytics provider. [44309]({{beats-pull}}44309)
- The `add_cloudfoundry_metadata` processor now uses `xxhash` instead of `SHA1` for sanitizing persistent cache filenames. Existing users will experience a one-time cache invalidation as the cache store will be recreated with the new filename format. [43964]({{beats-pull}}43964)

**Metricbeat**

- Add checks for the Resty response object in all Meraki module API calls to ensure proper handling of nil responses. [44193]({{beats-pull}}44193)
- Add a latency configuration option to the Azure Monitor module. [44366]({{beats-pull}}44366)

**Osquerybeat**

- Update osquery version to v5.15.0. [43426]({{beats-pull}}43426)

### Fixes [beats-9.0.2-fixes]

**Affecting all Beats**

- Fix the 'add_cloud_metadata' processor to better support custom certificate bundles by improving how the AWS provider HTTP client is overridden. [44189]({{beats-pull}}44189)

**Auditbeat**

- Fix a potential error in the system/package component that could occur during internal package database schema migration. [44294]({{beats-issue}}44294) [44296]({{beats-pull}}44296)

**Filebeat**

- Fix endpoint path typo in the Okta entity analytics provider. [44147]({{beats-pull}}44147)
- Fix a WebSocket panic scenario that occured after exhausting the maximum number of retries. [44342]({{beats-pull}}44342)

**Metricbeat**

- Add AWS OwningAccount support for cross-account monitoring. [40570]({{beats-issue}}40570) [40691]({{beats-pull}}40691)
- Use namespace for GetListMetrics calls in AWS when available. [41022]({{beats-pull}}41022)
- Limit index stats collection to cluster-level summaries. [36019]({{beats-issue}}36019) [42901]({{beats-pull}}42901)
- Omit `tier_preference`, `creation_date` and `version` fields in output documents when not pulled from source indices. [43637]({{beats-pull}}43637)
- Add support for `_nodes/stats` URIs compatible with legacy Elasticsearch versions. [44307]({{beats-pull}}44307)

## 9.0.1 [beats-9.0.1-release-notes]

### Features and enhancements [beats-9.0.1-features-enhancements]

* For all Beats: Publish `cloud.availability_zone` by `add_cloud_metadata` processor in Azure environments. [#42601]({{beats-issue}}42601) [#43618]({{beats-pull}}43618)
* Add pagination batch size support to Entity Analytics input's Okta provider in Filebeat. [#43655]({{beats-pull}}43655)
* Update CEL mito extensions version to v1.19.0 in Filebeat. [#44098]({{beats-pull}}44098)
* Upgrade node version to latest LTS v18.20.7 in Heartbeat. [#43511]({{beats-pull}}43511)
* Add `enable_batch_api` option in Azure monitor to allow metrics collection of multiple resources using Azure batch API in Metricbeat. [#41790]({{beats-pull}}41790)

### Fixes [beats-9.0.1-fixes]

Review the changes, fixes, and more in each version of {{product}}.

To check for security updates, go to [Security announcements for the Elastic Stack](https://discuss.elastic.co/c/announcements/security-announcements/31).

:::{admonition} Related release notes
{{agent}} integrates and manages {{beats}} for data collection. For changes to {{agent}}, refer to the [{{agent}} release notes](elastic-agent://release-notes/index.md).
:::

:::{include} /release-notes/_snippets/index.md
:::
