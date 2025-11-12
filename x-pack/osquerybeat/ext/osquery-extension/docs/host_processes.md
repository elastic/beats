# host_processes

Query running process information from the host system when running in a container environment.

## Platforms

- ✅ Linux
- ❌ macOS
- ❌ Windows

## Description

This table provides access to the host system's running processes when osquery is running inside a container. It reads from the `/hostfs/proc` filesystem (where the host's `/proc` is mounted), allowing containers to inspect all processes running on the host system without being limited to the container's process namespace.

This is particularly useful for:
- Container security monitoring
- Host system process auditing from within containers
- Detecting malicious processes on the host
- System performance monitoring
- Compliance and inventory management
- Incident response and forensics

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
| `on_disk` | `INTEGER` | The process path exists: yes=1, no=0, unknown=-1 |
| `wired_size` | `BIGINT` | Bytes of unpageable memory used by process (always 0 on Linux) |
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

## How It Works

When osquery runs in a container with the host's `/proc` filesystem mounted at `/hostfs/proc`, this table reads directly from the host's process information to enumerate all running processes. This bypasses the container's PID namespace isolation.

### Mounting Host Filesystem

To use this table, mount the host's `/proc` filesystem into the container:

```bash
docker run -v /proc:/hostfs/proc:ro osquery-container
```

Or mount the entire root filesystem:

```bash
docker run -v /:/hostfs:ro osquery-container
```

## Examples

### Basic Queries

```sql
-- Get all host processes
SELECT pid, name, cmdline FROM host_processes;

-- Find specific process by name
SELECT * FROM host_processes WHERE name = 'systemd';

-- Find process by PID
SELECT * FROM host_processes WHERE pid = 1;

-- Get process tree (parent-child relationships)
SELECT 
    p.pid,
    p.name,
    p.parent,
    parent_p.name as parent_name
FROM host_processes p
LEFT JOIN host_processes parent_p ON p.parent = parent_p.pid
ORDER BY p.parent, p.pid;
```

### Security Monitoring

```sql
-- Find processes running as root
SELECT pid, name, path, cmdline 
FROM host_processes 
WHERE uid = 0
ORDER BY start_time DESC;

-- Find processes with elevated privileges (setuid/setgid)
SELECT pid, name, uid, euid, gid, egid
FROM host_processes 
WHERE uid != euid OR gid != egid;

-- Find processes with suspicious names or paths
SELECT pid, name, path, cmdline
FROM host_processes 
WHERE name LIKE '%tmp%' 
   OR name LIKE '%.sh%'
   OR path LIKE '/tmp/%'
   OR path LIKE '/dev/shm/%';

-- Find processes without on-disk binaries (possible fileless malware)
SELECT pid, name, path, cmdline
FROM host_processes 
WHERE on_disk = 0;

-- Find zombie processes
SELECT pid, name, parent, state
FROM host_processes 
WHERE state = 'Z';

-- Find processes in uninterruptible sleep (often indicates I/O issues)
SELECT pid, name, cmdline, state
FROM host_processes 
WHERE state = 'D';
```

### Performance Monitoring

```sql
-- Top CPU consuming processes
SELECT 
    pid, 
    name, 
    (user_time + system_time) as total_cpu_ms,
    user_time,
    system_time
FROM host_processes 
ORDER BY total_cpu_ms DESC 
LIMIT 10;

-- Top memory consuming processes
SELECT 
    pid, 
    name, 
    resident_size / 1024 / 1024 as rss_mb,
    total_size / 1024 / 1024 as vsize_mb
FROM host_processes 
ORDER BY resident_size DESC 
LIMIT 10;

-- Top disk I/O processes
SELECT 
    pid, 
    name,
    disk_bytes_read / 1024 / 1024 as read_mb,
    disk_bytes_written / 1024 / 1024 as written_mb,
    (disk_bytes_read + disk_bytes_written) / 1024 / 1024 as total_io_mb
FROM host_processes 
ORDER BY total_io_mb DESC 
LIMIT 10;

-- Processes with most threads
SELECT pid, name, threads, cmdline
FROM host_processes 
ORDER BY threads DESC 
LIMIT 10;

-- Long-running processes
SELECT 
    pid, 
    name, 
    start_time,
    (strftime('%s', 'now') - start_time) / 86400 as days_running
FROM host_processes 
WHERE start_time > 0
ORDER BY start_time ASC 
LIMIT 10;
```

### Process Analysis

```sql
-- Count processes by user
SELECT uid, COUNT(*) as process_count
FROM host_processes 
GROUP BY uid 
ORDER BY process_count DESC;

-- Count processes by state
SELECT state, COUNT(*) as count
FROM host_processes 
GROUP BY state;

-- Find all children of a specific process
SELECT pid, name, cmdline 
FROM host_processes 
WHERE parent = 1234;

-- Find process hierarchy for a specific PID
WITH RECURSIVE proc_tree AS (
    SELECT pid, name, parent, 0 as level
    FROM host_processes 
    WHERE pid = 1234
    UNION ALL
    SELECT p.pid, p.name, p.parent, pt.level + 1
    FROM host_processes p
    JOIN proc_tree pt ON p.parent = pt.pid
)
SELECT * FROM proc_tree;

-- Compare process counts: container vs host
SELECT 
    (SELECT COUNT(*) FROM processes) as container_processes,
    (SELECT COUNT(*) FROM host_processes) as host_processes;
```

