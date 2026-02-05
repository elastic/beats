# sample_jumplists

Windows Jump Lists containing recently accessed files and pinned items

## Platforms

- ❌ Linux
- ❌ macOS
- ✅ Windows

## Description

Windows Jump Lists are a feature that provides quick access to recently
accessed files and common tasks for applications. This table parses both
custom destinations (pinned items) and automatic destinations (recent items).

Jump lists are stored per-user in the Recent folder and contain embedded
LNK (shortcut) file data with rich forensic information.

## Schema

| Column | Type | Description |
|--------|------|-------------|
| `ApplicationID` | `EMBEDDED` |  |
| `UserProfile` | `EMBEDDED` |  |
| `LnkMetadata` | `EMBEDDED` |  |
| `JumplistMeta` | `EMBEDDED` |  |
| `entry_index` | `INTEGER` | Index of the entry within the jumplist |

## Examples
### Find all jumplist entries for a specific application

```sql
SELECT application_name, local_path, username, jumplist_type
FROM sample_jumplists
WHERE application_id = '590aee7bdd69b59b';
```
### List recent file accesses by user

```sql
SELECT username, application_name, local_path, source_file_path
FROM sample_jumplists
ORDER BY username, application_name;
```

## Notes
- Uses shared types from shared_types.yaml for ApplicationID, UserProfile, and LnkMetadata
- Custom destinations contain user-pinned items
- Automatic destinations contain recently accessed files
