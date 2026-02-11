% This file is generated! See ext/osquery-extension/cmd/gentables.

# elastic_amcache_driver_package

Windows Amcache inventory driver package entries (Root\InventoryDriverPackage)

## Platforms

- ❌ Linux
- ❌ macOS
- ✅ Windows

## Description

Driver package inventory from Windows Amcache.
Queries Root\InventoryDriverPackage from the Amcache.hve registry hive.

## Schema

| Column | Type | Description |
|--------|------|-------------|
| `timestamp` | `BIGINT` | Last write time as Unix timestamp |
| `date_time` | `TEXT` | Last write time in RFC3339 |
| `class_guid` | `TEXT` | Class GUID |
| `class` | `TEXT` | Driver class |
| `directory` | `TEXT` | Directory path |
| `date` | `TEXT` | Date |
| `version` | `TEXT` | Version |
| `provider` | `TEXT` | Provider |
| `submission_id` | `TEXT` | Submission ID |
| `driver_in_box` | `TEXT` | In-box driver flag |
| `inf` | `TEXT` | INF path |
| `flight_ids` | `TEXT` | Flight IDs |
| `recovery_ids` | `TEXT` | Recovery IDs |
| `is_active` | `TEXT` | Is active flag |
| `hwids` | `TEXT` | Hardware IDs |
| `sysfile` | `TEXT` | SYS file path |

## Examples
### List all amcache driver packages

```sql
SELECT * FROM elastic_amcache_driver_package;
```
### Packages by provider

```sql
SELECT class_guid, version, provider, directory FROM elastic_amcache_driver_package WHERE provider != '';
```

## Notes
- Windows only. Requires Amcache.hve.

## Related Tables
- `elastic_amcache_driver_binary`
- `elastic_amcache_device_pnp`
