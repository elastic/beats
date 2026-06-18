% This file is generated! See ext/osquery-extension/cmd/gentables.

# elastic_ntfs_file

Returns information about files on NTFS volumes, parsed from the $MFT file on Windows systems.

## Platforms

- ❌ Linux
- ❌ macOS
- ✅ Windows

## Description

Returns information about files on NTFS volumes, parsed from the $MFT file on Windows systems.
 The $MFT file contains metadata about all files and directories on an NTFS volume, including their names, sizes, timestamps, and attributes. This table parses the $MFT file to extract this information and present it in a structured format.
 Note that this table is Windows-only, as the $MFT file is specific to the NTFS file system used by Windows.

## Schema

| Column | Type | Description |
|--------|------|-------------|
| `drive` | `TEXT` | Volume drive letter (e.g. C) |
| `device` | `TEXT` | Physical device path (e.g. \\\\.\PhysicalDrive0) |
| `partition` | `INTEGER` | Partition number of the volume (e.g. 1) |
| `inode` | `BIGINT` | MFT record number (inode) |
| `sequence_number` | `INTEGER` | MFT entry sequence number |
| `parent_inode` | `BIGINT` | MFT record number of the parent directory |
| `path` | `TEXT` | Full file path (e.g. C:\Windows\system32\ntoskrnl.exe) |
| `directory` | `TEXT` | Full path of the parent directory |
| `filename` | `TEXT` | File name component from the $FILE_NAME attribute |
| `type` | `TEXT` | Entry type ("file" or "directory") |
| `hard_link_count` | `INTEGER` | Number of hard links to this MFT entry |
| `active` | `INTEGER` | 1 if the MFT entry is allocated (active), 0 if not |
| `size` | `BIGINT` | Logical file size in bytes (from the default $DATA attribute) |
| `allocated_size` | `BIGINT` | Allocated size in bytes (from the $FILE_NAME attribute) |
| `flags` | `INTEGER` | File attribute flags from $STANDARD_INFORMATION |
| `ads` | `INTEGER` | 1 if the file has one or more Alternate Data Streams, 0 otherwise |
| `object_id` | `TEXT` | Object identifier GUID from the $OBJECT_ID attribute |
| `security_id` | `INTEGER` | Security descriptor identifier from $STANDARD_INFORMATION |
| `owner_id` | `INTEGER` | Owner identifier from $STANDARD_INFORMATION |
| `btime` | `BIGINT` | File creation time (Unix epoch) from $STANDARD_INFORMATION |
| `mtime` | `BIGINT` | File last-modified time (Unix epoch) from $STANDARD_INFORMATION |
| `ctime` | `BIGINT` | MFT entry last-modified time (Unix epoch) from $STANDARD_INFORMATION |
| `atime` | `BIGINT` | File last-accessed time (Unix epoch) from $STANDARD_INFORMATION |
| `fn_btime` | `BIGINT` | File creation time (Unix epoch) from $FILE_NAME |
| `fn_mtime` | `BIGINT` | File last-modified time (Unix epoch) from $FILE_NAME |
| `fn_ctime` | `BIGINT` | MFT entry last-modified time (Unix epoch) from $FILE_NAME |
| `fn_atime` | `BIGINT` | File last-accessed time (Unix epoch) from $FILE_NAME |

## Examples
### Path lookup

```sql
SELECT * FROM elastic_ntfs_file WHERE path = 'C:\Windows\System32\ntoskrnl.exe';
```
### Inode lookup (drive constraint required)

```sql
SELECT * FROM elastic_ntfs_file WHERE inode = 1972769 AND drive = 'C';
```
### Directory + filename glob

```sql
SELECT * FROM elastic_ntfs_file
WHERE directory = 'C:\Windows\System32' AND filename GLOB '*.exe';
```

## Notes
- Windows only
- Glob patterns are not recursive. A query like `WHERE directory = 'C:\Windows' AND filename GLOB '*.exe'` will not find `C:\Windows\System32\notepad.exe`
- Integer columns with value 0 are returned as empty string by the osquery wire protocol, not "0". Use numeric comparison (e.g. WHERE ads = 0, not ads = '0').

## Related Tables
- `elastic_ntfs_partitions`
- `elastic_ntfs_volumes`
