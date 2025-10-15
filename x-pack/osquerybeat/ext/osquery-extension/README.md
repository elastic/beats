# Osquery Extension for Elastic

This osquery extension provides additional custom tables that enhance osquery's capabilities with Elastic-specific functionality. The extension is designed to work seamlessly with Osquerybeat and provides deep system insights across Linux, macOS, and Windows platforms.

## Overview

The extension adds several custom tables to osquery that provide:
- Browser history analysis across multiple browsers
- Host filesystem information reading from alternative root paths
- Process information with enhanced details
- File analysis with code signing and dependency information (macOS)
- User and group management information

## Supported Platforms

| Table | Linux | macOS | Windows |
|-------|-------|-------|---------|
| `browser_history` | ✅ | ✅ | ✅ |
| `host_users` | ✅ | ✅ | ❌ |
| `host_groups` | ✅ | ✅ | ❌ |
| `host_processes` | ✅ | ❌ | ❌ |
| `elastic_file_analysis` | ❌ | ✅ | ❌ |

---

## Tables

### 1. `browser_history`

Query browser history from multiple browsers with unified schema and advanced filtering capabilities.

#### Supported Browsers

- **Chromium-based**: Chrome, Edge, Brave, Chromium, Opera
- **Firefox**: Firefox
- **Safari**: Safari (macOS only)

#### Schema

| Column | Type | Description |
|--------|------|-------------|
| `timestamp` | BIGINT | Unix timestamp of the visit |
| `datetime` | TEXT | Human-readable datetime in RFC3339 format |
| `url` | TEXT | The visited URL |
| `title` | TEXT | Page title |
| `browser` | TEXT | Browser name (chrome, edge, firefox, safari, etc.) |
| `parser` | TEXT | Parser used (chromium, firefox, safari) |
| `user` | TEXT | System user who owns the browser profile |
| `profile_name` | TEXT | Browser profile name (e.g., "Default", "Profile 1") |
| `transition_type` | TEXT | How the user navigated to the page |
| `referring_url` | TEXT | The referring URL |
| `visit_id` | BIGINT | Unique visit identifier |
| `from_visit_id` | BIGINT | Visit ID of the referring page |
| `url_id` | BIGINT | Unique URL identifier |
| `visit_count` | BIGINT | Number of times URL was visited |
| `typed_count` | INTEGER | Number of times URL was typed |
| `visit_source` | TEXT | Source of the visit |
| `is_hidden` | INTEGER | Whether visit is hidden (0 or 1) |
| `history_path` | TEXT | Path to the history database file |
| `ch_visit_duration_ms` | BIGINT | *Chromium only:* Visit duration in milliseconds |
| `ff_session_id` | BIGINT | *Firefox only:* Session identifier |
| `ff_frecency` | BIGINT | *Firefox only:* Frecency score (frequency + recency) |
| `sf_domain_expansion` | TEXT | *Safari only:* Domain classification |
| `sf_load_successful` | INTEGER | *Safari only:* Whether page loaded successfully |
| `custom_data_dir` | TEXT | Custom data directory path (if specified) |

#### Example Queries

```sql
-- Find all visits to a specific domain in the last 7 days
SELECT datetime, url, title, browser, user 
FROM browser_history 
WHERE url LIKE '%github.com%' 
  AND timestamp > (strftime('%s', 'now') - 604800)
ORDER BY timestamp DESC;

-- Get most visited sites by user
SELECT user, url, COUNT(*) as visit_count
FROM browser_history
GROUP BY user, url
ORDER BY visit_count DESC
LIMIT 10;

-- Find downloads (common download URLs)
SELECT datetime, url, browser, user, profile_name
FROM browser_history
WHERE url LIKE '%.exe%' 
   OR url LIKE '%.dmg%' 
   OR url LIKE '%.zip%'
ORDER BY timestamp DESC;

-- Analyze browser usage by user
SELECT user, browser, COUNT(*) as total_visits
FROM browser_history
GROUP BY user, browser
ORDER BY total_visits DESC;

-- Get all typed URLs (directly typed by user)
SELECT datetime, url, title, browser, user
FROM browser_history
WHERE typed_count > 0
ORDER BY timestamp DESC;

-- Find Chrome visits with long duration
SELECT datetime, url, title, ch_visit_duration_ms / 1000 as duration_seconds
FROM browser_history
WHERE browser = 'chrome' 
  AND ch_visit_duration_ms > 60000
ORDER BY ch_visit_duration_ms DESC;
```

#### Timestamp Format

The `datetime` field uses RFC3339 format for filtering:
```sql
-- Filter by datetime string (automatically converted to timestamp)
SELECT * FROM browser_history 
WHERE datetime >= '2024-01-01T00:00:00Z';
```

---

### 2. `host_users`

Query user information from `/etc/passwd` (Linux/macOS). This table is useful for reading user information from alternative filesystem roots (e.g., container inspection, mounted drives).

#### Schema

