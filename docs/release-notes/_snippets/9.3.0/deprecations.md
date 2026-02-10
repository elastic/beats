## 9.3.0 [beats-9.3.0-deprecations]


**Filebeat**

::::{dropdown} Change Azure Event Hub input processor version default from v1 to v2.
This PR is to change Filebeat azure eventhub input to use processor V2 as default instead of V1. Since we&#39;ve decided to retire processor v1, we plan to set processor v2 as the default option in the next release, and completely remove v1 in the following release.

For more information, check [#47292](https://github.com/elastic/beats/pull/47292).

**Impact**<br>Users of the Azure Event Hub input who were relying on processor v1 as the default behavior will now automatically use processor v2, which may have different behavior or performance characteristics.

**Action**<br>Users who want to continue using processor v1 should explicitly set `processor_version: v1` in their configuration.
However, v1 will be completely removed in a future release, so users should plan to migrate to v2.
Users who were not explicitly specifying a processor version will now use v2 by default and should test their configuration accordingly.

::::

