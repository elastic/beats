# host_groups

Query group information from `/etc/group`. This table reads group information from the configured filesystem root, useful for inspecting alternative roots like containers or mounted drives.

## Platforms

- ✅ Linux
- ✅ macOS
- ❌ Windows

## Schema

| Column | Type | Description |
|--------|------|-------------|
| `gid` | `BIGINT` | Group ID (unsigned) |
| `gid_signed` | `BIGINT` | Group ID (signed representation) |
| `groupname` | `TEXT` | Group name |

## Configuration

The table respects the `BEATS_HOSTFS` environment variable:

```bash
# Read from alternative filesystem root
export BEATS_HOSTFS=/hostfs

# Without BEATS_HOSTFS, reads from /etc/group
# With BEATS_HOSTFS=/hostfs, reads from /hostfs/etc/group
```

## Examples

### Basic Queries

```sql
-- List all groups
SELECT * FROM host_groups;

-- Find specific group
SELECT * FROM host_groups WHERE groupname = 'sudo';

-- Find groups with specific GID range
SELECT groupname, gid
FROM host_groups
WHERE gid >= 1000 AND gid < 65534;

-- List system groups
SELECT groupname, gid
FROM host_groups
WHERE gid < 1000
ORDER BY gid;
```

### Group Analysis

```sql
-- Count groups
SELECT COUNT(*) as total_groups FROM host_groups;

-- Find privileged groups (common administrative groups)
SELECT groupname, gid
FROM host_groups
WHERE groupname IN ('root', 'wheel', 'sudo', 'admin', 'docker')
ORDER BY groupname;

-- Check for duplicate GIDs
SELECT gid, COUNT(*) as count, GROUP_CONCAT(groupname) as groups
FROM host_groups
GROUP BY gid
HAVING count > 1;
```

### Cross-Reference with Users

```sql
-- Join with host_users to find users and their primary groups
SELECT 
    u.username,
    u.uid,
    g.groupname,
    g.gid
FROM host_users u
JOIN host_groups g ON u.gid = g.gid
ORDER BY u.uid;

-- Find users in administrative groups (primary group)
SELECT 
    u.username,
    g.groupname,
    u.shell
FROM host_users u
JOIN host_groups g ON u.gid = g.gid
WHERE g.groupname IN ('root', 'wheel', 'sudo', 'admin')
ORDER BY u.username;
```

## Use Cases

### Container Inspection

Inspect group configuration in container filesystems:

```bash
# Mount container filesystem and query groups
export BEATS_HOSTFS=/var/lib/docker/overlay2/[container-id]/merged
osqueryi --extension osquery-extension
```

```sql
SELECT * FROM host_groups;
```

### Forensics

Analyze group accounts from mounted forensic images:

```bash
export BEATS_HOSTFS=/mnt/forensics/linux-image
```

### Compliance Auditing

Verify group configurations match security policies:

```sql
-- Check that administrative groups exist
SELECT groupname FROM host_groups 
WHERE groupname IN ('wheel', 'sudo');

-- Verify system groups are in expected GID ranges
SELECT groupname, gid FROM host_groups 
WHERE gid < 1000;
```

## Performance Considerations

- Reading from `/etc/group` is generally fast
- File is typically small (< 500KB)
- No complex parsing required
- Minimal system impact

## Security Considerations

- Requires read access to `/etc/group` (or `$BEATS_HOSTFS/etc/group`)
- `/etc/group` is world-readable on most systems
- Group membership enumeration can be security-sensitive
- Note: This table only shows group definitions, not group membership
  - Use osquery's `user_groups` table to see which users are members of which groups

## Differences from osquery's `groups` Table

The built-in osquery `groups` table uses system APIs (getgrent) which always reads from the host system. The `host_groups` table:

- Reads directly from `/etc/group` file
- Respects `BEATS_HOSTFS` for alternative filesystem roots
- Useful for container inspection and forensic analysis
- Does not query system group databases (LDAP, AD, etc.)

## Troubleshooting

### Empty Results

- Check that `/etc/group` exists at the configured root
- Verify read permissions on the file
- Ensure `BEATS_HOSTFS` is set correctly if using alternative root

### Missing Groups

- This table only reads from `/etc/group`
- Groups from LDAP, Active Directory, or other sources won't appear
- Use osquery's built-in `groups` table for system-wide group enumeration

### Group Membership Not Shown

- This table shows group definitions only
- To see which users are members of groups, use:
  - osquery's `user_groups` table for full membership
  - Parse the members field from `/etc/group` directly (not exposed in this table)
