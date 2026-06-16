% This file is generated! See ext/osquery-extension/cmd/gentables.

# elastic_host_groups

Host system group information from /etc/group (e.g. when running in a container with hostfs mounted)

## Platforms

- ✅ Linux
- ✅ macOS
- ❌ Windows

## Description

Query group information from the host system's /etc/group file when running in a container.
Reads from the path given by hostfs (default /hostfs); set ELASTIC_OSQUERY_HOSTFS to override.
Use for container security auditing, host inventory, and compliance checks.

## Schema

| Column | Type | Description |
|--------|------|-------------|
| `gid` | `BIGINT` | Unsigned int64 group ID |
| `gid_signed` | `BIGINT` | Signed int64 version of gid |
| `groupname` | `TEXT` | Canonical local group name |

## Examples
### Get all host groups

```sql
SELECT * FROM elastic_host_groups;
```
### Find group by name

```sql
SELECT * FROM elastic_host_groups WHERE groupname = 'docker';
```
### Find group by GID

```sql
SELECT * FROM elastic_host_groups WHERE gid = 0;
```
### List system groups (GID < 1000)

```sql
SELECT groupname, gid FROM elastic_host_groups WHERE gid < 1000 ORDER BY gid;
```

## Notes
- Linux and macOS. Requires host filesystem mounted (e.g. -v /:/hostfs:ro).
- Use ELASTIC_OSQUERY_HOSTFS to override the hostfs root (default /hostfs).

## Related Tables
- `host_users`
- `groups`
