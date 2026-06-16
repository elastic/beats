% This file is generated! See ext/osquery-extension/cmd/gentables.

# elastic_file_analysis

Comprehensive security analysis of executable files on macOS (file type, code signing, dependencies, symbols, strings)

## Platforms

- ❌ Linux
- ✅ macOS
- ❌ Windows

## Description

Perform comprehensive security analysis of executable files on macOS. This table combines
multiple macOS system tools to extract metadata, code signing information, library
dependencies, symbols, and embedded strings from binary files. Query with a path
constraint (e.g. WHERE path = '/usr/bin/ssh'). Useful for malware analysis, code
signing verification, security auditing, binary forensics, and supply chain assessment.

## Schema

| Column | Type | Description |
|--------|------|-------------|
| `path` | `TEXT` | Absolute path to the file being analyzed |
| `mode` | `TEXT` | File permissions (e.g., 755) |
| `uid` | `BIGINT` | File owner user ID |
| `gid` | `BIGINT` | File owner group ID |
| `size` | `BIGINT` | File size in bytes |
| `mtime` | `BIGINT` | Last modification time (Unix timestamp) |
| `file_type` | `TEXT` | File type and architecture from the file command |
| `code_sign` | `TEXT` | Code signing information from codesign -dvvv |
| `dependencies` | `TEXT` | Linked libraries from otool -L |
| `symbols` | `TEXT` | Exported symbols from nm |
| `strings` | `TEXT` | Printable strings from binary (>= 4 characters) |

## Examples
### Analyze a specific executable

```sql
SELECT * FROM elastic_file_analysis
WHERE path = '/Applications/Safari.app/Contents/MacOS/Safari';
```
### Analyze executables in a directory

```sql
SELECT path, file_type, size
FROM elastic_file_analysis
WHERE path LIKE '/usr/local/bin/%';
```
### Get metadata and code signing

```sql
SELECT path, file_type, code_sign FROM elastic_file_analysis WHERE path = '/usr/bin/sudo';
```
### List library dependencies

```sql
SELECT path, dependencies FROM elastic_file_analysis WHERE path = '/usr/bin/ssh';
```
### Extract strings from binary

```sql
SELECT path, strings FROM elastic_file_analysis WHERE path = '/usr/bin/curl';
```

## Notes
- macOS only. Requires path constraint; uses file, codesign, otool, nm, and strings.
- Heavy operation: spawns multiple processes per row; use specific paths, avoid wildcards on large trees.
- Input paths are validated; only regular files are accepted.

## Related Tables
- `file`
- `hash`
