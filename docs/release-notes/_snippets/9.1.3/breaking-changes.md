## 9.1.3 [beats-9.1.3-breaking-changes]

**Metricbeat**

::::{dropdown} API used by index summary metricset changed.
Changed index summary metricset to use `_nodes/stats` API instead of `_stats` API to avoid data gaps.

% **Impact**<br>Add a description of the impact.

% **Action**<br>Add a description of what action to take.

For more information, check [#45049]({{beats-pull}}45049).
::::
