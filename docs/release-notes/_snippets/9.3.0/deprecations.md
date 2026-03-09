## 9.3.0 [beats-9.3.0-deprecations]


**Filebeat**

::::{dropdown} The Azure Event Hub input now defaults to processor v2. Processor v1 is deprecated.
Starting in 9.3.0, the `azure-eventhub` input defaults to processor v2, a drop-in replacement for v1 built on the modern Azure SDK for Go. Existing configurations work without changes.

For more information, refer to [#47292](https://github.com/elastic/beats/pull/47292).

**Impact**<br>

- Processor v2 is fully compatible with v1 configurations — no config changes are needed for existing setups.
- Checkpoint data is migrated automatically on startup (`migrate_checkpoint` defaults to `true`), so no events are reprocessed.
- `storage_account_key` still works (v2 auto-constructs a connection string from it), but users should plan to switch to `storage_account_connection_string`.
- For sovereign clouds, `resource_manager_endpoint` (v1) should be replaced with `authority_host` (v2).

**Action**<br>

- To continue using v1 temporarily, set `processor_version: "v1"` in your configuration. Processor v1 is planned for removal in 9.4.0.
- Refer to the [Migrating from processor v1 to v2](/reference/beats/filebeat/filebeat-input-azure-eventhub#_migrating_from_processor_v1_to_v2) guide for full details.

::::

