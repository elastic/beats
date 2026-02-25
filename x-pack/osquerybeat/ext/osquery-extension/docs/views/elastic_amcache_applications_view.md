% This file is generated! See ext/osquery-extension/cmd/gentables.

# elastic_amcache_applications_view

Combined view of amcache applications and application files (apps with optional file rows, plus file-only rows)

## Platforms

- ❌ Linux
- ❌ macOS
- ✅ Windows

## Description

Combined view of Windows Amcache application and application file inventory.
Part 1: all application rows with optional matching file rows (LEFT JOIN on program_id).
Part 2: all file rows that have no matching application (file-only rows with NULL app columns).
Use for listing every app and/or file record from Amcache with a single query.

## Schema

| Column | Type | Description |
|--------|------|-------------|
| `timestamp` | `BIGINT` | Last write time as Unix timestamp |
| `date_time` | `TEXT` | Last write time in RFC3339 |
| `program_id` | `TEXT` | Program identifier |
| `file_id` | `TEXT` | File identifier |
| `lower_case_long_path` | `TEXT` | Lowercase long path |
| `name` | `TEXT` | Application or file name |
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
| `file_sha1` | `TEXT` | File SHA1 hash |
| `program_instance_id` | `TEXT` | Program instance identifier |
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
| `app_sha1` | `TEXT` | Application SHA1 hash |

## Required Tables

This view requires the following tables to be available:
- `elastic_amcache_application`
- `elastic_amcache_application_file`

## View Definition

```sql
CREATE VIEW elastic_amcache_applications_view AS
SELECT
  COALESCE(file.timestamp, app.timestamp) AS timestamp,
  COALESCE(file.date_time, app.date_time) AS date_time,
  app.program_id,
  file.file_id,
  file.lower_case_long_path,
  COALESCE(file.name, app.name) AS name,
  file.original_file_name,
  COALESCE(file.publisher, app.publisher) AS publisher,
  COALESCE(file.version, app.version) AS version,
  file.bin_file_version,
  file.binary_type,
  file.product_name,
  file.product_version,
  file.link_date,
  file.bin_product_version,
  file.size,
  COALESCE(file.language, app.language) AS language,
  file.usn,
  file.appx_package_full_name,
  file.is_os_component,
  file.appx_package_relative_id,
  file.sha1 AS file_sha1,
  app.program_instance_id,
  app.install_date,
  app.source,
  app.root_dir_path,
  app.hidden_arp,
  app.uninstall_string,
  app.registry_key_path,
  app.store_app_type,
  app.inbox_modern_app,
  app.manifest_path,
  app.package_full_name,
  app.msi_package_code,
  app.msi_product_code,
  app.msi_install_date,
  app.bundle_manifest_path,
  app.user_sid,
  app.sha1 AS app_sha1
FROM elastic_amcache_application AS app
LEFT JOIN elastic_amcache_application_file AS file ON app.program_id = file.program_id
UNION ALL
SELECT
  file.timestamp,
  file.date_time,
  file.program_id,
  file.file_id,
  file.lower_case_long_path,
  file.name,
  file.original_file_name,
  file.publisher,
  file.version,
  file.bin_file_version,
  file.binary_type,
  file.product_name,
  file.product_version,
  file.link_date,
  file.bin_product_version,
  file.size,
  file.language,
  file.usn,
  file.appx_package_full_name,
  file.is_os_component,
  file.appx_package_relative_id,
  file.sha1 AS file_sha1,
  NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL AS app_sha1
FROM elastic_amcache_application_file AS file
LEFT JOIN elastic_amcache_application AS app ON file.program_id = app.program_id
WHERE app.program_id IS NULL;
```

## Examples
### Query the combined applications view

```sql
SELECT * FROM elastic_amcache_applications_view;
```
### Find entries by name

```sql
SELECT program_id, name, publisher, version FROM elastic_amcache_applications_view WHERE name LIKE '%Microsoft%';
```

## Notes
- Windows only. Depends on elastic_amcache_application and elastic_amcache_application_file.

## Related Tables
- `elastic_amcache_application`
- `elastic_amcache_application_file`
