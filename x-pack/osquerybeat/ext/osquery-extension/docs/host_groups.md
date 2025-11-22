# host_groups

Query group information from the host system's `/etc/group` file when running in a container environment.

## Platforms

- ✅ Linux
- ✅ macOS
- ❌ Windows

## Description

This table provides access to the host system's group information when osquery is running inside a container. It reads from the `/hostfs/etc/group` file (where the host filesystem is mounted), allowing containers to inspect the host system's user group configuration without needing to query the container's own group database.

This is particularly useful for:
- Container security auditing
- Host system inventory from within containers
- Compliance checks on host group configurations
- Detecting unauthorized group modifications on the host

## Schema

| Column | Type | Description |
|--------|------|-------------|
| `gid` | `BIGINT` | Unsigned int64 group ID |
| `gid_signed` | `BIGINT` | A signed int64 version of gid |
| `groupname` | `TEXT` | Canonical local group name |

## How It Works

When osquery runs in a container with the host filesystem mounted at `/hostfs`, this table reads directly from `/hostfs/etc/group` to retrieve the host system's group information. This is different from osquery's built-in `groups` table, which reads from the container's own `/etc/group` file.

### Mounting Host Filesystem

To use this table, mount the host's root filesystem into the container:

```bash
docker run -v /:/hostfs:ro osquery-container
```

## Examples

### Basic Queries

```sql
-- Get all host system groups
SELECT * FROM host_groups;

-- Find specific group by name
SELECT * FROM host_groups WHERE groupname = 'docker';

-- Find group by GID
SELECT * FROM host_groups WHERE gid = 0;

-- List all system groups (typically GID < 1000)
SELECT groupname, gid FROM host_groups 
WHERE gid < 1000 
ORDER BY gid;
```

### Security Auditing

```sql
-- Check for root group (GID 0)
SELECT * FROM host_groups WHERE gid = 0;

-- Find groups with negative GIDs (unusual configuration)
SELECT groupname, gid, gid_signed 
FROM host_groups 
WHERE gid_signed < 0;

-- List privileged groups (common system administration groups)
SELECT groupname, gid FROM host_groups 
WHERE groupname IN ('root', 'wheel', 'sudo', 'admin', 'docker')
ORDER BY groupname;

-- Find duplicate GIDs (potential security issue)
SELECT gid, COUNT(*) as count, GROUP_CONCAT(groupname) as groups
FROM host_groups
GROUP BY gid
HAVING count > 1;
```

### Compliance and Inventory

```sql
-- Count total groups on host
SELECT COUNT(*) as total_groups FROM host_groups;

-- Compare container groups vs host groups
SELECT 
    hg.groupname as host_group,
    hg.gid as host_gid,
    g.groupname as container_group,
    g.gid as container_gid
FROM host_groups hg
LEFT JOIN groups g ON hg.groupname = g.groupname;

-- Find groups that exist on host but not in container
SELECT groupname, gid FROM host_groups
WHERE groupname NOT IN (SELECT groupname FROM groups);

-- Find groups that exist in container but not on host
SELECT groupname, gid FROM groups
WHERE groupname NOT IN (SELECT groupname FROM host_groups);
```

## Comparison with Built-in Tables

| Feature | `host_groups` | `groups` |
|---------|---------------|----------|
| Data Source | Host's `/hostfs/etc/group` | Container's `/etc/group` |
| Use Case | Audit host system from container | Query container's own groups |
| Requires `/hostfs` mount | ✅ Yes | ❌ No |
| Available in | Osquery extension | Built-in osquery |

## Performance Considerations

- Fast table - reads a single file from disk
- File is cached by the OS
- Minimal overhead for repeated queries
- No external dependencies or network calls

## Security Considerations

- Requires read access to `/hostfs/etc/group`
- Should run with appropriate container permissions
- The `/hostfs` mount should be read-only (`:ro`)
- Contains no sensitive data (group passwords are stored in `/etc/gshadow`)
- Useful for security auditing and compliance

## Troubleshooting

### No Results Returned

1. **Host filesystem not mounted**: Ensure `/hostfs` mount exists
   ```bash
   docker run -v /:/hostfs:ro your-image
   ```

2. **Permission denied**: Container needs read access
   ```bash
   # Add read permissions or run with appropriate user
   --user $(id -u):$(id -g)
   ```

3. **File not found**: Verify the file exists
   ```bash
   docker exec container ls -la /hostfs/etc/group
   ```

### Negative GID Values

The `gid_signed` field provides a signed interpretation of the GID, which can handle systems that use signed integers for group IDs. On most systems, GIDs are unsigned, so `gid` and `gid_signed` will have the same value. However, some systems may use negative values internally, which is why both fields are provided.

## Implementation Details

### File Format

The `/etc/group` file follows this format:
```
groupname:password:gid:user_list
```

Example:
```
root:x:0:
docker:x:999:user1,user2
developers:x:1000:alice,bob
```

### Field Mapping

- `groupname`: First field (group name)
- `gid`: Third field (group ID), parsed as unsigned integer
- `gid_signed`: Third field (group ID), parsed as signed integer

### Platform Differences

**Linux:**
- Standard `/etc/group` format
- System groups typically have GID < 1000
- User groups typically have GID >= 1000

**macOS:**
- Also uses `/etc/group` but less commonly
- Many groups managed by OpenDirectory instead
- Local groups still in `/etc/group`
- System groups often have GID < 500

## Related Tables

- `groups` - Query container's own group database
- `host_users` - Query host system's user information
- `host_processes` - Query host system's running processes