| Column | Type | Description |
|--------|------|-------------|
| `uid` | BIGINT | User ID (unsigned) |
| `gid` | BIGINT | Primary group ID (unsigned) |
| `uid_signed` | BIGINT | User ID (signed) |
| `gid_signed` | BIGINT | Primary group ID (signed) |
| `username` | TEXT | Username/login name |
| `description` | TEXT | User description/full name (GECOS field) |
| `directory` | TEXT | Home directory path |
| `shell` | TEXT | Default shell |
| `uuid` | TEXT | User UUID (if available) |

#### Example Queries

```sql
-- List all users
SELECT * FROM host_users ORDER BY uid;

-- Find users with UID >= 1000 (typical human users)
SELECT username, uid, directory, shell 
FROM host_users 
WHERE uid >= 1000;

-- Find users with no login shell
SELECT username, uid, shell 
FROM host_users 
WHERE shell LIKE '%nologin' OR shell LIKE '%false';

-- Find users with bash shell
SELECT username, uid, directory 
FROM host_users 
WHERE shell LIKE '%bash';
```

---

### 3. `host_groups`

Query group information from `/etc/group` (Linux/macOS). This table reads group information from the configured filesystem root.

#### Schema

| Column | Type | Description |
|--------|------|-------------|
| `gid` | BIGINT | Group ID (unsigned) |
| `gid_signed` | BIGINT | Group ID (signed) |
| `groupname` | TEXT | Group name |

#### Example Queries

```sql
-- List all groups
SELECT * FROM host_groups ORDER BY gid;

-- Find system groups (GID < 1000)
SELECT groupname, gid 
FROM host_groups 
WHERE gid < 1000;

-- Find specific group
SELECT * FROM host_groups WHERE groupname = 'docker';
```

---

### 4. `host_processes`

Query detailed process information by reading directly from `/proc` filesystem (Linux only). This provides more control and can read from alternative filesystem roots.

#### Schema

| Column | Type | Description |
|--------|------|-------------|
| `pid` | BIGINT | Process ID |
| `name` | TEXT | Process name |
| `path` | TEXT | Path to process executable |
| `cmdline` | TEXT | Complete command line |
| `state` | TEXT | Process state (R, S, D, Z, T, etc.) |
| `cwd` | TEXT | Current working directory |
| `root` | TEXT | Root directory |
| `uid` | BIGINT | Real user ID |
| `gid` | BIGINT | Real group ID |
| `euid` | BIGINT | Effective user ID |
| `egid` | BIGINT | Effective group ID |
| `suid` | BIGINT | Saved user ID |
| `sgid` | BIGINT | Saved group ID |
| `on_disk` | INTEGER | Whether executable exists on disk (-1 = unknown) |
| `wired_size` | BIGINT | Wired memory size (always 0 on Linux) |
| `resident_size` | BIGINT | Resident set size (RSS) in bytes |
| `total_size` | BIGINT | Total virtual memory size in bytes |
| `user_time` | BIGINT | CPU time in user mode (milliseconds) |
| `system_time` | BIGINT | CPU time in kernel mode (milliseconds) |
| `disk_bytes_read` | BIGINT | Bytes read from disk |
| `disk_bytes_written` | BIGINT | Bytes written to disk |
| `start_time` | BIGINT | Process start time (Unix timestamp) |
| `parent` | BIGINT | Parent process ID |
| `pgroup` | BIGINT | Process group ID |
| `threads` | INTEGER | Number of threads |
| `nice` | INTEGER | Nice value (-20 to 19) |

#### Example Queries

```sql
-- Find all processes by user
SELECT pid, name, cmdline, user_time, system_time
FROM host_processes
WHERE uid = 1000;

-- Find high memory processes
SELECT pid, name, resident_size / 1024 / 1024 as rss_mb
FROM host_processes
WHERE resident_size > 100000000
ORDER BY resident_size DESC;

-- Find processes by name pattern
SELECT pid, name, cmdline, parent
FROM host_processes
WHERE name LIKE '%python%';

-- Find zombie processes
SELECT pid, name, parent, state
FROM host_processes
WHERE state = 'Z';

-- Find processes with most CPU time
SELECT pid, name, (user_time + system_time) / 1000 as cpu_seconds
FROM host_processes
ORDER BY (user_time + system_time) DESC
LIMIT 10;

-- Find recently started processes (last hour)
SELECT pid, name, cmdline, start_time
FROM host_processes
WHERE start_time > (strftime('%s', 'now') - 3600)
ORDER BY start_time DESC;

-- Find processes with high I/O
SELECT pid, name, disk_bytes_read, disk_bytes_written,
       (disk_bytes_read + disk_bytes_written) / 1024 / 1024 as total_io_mb
FROM host_processes
WHERE (disk_bytes_read + disk_bytes_written) > 10000000
ORDER BY total_io_mb DESC;
```

---

### 5. `elastic_file_analysis`

Perform deep file analysis on macOS using native system tools. This table provides comprehensive file metadata, code signing information, dependencies, symbols, and strings extraction.

**Platform:** macOS only

#### Schema

