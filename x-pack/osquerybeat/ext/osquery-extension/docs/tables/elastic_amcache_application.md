% This file is generated! See ext/osquery-extension/cmd/gentables.

# elastic_amcache_application

Windows Amcache inventory application entries (Root\InventoryApplication)

## Platforms

- ❌ Linux
- ❌ macOS
- ✅ Windows

## Description

Application inventory from Windows Amcache (Application Compatibility Cache).
Queries Root\InventoryApplication from the Amcache.hve registry hive.

## Schema

| Column | Type | Description |
|--------|------|-------------|
| `timestamp` | `BIGINT` | Last write time as Unix timestamp |
| `date_time` | `TEXT` | Last write time in RFC3339 |
| `program_id` | `TEXT` | Program identifier |
| `program_instance_id` | `TEXT` | Program instance identifier |
| `name` | `TEXT` | Application name |
| `version` | `TEXT` | Version string |
| `publisher` | `TEXT` | Publisher name |
| `language` | `BIGINT` | Language ID |
| `install_date` | `TEXT` | Install date |
| `source` | `TEXT` | Source |
| `root_dir_path` | `TEXT` | Root directory path |
| `hidden_arp` | `BIGINT` | Hidden ARP flag |
| `uninstall_string` | `TEXT` | Uninstall command string |
| `registry_key_path` | `TEXT` | Registry key path |
| `store_app_type` | `TEXT` | Store app type |
| `inbox_modern_app` | `TEXT` | Inbox modern app flag |
| `manifest_path` | `TEXT` | Manifest path |
| `package_full_name` | `TEXT` | Package full name |
| `msi_package_code` | `TEXT` | MSI package code |
| `msi_product_code` | `TEXT` | MSI product code |
| `msi_install_date` | `TEXT` | MSI install date |
| `bundle_manifest_path` | `TEXT` | Bundle manifest path |
| `user_sid` | `TEXT` | User SID |
| `sha1` | `TEXT` | SHA1 hash (last 40 chars of program_id) |

## Examples
### List all amcache applications

```sql
SELECT * FROM elastic_amcache_application;
```
### Find application by name

```sql
SELECT program_id, name, publisher, version FROM elastic_amcache_application WHERE name LIKE '%Chrome%';
```

## Notes
- Windows only. Requires Amcache.hve (e.g. from C:\Windows\appcompat\Programs\Amcache.hve).

## Related Tables
- `elastic_amcache_application_file`
- `elastic_amcache_application_shortcut`
- `elastic_amcache_applications_view`
