# host_processes

Query detailed process information by reading directly from the `/proc` filesystem. This table provides more control than the built-in osquery `processes` table and can read from alternative filesystem roots.

## Platforms

- ✅ Linux
- ❌ macOS
- ❌ Windows

## Schema

| Column | Type | Description |
|--------|------|-------------|
| `pid` | `BIGINT` | Process ID |
| `name` | `TEXT` | Process name (from /proc/[pid]/comm) |
| `path` | `TEXT` | Path to the executable |
| `cmdline` | `TEXT` | Complete command line |
| `state` | `TEXT` | Process state (R, S, D, Z, T, etc.) |
| `cwd` | `TEXT` | Current working directory |
| `root` | `TEXT` | Root directory (for chrooted processes) |
| `uid` | `BIGINT` | Real user ID |
| `gid` | `BIGINT` | Real group ID |
| `euid` | `BIGINT` | Effective user ID |
| `egid` | `BIGINT` | Effective group ID |
| `suid` | `BIGINT` | Saved user ID |
| `sgid` | `BIGINT` | Saved group ID |
| `on_disk` | `INTEGER` | Whether executable exists on disk (-1 if unknown) |
| `wired_size` | `BIGINT` | Bytes of wired memory used by process |
| `resident_size` | `BIGINT` | Bytes of resident memory (RSS) |
| `total_size` | `BIGINT` | Total virtual memory size |
| `user_time` | `BIGINT` | CPU time in user mode (milliseconds) |
| `system_time` | `BIGINT` | CPU time in system mode (milliseconds) |
| `disk_bytes_read` | `BIGINT` | Bytes read from disk |
| `disk_bytes_written` | `BIGINT` | Bytes written to disk |
| `start_time` | `BIGINT` | Process start time (Unix timestamp) |
| `parent` | `BIGINT` | Parent process ID |
| `pgroup` | `BIGINT` | Process group ID |
| `threads` | `INTEGER` | Number of threads |
| `nice` | `INTEGER` | Nice value (process priority) |

## Configuration

The table respects the `BEATS_HOSTFS` environment variable:

```bash
# Read from alternative filesystem root
export BEATS_HOSTFS=/hostfs

# Without BEATS_HOSTFS, reads from /proc
# With BEATS_HOSTFS=/hostfs, reads from /hostfs/proc
```

## Examples

### Basic Queries

```sql
-- List all processes
SELECT pid, name, state, uid FROM host_processes;

-- Find specific process
SELECT * FROM host_processes WHERE name = 'sshd';

-- Find processes by user
SELECT pid, name, cmdline 
FROM host_processes 
WHERE uid = 1000;

-- Running processes only
SELECT pid, name, cmdline
FROM host_processes
WHERE state = 'R';
```

### Process Tree Analysis

```sql
-- Find child processes of a specific parent
SELECT pid, name, cmdline
FROM host_processes
WHERE ppid = 1234;

-- Count processes per parent
SELECT ppid, COUNT(*) as child_count
FROM host_processes
GROUP BY ppid
ORDER BY child_count DESC
LIMIT 10;

-- Find process hierarchy (requires recursive query)
WITH RECURSIVE process_tree AS (
  SELECT pid, name, ppid, cmdline, 0 as level
  FROM host_processes
  WHERE pid = 1
  UNION ALL
  SELECT p.pid, p.name, p.ppid, p.cmdline, pt.level + 1
  FROM host_processes p
  JOIN process_tree pt ON p.ppid = pt.pid
  WHERE pt.level < 5
)
SELECT level, pid, ppid, name FROM process_tree;
```

### Security Queries

```sql
-- Find processes running as root
SELECT pid, name, cmdline, euid
FROM host_processes
WHERE euid = 0;

-- Find setuid processes (euid != uid)
SELECT pid, name, uid, euid, cmdline
FROM host_processes
WHERE euid != uid;

-- Find processes in different root (chrooted)
SELECT pid, name, root, cmdline
FROM host_processes
WHERE root != '/';

-- Find zombie processes
SELECT pid, name, ppid, state
FROM host_processes
WHERE state = 'Z';
```

### Resource Monitoring

```sql
-- Find processes with high nice values (low priority)
SELECT pid, name, nice, cmdline
FROM host_processes
WHERE nice > 10
ORDER BY nice DESC;

-- Find processes with negative nice (high priority)
SELECT pid, name, nice, cmdline
FROM host_processes
WHERE nice < 0
ORDER BY nice ASC;

-- Count processes by state
SELECT state, COUNT(*) as count
FROM host_processes
GROUP BY state;
```

### Container Inspection

```sql
-- List processes from container (with BEATS_HOSTFS=/var/lib/docker/.../merged)
SELECT pid, name, cmdline, uid
FROM host_processes
ORDER BY pid;

-- Compare host vs container process namespaces
-- (Query from both host and container roots separately)
```

## Use Cases

### Container Monitoring

Monitor processes running inside containers:

```bash
export BEATS_HOSTFS=/var/lib/docker/overlay2/[container-id]/merged
```

### Forensics

Analyze process state from mounted forensic images or live systems.

### Alternative to Built-in Processes Table

When you need:
- Direct `/proc` filesystem access
- Container process inspection
- More control over data source

## Performance Considerations

- Reading from `/proc` is fast but can be CPU-intensive for many processes
- Use filters (WHERE clauses) to limit results
- Avoid querying all processes frequently
- Consider caching results for monitoring use cases

## Security Considerations

- Requires read access to `/proc` filesystem
- Process command lines may contain sensitive information (passwords, tokens)
- Some `/proc` entries may require elevated privileges
- Be careful when exposing process information in logs

## Differences from osquery's `processes` Table

The built-in osquery `processes` table uses system APIs and always reads from the host. The `host_processes` table:

- Reads directly from `/proc` filesystem
- Respects `BEATS_HOSTFS` for alternative filesystem roots
- Useful for container inspection and forensic analysis
- May have less information than the built-in table (no CPU/memory stats)
- Does not work on macOS/Windows (Linux-specific `/proc` filesystem)

## Linux Process States

| State | Description |
|-------|-------------|
| `R` | Running or runnable |
| `S` | Interruptible sleep (waiting for an event) |
| `D` | Uninterruptible sleep (usually I/O) |
| `Z` | Zombie (terminated but not reaped by parent) |
| `T` | Stopped (on a signal) |
| `t` | Tracing stop |
| `W` | Paging (not valid since Linux 2.6) |
| `X` | Dead (should never be seen) |
| `x` | Dead (should never be seen) |
| `K` | Wakekill |
| `P` | Parked |
| `I` | Idle |

## Troubleshooting

### Empty Results

- Check that `/proc` exists and is mounted
- Verify read permissions
- Ensure `BEATS_HOSTFS` is set correctly if using alternative root

### Permission Denied Errors

- Some `/proc/[pid]` entries require elevated privileges
- Run with appropriate permissions or as root
- Consider running osquery as a privileged user

### Incomplete Information

- Some processes may restrict access to their `/proc` entries
- Kernel threads may have limited information
- Container processes may have different namespaces