| Column | Type | Description |
|--------|------|-------------|
| `path` | TEXT | **Required constraint** - File path to analyze |
| `mode` | TEXT | File permissions (octal) |
| `uid` | BIGINT | File owner user ID |
| `gid` | BIGINT | File owner group ID |
| `size` | BIGINT | File size in bytes |
| `mtime` | BIGINT | Last modification time (Unix timestamp) |
| `file_type` | TEXT | Output from `file` command |
| `code_sign` | TEXT | Code signing information from `codesign -dvvv` |
| `dependencies` | TEXT | Library dependencies from `otool -L` |
| `symbols` | TEXT | Symbol table from `nm` |
| `strings` | TEXT | Printable strings from `strings -a` |

#### Important Notes

- **`path` constraint is required** - You must specify a file path to analyze
- Only works with regular files (not directories or special files)
- Uses native macOS command-line tools for analysis
- Output can be large for complex binaries

#### Example Queries

```sql
-- Analyze a specific binary
SELECT * FROM elastic_file_analysis 
WHERE path = '/usr/bin/python3';

-- Check code signing of application
SELECT path, code_sign 
FROM elastic_file_analysis 
WHERE path = '/Applications/Safari.app/Contents/MacOS/Safari';

-- Find dependencies of a library
SELECT path, dependencies 
FROM elastic_file_analysis 
WHERE path = '/usr/lib/libcurl.dylib';

-- Extract strings from binary
SELECT path, strings 
FROM elastic_file_analysis 
WHERE path = '/usr/local/bin/suspicious_binary';

-- Analyze file metadata and type
SELECT path, mode, uid, gid, size, file_type
FROM elastic_file_analysis 
WHERE path = '/tmp/downloaded_file';

-- Check symbols in an executable
SELECT path, symbols 
FROM elastic_file_analysis 
WHERE path = '/usr/bin/ls';
```

---

## Configuration

### Host Filesystem Root

The extension respects the `BEATS_HOSTFS` environment variable for reading from alternative filesystem roots. This is particularly useful when:
- Inspecting container filesystems from the host
- Analyzing mounted forensic images
- Reading from chrooted environments

```bash
# Set alternative root for host filesystem reads
export BEATS_HOSTFS=/hostfs

# Now host_users, host_groups, and host_processes will read from /hostfs
```

Without `BEATS_HOSTFS`, tables read from the normal root `/`.

### Browser History Custom Paths

The `browser_history` table automatically discovers browsers from standard user profile locations. To query history from custom locations (e.g., forensic analysis, backup inspection):

```sql
-- Query from custom Chrome profile
SELECT * FROM browser_history 
WHERE custom_data_dir = '/mnt/backup/Users/john/AppData/Local/Google';

-- Query all profiles in a custom location using glob
SELECT * FROM browser_history 
WHERE custom_data_dir GLOB '/forensics/users/*/Library/Application Support/Google';
```

---

## Performance Considerations

### Browser History

- The table reads SQLite databases from disk - performance depends on database size
- Use timestamp filters to limit results: `WHERE timestamp > X`
- Filtering by `browser`, `user`, or `profile_name` improves performance
- Custom data directory constraints are evaluated before database parsing
- For forensic analysis of large datasets, use specific time ranges

### Host Processes

- PID filtering is highly efficient (direct `/proc/[pid]` read)
- Without PID filter, all processes are enumerated
- Reading process details is I/O intensive for many processes
- Consider filtering by specific criteria to reduce result set

### File Analysis

- **Required path constraint** - the table only analyzes files you explicitly specify
- Analysis involves executing multiple command-line tools
- Large binaries will have large output (especially `symbols` and `strings`)
- Use specific queries rather than broad scans

---

## Security Considerations

### Permissions

- Browser history: Requires read access to browser profile directories
- Host users/groups: Requires read access to `/etc/passwd` and `/etc/group`
- Host processes: Requires read access to `/proc` filesystem
- File analysis: Requires read access to target files and execution of system commands

### File Analysis Security

The `elastic_file_analysis` table implements security measures:
- Path validation to ensure target is a regular file
- No directory traversal or symbolic link following
- Only analyzes explicitly specified files
- Fails safely with errors for invalid inputs

### Browser History Privacy

- Contains potentially sensitive browsing history
- Access should be restricted to authorized users
- Consider data retention policies when logging results
- Be aware of privacy regulations when collecting history

---

## Building and Installation

### Build the Extension

From the osquerybeat directory:

```bash
# Build for current platform
mage buildext

# The extension binary will be created at:
# Linux: ext/osquery-extension/build/linux/osquery-extension
# macOS: ext/osquery-extension/build/darwin/osquery-extension  
# Windows: ext/osquery-extension/build/windows/osquery-extension.ext
```

### Using with Osquery

The extension is automatically loaded by Osquerybeat. To use it manually with osquery:

```bash
# Start osquery with the extension
osqueryi --extension /path/to/osquery-extension [--allow-unsafe]

# Verify tables are loaded
osqueryi> .tables elastic
  => browser_history
  => elastic_file_analysis (macOS only)
  => host_groups
  => host_processes (Linux only)
  => host_users

# Query the tables
osqueryi> SELECT * FROM browser_history LIMIT 10;
```