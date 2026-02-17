% This file is generated! See ext/osquery-extension/cmd/gentables.

# elastic_amcache_application_shortcut

Windows Amcache inventory application shortcut entries (Root\InventoryApplicationShortcut)

## Platforms

- ❌ Linux
- ❌ macOS
- ✅ Windows

## Description

Application shortcut inventory from Windows Amcache.
Queries Root\InventoryApplicationShortcut from the Amcache.hve registry hive.

## Schema

| Column | Type | Description |
|--------|------|-------------|
| `timestamp` | `BIGINT` | Last write time as Unix timestamp |
| `date_time` | `TEXT` | Last write time in RFC3339 |
| `shortcut_path` | `TEXT` | Shortcut path |
| `shortcut_target_path` | `TEXT` | Shortcut target path |
| `shortcut_aumid` | `TEXT` | Shortcut AUMID |
| `shortcut_program_id` | `TEXT` | Shortcut program ID |

## Examples
### List all amcache application shortcuts

```sql
SELECT * FROM elastic_amcache_application_shortcut;
```
### Shortcuts by program ID

```sql
SELECT shortcut_path, shortcut_target_path, shortcut_program_id FROM elastic_amcache_application_shortcut;
```

## Notes
- Windows only. Requires Amcache.hve.

## Related Tables
- `elastic_amcache_application`
