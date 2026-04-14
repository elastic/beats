% This file is generated! See ext/osquery-extension/cmd/gentables.

# elastic_ntfs_volumes

Windows mounted volume information including device type, drive letter, and filesystem

## Platforms

- ❌ Linux
- ❌ macOS
- ✅ Windows

## Description

Returns all volumes detected on the system

## Schema

| Column | Type | Description |
|--------|------|-------------|
| `device` | `TEXT` | Volume device path (e.g. \\.\C:) |
| `device_type` | `TEXT` | Volume device type (e.g. "Fixed", "Removable", "CD-ROM") |
| `drive_letter` | `TEXT` | Volume drive letter (e.g. C:) |
| `volume_label` | `TEXT` | Volume label (e.g. "OS") |
| `file_system_name` | `TEXT` | Volume file system name (e.g. "NTFS") |

## Examples
### List all volumes

```sql
SELECT * FROM elastic_ntfs_volumes;
```

## Notes
- Windows only

## Related Tables
- `elastic_ntfs_partitions`
