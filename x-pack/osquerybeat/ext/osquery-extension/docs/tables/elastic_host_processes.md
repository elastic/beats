% This file is generated! See ext/osquery-extension/cmd/gentables.

# elastic_host_processes

Host system running processes from /proc (e.g. when running in a container with hostfs mounted)

## Platforms

- ✅ Linux
- ❌ macOS
- ❌ Windows

## Description

Query running process information from the host system when running in a container.
Reads from the host's /proc via hostfs (default /hostfs); set ELASTIC_OSQUERY_HOSTFS to override.
Use for container security monitoring, host process auditing, and forensics.

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

## Examples
### Get all host processes

```sql
SELECT pid, name, cmdline FROM elastic_host_processes;
```
### Find process by PID

```sql
SELECT * FROM elastic_host_processes WHERE pid = 1;
```
### Find processes running as root

```sql
SELECT pid, name, uid, cmdline FROM elastic_host_processes WHERE uid = 0;
```

## Notes
- Linux only. Requires host /proc (or root) mounted (e.g. -v /:/hostfs:ro or -v /proc:/hostfs/proc:ro).
- Use ELASTIC_OSQUERY_HOSTFS to override the hostfs root (default /hostfs).

## Related Tables
- `elastic_host_users`
- `processes`
