% This file is generated! See ext/osquery-extension/cmd/gentables.

# elastic_amcache_application_file

Windows Amcache inventory application file entries (Root\InventoryApplicationFile)

## Platforms

- ❌ Linux
- ❌ macOS
- ✅ Windows

## Description

Application file inventory from Windows Amcache.
Queries Root\InventoryApplicationFile from the Amcache.hve registry hive.

## Schema

| Column | Type | Description |
|--------|------|-------------|
| `timestamp` | `BIGINT` | Last write time as Unix timestamp |
| `date_time` | `TEXT` | Last write time in RFC3339 |
| `program_id` | `TEXT` | Program identifier |
| `file_id` | `TEXT` | File identifier |
| `lower_case_long_path` | `TEXT` | Lowercase long path |
| `name` | `TEXT` | File name |
| `original_file_name` | `TEXT` | Original file name |
| `publisher` | `TEXT` | Publisher |
| `version` | `TEXT` | Version |
| `bin_file_version` | `TEXT` | Binary file version |
| `binary_type` | `TEXT` | Binary type |
| `product_name` | `TEXT` | Product name |
| `product_version` | `TEXT` | Product version |
| `link_date` | `TEXT` | Link date |
| `bin_product_version` | `TEXT` | Binary product version |
| `size` | `BIGINT` | File size |
| `language` | `BIGINT` | Language ID |
| `usn` | `BIGINT` | Update sequence number |
| `appx_package_full_name` | `TEXT` | AppX package full name |
| `is_os_component` | `TEXT` | Is OS component flag |
| `appx_package_relative_id` | `TEXT` | AppX package relative ID |
| `sha1` | `TEXT` | SHA1 hash |

## Examples
### List all amcache application files

```sql
SELECT * FROM elastic_amcache_application_file;
```
### Find files by product name

```sql
SELECT program_id, name, product_name, lower_case_long_path FROM elastic_amcache_application_file WHERE product_name != '';
```

## Notes
- Windows only. Requires Amcache.hve.

## Related Tables
- `elastic_amcache_application`
- `elastic_amcache_applications_view`
