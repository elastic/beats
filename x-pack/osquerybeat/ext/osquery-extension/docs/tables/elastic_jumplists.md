% This file is generated! See ext/osquery-extension/cmd/gentables.

# elastic_jumplists

Windows Jump Lists containing recent and pinned items with LNK and destination metadata

## Platforms

- ❌ Linux
- ❌ macOS
- ✅ Windows

## Description

Parse Windows Jump Lists for all non-special local user profiles and return one
row per embedded LNK entry with application, user, destination, and shortcut metadata.
Both custom destinations (pinned items) and automatic destinations (recent items)
are supported.

## Schema

| Column | Type | Description |
|--------|------|-------------|
| `application_id` | `TEXT` | Jump List application identifier from the source filename |
| `application_name` | `TEXT` | Resolved application name for the application_id when known |
| `username` | `TEXT` | Username that owns the Jump List file |
| `sid` | `TEXT` | Windows SID for the user profile |
| `jumplist_type` | `TEXT` | Jump List type (custom or automatic) |
| `source_file_path` | `TEXT` | Path to the .customDestinations-ms or .automaticDestinations-ms source file |
| `hostname` | `TEXT` | Hostname parsed from the automatic DestList entry metadata |
| `entry_number` | `INTEGER` | DestList entry number for automatic Jump Lists |
| `last_modified_time` | `TEXT` | DestList entry last modified time |
| `is_pinned` | `INTEGER` | Whether the destination entry is pinned (1) or not pinned (0) |
| `interaction_count` | `INTEGER` | Interaction count from the DestList entry |
| `dest_entry_path` | `TEXT` | Raw destination path stored in the DestList entry |
| `dest_entry_path_resolved` | `TEXT` | Resolved destination path with known-folder GUIDs translated when possible |
| `mac_address` | `TEXT` | MAC address derived from DestList metadata |
| `creation_time` | `TEXT` | Destination creation time derived from DestList metadata |
| `local_path` | `TEXT` | Local target path from embedded LNK metadata |
| `file_size` | `INTEGER` | Target file size from embedded LNK metadata |
| `hot_key` | `TEXT` | Hotkey configured for the shortcut target, if any |
| `icon_index` | `INTEGER` | Icon index referenced by the shortcut |
| `show_window` | `TEXT` | Window display mode from the shortcut metadata |
| `icon_location` | `TEXT` | Icon location path from the shortcut metadata |
| `command_line_arguments` | `TEXT` | Command-line arguments stored in the shortcut |
| `target_modification_time` | `TEXT` | Shortcut target modification time from LNK metadata |
| `target_last_accessed_time` | `TEXT` | Shortcut target last accessed time from LNK metadata |
| `target_creation_time` | `TEXT` | Shortcut target creation time from LNK metadata |
| `volume_serial_number` | `TEXT` | Volume serial number associated with the shortcut target |
| `volume_type` | `TEXT` | Drive type associated with the shortcut target volume |
| `volume_label` | `TEXT` | Volume label associated with the shortcut target |
| `volume_label_offset` | `INTEGER` | Volume label offset in LinkInfo volume data |
| `name` | `TEXT` | Name string from LNK metadata |

## Examples
### List all Jump List entries

```sql
SELECT * FROM elastic_jumplists;
```
### Recent entries for a specific application ID

```sql
SELECT username, application_name, local_path, source_file_path
FROM elastic_jumplists
WHERE application_id = '590aee7bdd69b59b';
```
### Find pinned entries

```sql
SELECT username, application_name, local_path, is_pinned
FROM elastic_jumplists
WHERE is_pinned = 1;
```

## Notes
- Windows only. Reads per-user Recent\AutomaticDestinations and Recent\CustomDestinations directories.
- One row is emitted per parsed LNK entry; application_name may be empty when application_id is unknown.
- Automatic destination metadata fields (for example entry_number, dest_entry_path) may be empty for custom destination rows.

## Related Tables
- `users`
