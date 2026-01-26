## 9.1.1 [beats-9.1.1-breaking-changes]

**All Beats**
::::{dropdown} Update user agent used by Beats HTTP clients.

The default user agent was updated to distinguish between all beat modes:

* **Standalone** indicates that the beat is not running under agent.
* **Unmanaged** indicates that the beat is running under agent but not managed by Fleet.
* **Managed** indicates that the beat is running under agent and managed by Fleet.

Users relying on specific user agents could be impacted.

For more information, check [#45251]({{beats-pull}}45251).
::::

