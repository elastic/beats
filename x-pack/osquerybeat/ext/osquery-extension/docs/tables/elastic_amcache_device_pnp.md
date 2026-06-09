% This file is generated! See ext/osquery-extension/cmd/gentables.

# elastic_amcache_device_pnp

Windows Amcache inventory device PnP entries (Root\InventoryDevicePnp)

## Platforms

- ❌ Linux
- ❌ macOS
- ✅ Windows

## Description

Device PnP inventory from Windows Amcache.
Queries Root\InventoryDevicePnp from the Amcache.hve registry hive.

## Schema

| Column | Type | Description |
|--------|------|-------------|
| `timestamp` | `BIGINT` | Last write time as Unix timestamp |
| `date_time` | `TEXT` | Last write time in RFC3339 |
| `model` | `TEXT` | Device model |
| `manufacturer` | `TEXT` | Manufacturer |
| `driver_name` | `TEXT` | Driver name |
| `parent_id` | `TEXT` | Parent device ID |
| `matching_id` | `TEXT` | Matching ID |
| `class` | `TEXT` | Device class |
| `class_guid` | `TEXT` | Class GUID |
| `description` | `TEXT` | Device description |
| `enumerator` | `TEXT` | Enumerator |
| `service` | `TEXT` | Service name |
| `install_state` | `TEXT` | Install state |
| `device_state` | `TEXT` | Device state |
| `inf` | `TEXT` | INF path |
| `driver_ver_date` | `TEXT` | Driver version date |
| `install_date` | `TEXT` | Install date |
| `first_install_date` | `TEXT` | First install date |
| `driver_package_strong_name` | `TEXT` | Driver package strong name |
| `driver_ver_version` | `TEXT` | Driver version |
| `container_id` | `TEXT` | Container ID |
| `problem_code` | `TEXT` | Problem code |
| `provider` | `TEXT` | Provider |
| `driver_id` | `TEXT` | Driver ID |
| `bus_reported_description` | `TEXT` | Bus-reported description |
| `hw_id` | `TEXT` | Hardware ID |
| `extended_infs` | `TEXT` | Extended INFs |
| `compid` | `TEXT` | Component ID |
| `stack_id` | `TEXT` | Stack ID |
| `upper_class_filters` | `TEXT` | Upper class filters |
| `lower_class_filters` | `TEXT` | Lower class filters |
| `upper_filters` | `TEXT` | Upper filters |
| `lower_filters` | `TEXT` | Lower filters |
| `device_interface_classes` | `TEXT` | Device interface classes |
| `location_paths` | `TEXT` | Location paths |

## Examples
### List all amcache device PnP entries

```sql
SELECT * FROM elastic_amcache_device_pnp;
```
### Devices by class

```sql
SELECT model, manufacturer, class, driver_name FROM elastic_amcache_device_pnp WHERE class != '';
```

## Notes
- Windows only. Requires Amcache.hve.

## Related Tables
- `elastic_amcache_driver_binary`
- `elastic_amcache_driver_package`
