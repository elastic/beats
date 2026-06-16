% This file is generated! See ext/osquery-extension/cmd/gentables.

# host_users

(Deprecated.) Backward-compatible view over elastic_host_users; use elastic_host_users instead.

## Platforms

- ✅ Linux
- ✅ macOS
- ❌ Windows

## Description

**Deprecated.** This view is deprecated in favor of the elastic_host_users table.
It exists only for backward compatibility (SELECT * FROM elastic_host_users).
Use elastic_host_users directly in new use cases.

## Schema

| Column | Type | Description |
|--------|------|-------------|
| `uid` | `BIGINT` | User ID (unsigned) |
| `gid` | `BIGINT` | Default group ID (unsigned) |
| `uid_signed` | `BIGINT` | User ID as int64 signed (for Apple systems) |
| `gid_signed` | `BIGINT` | Default group ID as int64 signed (for Apple systems) |
| `username` | `TEXT` | Username / login name |
| `description` | `TEXT` | Optional user description / full name (GECOS field) |
| `directory` | `TEXT` | User's home directory path |
| `shell` | `TEXT` | User's configured default shell |
| `uuid` | `TEXT` | User's UUID (Apple) or SID (Windows); typically empty on Linux |

## Required Tables

This view requires the following tables to be available:
- `elastic_host_users`

## View Definition

```sql
CREATE VIEW host_users AS
SELECT * FROM elastic_host_users;
```

## Examples
### Query host users (same as elastic_host_users)

```sql
SELECT * FROM host_users;
```

## Notes
- Deprecated in favor of elastic_host_users; use the table directly for new queries.

## Related Tables
- `elastic_host_users`
- `elastic_host_groups`
