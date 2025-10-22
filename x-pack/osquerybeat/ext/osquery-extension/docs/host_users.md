# host_users

Query user information from `/etc/passwd`. This table reads user information from the configured filesystem root, making it useful for inspecting alternative roots like containers or mounted drives.

## Platforms

- ✅ Linux
- ✅ macOS
- ❌ Windows

## Schema

| Column | Type | Description |
|--------|------|-------------|
| `uid` | `BIGINT` | User ID (unsigned) |
| `gid` | `BIGINT` | Primary group ID (unsigned) |
| `uid_signed` | `BIGINT` | User ID (signed representation) |
| `gid_signed` | `BIGINT` | Primary group ID (signed representation) |
| `username` | `TEXT` | Username |
| `description` | `TEXT` | User description/GECOS field |
| `directory` | `TEXT` | Home directory path |
| `shell` | `TEXT` | Login shell |
| `uuid` | `TEXT` | User UUID (macOS) |

## Configuration

The table respects the `BEATS_HOSTFS` environment variable:

```bash
# Read from alternative filesystem root
export BEATS_HOSTFS=/hostfs

# Without BEATS_HOSTFS, reads from /etc/passwd
# With BEATS_HOSTFS=/hostfs, reads from /hostfs/etc/passwd
```

## Examples

### Basic Queries

```sql
-- List all users
SELECT * FROM host_users;

-- Find specific user
SELECT * FROM host_users WHERE username = 'root';

-- Find users with specific UID range
SELECT username, uid, directory 
FROM host_users 
WHERE uid >= 1000 AND uid < 65534;

-- List users with login shells
SELECT username, shell 
FROM host_users 
WHERE shell NOT LIKE '%nologin' AND shell NOT LIKE '%false';
```

### System Analysis

```sql
-- Count users by shell
SELECT shell, COUNT(*) as user_count
FROM host_users
GROUP BY shell
ORDER BY user_count DESC;

-- Find users without home directories
SELECT username, directory
FROM host_users
WHERE directory = '/nonexistent' OR directory = '';

-- List service accounts (system users)
SELECT username, uid, shell
FROM host_users
WHERE uid < 1000
ORDER BY uid;
```

### Security Queries

```sql
-- Find users with bash shells (potential login accounts)
SELECT username, uid, directory, shell
FROM host_users
WHERE shell LIKE '%bash%';

-- Find root-equivalent users
SELECT username, uid, gid
FROM host_users
WHERE uid = 0;

-- Check for disabled accounts
SELECT username, shell
FROM host_users
WHERE shell IN ('/usr/sbin/nologin', '/bin/false', '/sbin/nologin');
```

## Use Cases

### Container Inspection

Inspect user configuration in container filesystems:

```bash
# Mount container filesystem and query users
export BEATS_HOSTFS=/var/lib/docker/overlay2/[container-id]/merged
osqueryi --extension osquery-extension
```

```sql
SELECT * FROM host_users;
```

### Forensics

Analyze user accounts from mounted forensic images:

```bash
export BEATS_HOSTFS=/mnt/forensics/linux-image
```

### Multi-Host Monitoring

Compare user configurations across multiple systems by collecting from different filesystem roots.

## Performance Considerations

- Reading from `/etc/passwd` is generally fast
- File is typically small (< 1MB)
- No complex parsing required
- Minimal system impact

## Security Considerations

- Requires read access to `/etc/passwd` (or `$BEATS_HOSTFS/etc/passwd`)
- `/etc/passwd` is world-readable on most systems
- Does not expose password hashes (those are in `/etc/shadow`)
- User enumeration can be security-sensitive

## Differences from osquery's `users` Table

The built-in osquery `users` table uses system APIs (getpwent) which always reads from the host system. The `host_users` table:

- Reads directly from `/etc/passwd` file
- Respects `BEATS_HOSTFS` for alternative filesystem roots
- Useful for container inspection and forensic analysis
- Does not query system user databases (LDAP, AD, etc.)

## Troubleshooting

### Empty Results

- Check that `/etc/passwd` exists at the configured root
- Verify read permissions on the file
- Ensure `BEATS_HOSTFS` is set correctly if using alternative root

### Missing Users

- This table only reads from `/etc/passwd`
- Users from LDAP, Active Directory, or other sources won't appear
- Use osquery's built-in `users` table for system-wide user enumeration