### Docker/Container Detection

```sql
-- Find Docker daemon and containerd processes
SELECT pid, name, cmdline 
FROM host_processes 
WHERE name IN ('dockerd', 'containerd', 'docker-proxy', 'containerd-shim');

-- Find container runtime processes
SELECT pid, name, cmdline, parent
FROM host_processes 
WHERE name LIKE '%containerd%' 
   OR name LIKE '%docker%'
   OR name LIKE '%runc%'
   OR name LIKE '%cri-o%';
```

## Comparison with Built-in Tables

| Feature | `host_processes` | `processes` |
|---------|------------------|-------------|
| Data Source | Host's `/hostfs/proc` | Container's `/proc` |
| Process Visibility | All host processes | Only container processes |
| Use Case | Monitor host from container | Query container's processes |
| Requires `/hostfs` mount | ✅ Yes | ❌ No |
| PID Namespace | Host PID namespace | Container PID namespace |
| Available in | Osquery extension | Built-in osquery |

## Performance Considerations

- Reading `/proc` for all processes can be resource-intensive
- Use PID constraints when possible: `WHERE pid = ?`
- Limit results when doing exploratory queries
- Process information is read on-demand from `/proc`
- Large systems may have thousands of processes
- Consider caching or sampling for continuous monitoring

## Security Considerations

- Requires read access to `/hostfs/proc`
- Should run with appropriate container permissions
- The `/hostfs` mount should be read-only (`:ro`)
- Exposes all host process information to container
- Can reveal sensitive command-line arguments and environment
- Useful for security monitoring and incident response
- May expose privileged process details

## Troubleshooting

### No Results Returned

1. **Host /proc not mounted**: Ensure `/hostfs/proc` mount exists
   ```bash
   docker run -v /proc:/hostfs/proc:ro your-image
   ```

2. **Permission denied**: Container needs read access to `/proc`
   ```bash
   # May need additional capabilities
   --cap-add SYS_PTRACE
   ```

3. **Verify mount**:
   ```bash
   docker exec container ls -la /hostfs/proc
   ```

### Partial Information

Some fields may be unavailable for certain processes:
- `on_disk = -1`: Unable to determine if binary exists (permission denied)
- `start_time = -1`: Unable to read process start time
- Empty `cmdline`: Process may be a kernel thread
- Empty `path`: Process binary may be deleted or inaccessible

### Understanding Process States

Linux process states (`state` field):
- `R` - Running or runnable (on run queue)
- `S` - Interruptible sleep (waiting for an event)
- `D` - Uninterruptible sleep (usually I/O)
- `Z` - Zombie (terminated but not reaped by parent)
- `T` - Stopped (on a signal or being traced)
- `t` - Tracing stop (ptrace)
- `X` - Dead (should never be visible)
- `I` - Idle (kernel thread)

### UID/GID Fields Explained

- `uid` / `gid`: Real user/group ID (who owns the process)
- `euid` / `egid`: Effective user/group ID (used for permission checks)
- `suid` / `sgid`: Saved user/group ID (for privilege dropping)

These can differ when processes use setuid/setgid bits or when privileges are dropped.

## Implementation Details

### Data Source

Reads from `/proc/[pid]/` for each process:
- `/proc/[pid]/stat` - Process statistics
- `/proc/[pid]/status` - Process status information
- `/proc/[pid]/cmdline` - Command line arguments
- `/proc/[pid]/exe` - Symbolic link to executable
- `/proc/[pid]/cwd` - Symbolic link to current directory
- `/proc/[pid]/root` - Symbolic link to root directory
- `/proc/[pid]/io` - I/O statistics

### Time Calculations

- CPU times are converted from kernel clock ticks to milliseconds
- Clock tick rate (CLK_TCK) is 100 Hz on x86
- Start time is calculated from system boot time + process start ticks

### Memory Fields

- `resident_size`: RSS (Resident Set Size) - physical memory used
- `total_size`: Virtual memory size (may be much larger than RSS)
- `wired_size`: Always 0 on Linux (not applicable)

### Limitations

- **Platform**: Linux only (relies on `/proc` filesystem)
- **PID filtering**: Most efficient with specific PID constraints
- **Permissions**: Some process details require elevated privileges
- **Kernel threads**: May have limited information available
- **Short-lived processes**: May miss processes that start and stop quickly

## Related Tables

- `processes` - Query container's own processes
- `host_users` - Map UIDs to usernames on the host
- `host_groups` - Map GIDs to group names on the host
- `process_open_files` (built-in) - Open files per process
- `process_open_sockets` (built-in) - Network connections per process
- `process_envvars` (built-in) - Environment variables per process
