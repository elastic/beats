% This file is generated! See ext/osquery-extension/cmd/gentables.

# sample_recent_files

Windows Recent Files tracking user file access history

## Platforms

- ❌ Linux
- ❌ macOS
- ✅ Windows

## Description

Windows maintains LNK files in the Recent folder to track recently accessed
files for each user. This table parses these LNK files to provide forensic
information about file access history.

This table shares the UserProfile and LnkMetadata types with sample_jumplists,
demonstrating how shared types enable consistent column definitions across
related tables.

## Schema

| Column | Type | Description |
|--------|------|-------------|
| `username` | `TEXT` | The username |
| `domain` | `TEXT` | The domain or computer name |
| `sid` | `TEXT` | The Windows Security Identifier (SID) |
| `local_path` | `TEXT` | Target local path |
| `file_size` | `INTEGER` | Target file size in bytes |
| `hot_key` | `TEXT` | Assigned hotkey combination |
| `command_line_arguments` | `TEXT` | Command line arguments for the target |
| `icon_location` | `TEXT` | Path to the icon file |
| `volume_serial_number` | `TEXT` | Volume serial number (format XXXX-XXXX) |
| `volume_label` | `TEXT` | Volume label |
| `created_time` | `BIGINT` | File creation timestamp |
| `modified_time` | `BIGINT` | File modification timestamp |
| `accessed_time` | `BIGINT` | File last access timestamp |
| `lnk_file_path` | `TEXT` | Path to the LNK file in the Recent folder |
| `target_exists` | `INTEGER` | Whether the target file still exists (1=yes, 0=no) |

## Examples
### Find recent files accessed by a user

```sql
SELECT username, local_path, modified_time
FROM sample_recent_files
WHERE username = 'john.doe'
ORDER BY modified_time DESC;
```
### Find files accessed from removable drives

```sql
SELECT username, local_path, volume_serial_number, volume_label
FROM sample_recent_files
WHERE volume_label LIKE '%USB%';
```

## Notes
- Shares UserProfile and LnkMetadata types with sample_jumplists table
- Uses FileTimestamps for standard timestamp fields
- LNK files persist even after the target file is deleted
