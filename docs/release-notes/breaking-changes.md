---
navigation_title: "Breaking changes"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/libbeat/current/breaking-changes.html
---

# Beats breaking changes [beats-breaking-changes]
Breaking changes can impact your Elastic applications, potentially disrupting normal operations. Before you upgrade, carefully review the Beats breaking changes and take the necessary steps to mitigate any issues. To learn how to upgrade, check out [Upgrade](docs-content://deploy-manage/upgrade.md).

% ## Next version

% Description and impact of the breaking change.
% For more information, check [PR #](PR link).

## 9.0.1 [beats-9.0.1-breaking-changes]

_No breaking changes_

## 9.0.0 [beats-900-breaking-changes]

% Description and impact of the breaking change.
% For more information, check [PR #](PR link).

::::{dropdown} Set default Kafka version to 2.1.0 in Kafka output and Filebeat.
For more information, check [#41662]({{beats-pull}}41662).
::::

::::{dropdown} Replaced default Ubuntu-based images with UBI-minimal-based ones.
For more information, check  [#42150]({{beats-pull}}42150).
::::

::::{dropdown} Removed support for a single "-" to precede multi-letter command line arguments.  Use "--" instead.
For more information, check [#42117]({{beats-issue}}42117) [#42209]({{beats-pull}}42209).
::::

::::{dropdown} Filebeat fails to start if there is any input with a duplicated ID. It logs the duplicated IDs and the offending inputs configurations.
For more information, check [#41731]({{beats-pull}}41731).
::::

::::{dropdown} Filestream inputs with duplicated IDs will fail to start. An error is logged showing the ID and the full input configuration.
For more information, check [#41938]({{beats-issue}}41938) [#41954]({{beats-pull}}41954).
::::

::::{dropdown} Filestream inputs can define "allow_deprecated_id_duplication: true" to run keep the previous behaviour of running inputs with duplicated IDs.
For more information, check [#41938]({{beats-issue}}41938) [#41954]({{beats-pull}}41954).
::::

::::{dropdown} Filestream inputs now starts ingesting files only if they are 1024 bytes or larger because the default file identity has changed from native to fingerprint.

At startup Filebeat automatically updates the state from known, active files (i.e: files that are still present on the disk and have not changed path since Filebeat was stopped) to use the new file identity. If Filebeat cannot migrate the state to the new file identity, the file will be re-ingested. To preserve the behaviour from 8.x, set `file_identity.native: ~` and `prospector.scanner.fingerprint.enabled: false`.

Refer to the file identity documentation for more details. You can also check [#40197]({{beats-issue}}40197) [#41762]({{beats-pull}}41762).
::::

::::{dropdown} Filebeat fails to start when its configuration contains usage of the deprecated log or container inputs. However, they can still be used when "allow_deprecated_use: true" is set in their configuration.
For more information, check [#42295]({{beats-pull}}42295).
::::

::::{dropdown} Upgrade osquery version to 5.13.1.
For more information, check [#40849]({{beats-pull}}40849).
::::

::::{dropdown} Use base-16 for reporting serial_number value in TLS fields in line with the ECS recommendation.
For more information, check [#41542]({{beats-pull}}41542).
::::

::::{dropdown} Default to use raw API and delete older XML implementation.
For more information, check [#42275]({{beats-pull}}42275).
::::

::::{dropdown} The Beats logger and file output rotate files when necessary. The beat now forces a file rotation when unexpectedly writing to a file through a symbolic link.
::::

::::{dropdown} Remove kibana.settings metricset since the API was removed in 8.0 in Metricbeat.
For more information, check [#30592]({{beats-issue}}30592). [#42937]({{beats-pull}}42937).
::::

::::{dropdown} Removed support for the Enterprise Search module in Metricbeat.
For more information, check [#42915]({{beats-pull}}42915).
::::

::::{dropdown} Allow faccessat(2) in seccomp.
For more information, check [#43322]({{beats-pull}}43322).
::::