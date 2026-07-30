% This file is generated! See ext/osquery-extension/cmd/gentables.

# elastic_ntfs_partitions

Windows disk partition layout information from IOCTL_DISK_GET_DRIVE_LAYOUT_EX

## Platforms

- ❌ Linux
- ❌ macOS
- ✅ Windows

## Description

Returns all partitions detected on the system

## Schema

| Column | Type | Description |
|--------|------|-------------|
| `device` | `TEXT` | Device name (e.g. "\\.\PHYSICALDRIVE0") |
| `drive_letter` | `TEXT` | Drive letter assigned to the partition (e.g. "C:") |
| `id` | `TEXT` | Partition id |
| `number` | `BIGINT` | Partition number (e.g. 1) |
| `style` | `TEXT` | Partition style (e.g. MBR, GPT, RAW) |
| `type` | `TEXT` | Partition type (e.g. "System", "Basic", "Recovery") |
| `starting_offset` | `BIGINT` | Starting offset of the partition in bytes |
| `length` | `BIGINT` | Length of the partition in bytes |
| `attributes_mask` | `TEXT` | Raw partition attributes bitmask value, formatted as hexadecimal string (e.g. "0x0000000000000001") |
| `attributes` | `TEXT` | Human-readable partition attributes (e.g. "RequiredPartition,NoDriveLetter") |
| `name` | `TEXT` | Partition name (e.g. "Basic data partition") |

## Examples
### List all partitions

```sql
SELECT * FROM elastic_ntfs_partitions;
```

## Notes
- Windows only

## Related Tables
- `elastic_ntfs_volumes`
