## 9.1.9 [beats-9.1.9-breaking-changes]


**All**

::::{dropdown} Remove otel.component.id and otel.component.kind from beat receiver events.
% Describe the functionality that changed

For more information, check [#47729](https://github.com/elastic/beats/pull/47729)[#47600](https://github.com/elastic/beats/issues/47600).

**Impact**<br>If previously using the `otel.component.id` or `otel.component.kind` to track Beats receiver events, the fields are no longer used.

% **Action**<br>_Add a description of the what action to take_
::::
