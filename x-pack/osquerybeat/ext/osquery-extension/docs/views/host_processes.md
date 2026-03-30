% This file is generated! See ext/osquery-extension/cmd/gentables.

# host_processes

(Deprecated.) Backward-compatible view over elastic_host_processes; use elastic_host_processes instead.

## Platforms

- ✅ Linux
- ❌ macOS
- ❌ Windows

## Description

**Deprecated.** This view is deprecated in favor of the elastic_host_processes table.
It exists only for backward compatibility (SELECT * FROM elastic_host_processes).
Use elastic_host_processes directly in new use cases.

## Schema

| Column | Type | Description |
|--------|------|-------------|
| `pid` | `BIGINT` | Process (or thread) ID |
| `name` | `TEXT` | The process path or shorthand argv[0] |
| `path` | `TEXT` | Path to executed binary |
| `cmdline` | `TEXT` | Complete argv (command line arguments) |
| `state` | `TEXT` | Process state (R=running, S=sleeping, D=disk sleep, Z=zombie, T=stopped) |
| `cwd` | `TEXT` | Process current working directory |
| `root` | `TEXT` | Process virtual root directory |
| `uid` | `BIGINT` | Unsigned user ID (real UID) |
| `gid` | `BIGINT` | Unsigned group ID (real GID) |
| `euid` | `BIGINT` | Unsigned effective user ID |
| `egid` | `BIGINT` | Unsigned effective group ID |
| `suid` | `BIGINT` | Unsigned saved user ID |
| `sgid` | `BIGINT` | Unsigned saved group ID |
| `on_disk` | `INTEGER` | The process path exists; yes=1, no=0, unknown=-1 |
| `wired_size` | `BIGINT` | Bytes of unpageable memory (always 0 on Linux) |
| `resident_size` | `BIGINT` | Bytes of private memory used by process (RSS) |
| `total_size` | `BIGINT` | Total virtual memory size |
| `user_time` | `BIGINT` | CPU time in milliseconds spent in user space |
| `system_time` | `BIGINT` | CPU time in milliseconds spent in kernel space |
| `disk_bytes_read` | `BIGINT` | Bytes read from disk |
| `disk_bytes_written` | `BIGINT` | Bytes written to disk |
| `start_time` | `BIGINT` | Process start time in seconds since Epoch, or -1 if error |
| `parent` | `BIGINT` | Process parent's PID (PPID) |
| `pgroup` | `BIGINT` | Process group ID |
| `threads` | `INTEGER` | Number of threads used by process |
| `nice` | `INTEGER` | Process nice level (-20 to 20, default 0) |

## Required Tables

This view requires the following tables to be available:
- `elastic_host_processes`

## View Definition

```sql
CREATE VIEW host_processes AS
SELECT * FROM elastic_host_processes;
```

## Examples
### Query host processes (same as elastic_host_processes)

```sql
SELECT * FROM host_processes;
```

## Notes
- Deprecated in favor of elastic_host_processes; use the table directly for new queries.

## Related Tables
- `elastic_host_processes`
- `elastic_host_users`
