% This file is generated! See ext/osquery-extension/cmd/gentables.

# host_groups

(Deprecated.) Backward-compatible view over elastic_host_groups; use elastic_host_groups instead.

## Platforms

- ✅ Linux
- ✅ macOS
- ❌ Windows

## Description

**Deprecated.** This view is deprecated in favor of the elastic_host_groups table.
It exists only for backward compatibility (SELECT * FROM elastic_host_groups).
Use elastic_host_groups directly in new use cases.

## Schema

| Column | Type | Description |
|--------|------|-------------|
| `gid` | `BIGINT` | Unsigned int64 group ID |
| `gid_signed` | `BIGINT` | Signed int64 version of gid |
| `groupname` | `TEXT` | Canonical local group name |

## Required Tables

This view requires the following tables to be available:
- `elastic_host_groups`

## View Definition

```sql
CREATE VIEW host_groups AS
SELECT * FROM elastic_host_groups;
```

## Examples
### Query host groups (same as elastic_host_groups)

```sql
SELECT * FROM host_groups;
```

## Notes
- Deprecated in favor of elastic_host_groups; use the table directly for new queries.

## Related Tables
- `elastic_host_groups`
- `host_users`
