## 9.2.2 [beats-9.2.2-breaking-changes]


**All**

::::{dropdown} Remove otel.component.id and otel.component.kind from beat receiver events.
In a previous version, we added the `otel.component.id` and `otel.component.kind` fields, but after running the relevant benchmarks we found that there was not enough value to justify the cost of sending the extra data in every event.

For more information, check [#47729](https://github.com/elastic/beats/pull/47729)[#47600](https://github.com/elastic/beats/issues/47600).

**Impact**<br>These fields were not publicly documented and were not part of an API. However, if you do rely on them, they will no longer be available.

% **Action**<br>_Add a description of the what action to take_
::::
