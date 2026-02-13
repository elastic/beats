% This file is generated! See ext/osquery-extension/cmd/gentables.

# elastic_host_users

Host system user account information from /etc/passwd (e.g. when running in a container with hostfs mounted)

## Platforms

- ✅ Linux
- ✅ macOS
- ❌ Windows

## Description

Query user account information from the host system's /etc/passwd file when running in a container.
Reads from the path given by hostfs (default /hostfs); set ELASTIC_OSQUERY_HOSTFS to override.
Use for container security auditing, host user inventory, and compliance checks.

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

## Examples
### Get all host users

```sql
SELECT * FROM elastic_host_users;
```
### Find user by username

```sql
SELECT * FROM elastic_host_users WHERE username = 'root';
```
### Find user by UID

```sql
SELECT * FROM elastic_host_users WHERE uid = 1000;
```
### List system users (UID < 1000)

```sql
SELECT username, uid, shell, directory FROM elastic_host_users WHERE uid < 1000 ORDER BY uid;
```

## Notes
- Linux and macOS. Requires host filesystem mounted (e.g. -v /:/hostfs:ro).
- Use ELASTIC_OSQUERY_HOSTFS to override the hostfs root (default /hostfs).

## Related Tables
- `elastic_host_groups`
- `users`
