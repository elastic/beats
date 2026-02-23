% This file is generated! See ext/osquery-extension/cmd/gentables.

# elastic_amcache_driver_binary

Windows Amcache inventory driver binary entries (Root\InventoryDriverBinary)

## Platforms

- ❌ Linux
- ❌ macOS
- ✅ Windows

## Description

Driver binary inventory from Windows Amcache.
Queries Root\InventoryDriverBinary from the Amcache.hve registry hive.

## Schema

| Column | Type | Description |
|--------|------|-------------|
| `timestamp` | `BIGINT` | Last write time as Unix timestamp |
| `date_time` | `TEXT` | Last write time in RFC3339 |
| `driver_name` | `TEXT` | Driver name |
| `inf` | `TEXT` | INF path |
| `driver_version` | `TEXT` | Driver version |
| `product` | `TEXT` | Product name |
| `product_version` | `TEXT` | Product version |
| `wdf_version` | `TEXT` | WDF version |
| `driver_company` | `TEXT` | Driver company |
| `driver_package_strong_name` | `TEXT` | Driver package strong name |
| `service` | `TEXT` | Service name |
| `driver_in_box` | `TEXT` | In-box driver flag |
| `driver_signed` | `TEXT` | Driver signed flag |
| `driver_is_kernel_mode` | `TEXT` | Kernel mode driver flag |
| `driver_id` | `TEXT` | Driver ID |
| `driver_last_write_time` | `TEXT` | Driver last write time |
| `driver_type` | `BIGINT` | Driver type |
| `driver_time_stamp` | `BIGINT` | Driver timestamp |
| `driver_check_sum` | `BIGINT` | Driver checksum |
| `image_size` | `BIGINT` | Image size |

## Examples
### List all amcache driver binaries

```sql
SELECT * FROM elastic_amcache_driver_binary;
```
### Signed drivers

```sql
SELECT driver_name, driver_version, driver_signed, driver_company FROM elastic_amcache_driver_binary WHERE driver_signed = '1';
```

## Notes
- Windows only. Requires Amcache.hve.

## Related Tables
- `elastic_amcache_driver_package`
