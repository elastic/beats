# host_users

Query user account information from the host system's `/etc/passwd` file when running in a container environment.

## Platforms

- ✅ Linux
- ✅ macOS
- ❌ Windows

## Description

This table provides access to the host system's user account information when osquery is running inside a container. It reads from the `/hostfs/etc/passwd` file (where the host filesystem is mounted), allowing containers to inspect the host system's user account configuration without needing to query the container's own user database.

This is particularly useful for:
- Container security auditing
- Host system user inventory from within containers
- Compliance checks on host user configurations
- Detecting unauthorized user account creation on the host
- Investigating potential security incidents

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
| `uuid` | `TEXT` | User's UUID (Apple) or SID (Windows) - typically empty on Linux |

## How It Works

When osquery runs in a container with the host filesystem mounted at `/hostfs`, this table reads directly from `/hostfs/etc/passwd` to retrieve the host system's user information. This is different from osquery's built-in `users` table, which reads from the container's own `/etc/passwd` file.

### Mounting Host Filesystem

To use this table, mount the host's root filesystem into the container:

```bash
docker run -v /:/hostfs:ro osquery-container
```

## Examples

### Basic Queries

```sql
-- Get all host system users
SELECT * FROM host_users;

-- Find specific user by username
SELECT * FROM host_users WHERE username = 'root';

-- Find user by UID
SELECT * FROM host_users WHERE uid = 1000;

-- List all system users (typically UID < 1000)
SELECT username, uid, shell, directory 
FROM host_users 
WHERE uid < 1000 
ORDER BY uid;

-- List all regular users (typically UID >= 1000)
SELECT username, uid, description, directory, shell
FROM host_users 
WHERE uid >= 1000 AND uid < 65534
ORDER BY uid;
```

### Security Auditing

```sql
-- Find root user (UID 0)
SELECT * FROM host_users WHERE uid = 0;

-- Find users with login shells (potential security concern)
SELECT username, uid, shell, directory
FROM host_users 
WHERE shell NOT IN ('/bin/false', '/sbin/nologin', '/usr/sbin/nologin')
ORDER BY uid;

-- Find users with bash/zsh shells (interactive users)
SELECT username, uid, shell, directory
FROM host_users 
WHERE shell IN ('/bin/bash', '/bin/zsh', '/bin/sh')
ORDER BY uid;

-- Find users without home directories
SELECT username, uid, directory 
FROM host_users 
WHERE directory IN ('/', '/nonexistent', '/dev/null');

-- Find users with negative UIDs (unusual configuration)
SELECT username, uid, uid_signed 
FROM host_users 
WHERE uid_signed < 0;

-- Find duplicate UIDs (potential security issue)
SELECT uid, COUNT(*) as count, GROUP_CONCAT(username) as users
FROM host_users
GROUP BY uid
HAVING count > 1;

-- Find service accounts with home directories
SELECT username, uid, directory, shell
FROM host_users 
WHERE uid < 1000 
  AND directory NOT IN ('/', '/nonexistent', '/dev/null')
  AND shell NOT IN ('/bin/false', '/sbin/nologin', '/usr/sbin/nologin')
ORDER BY uid;
```

### Compliance and Inventory

```sql
-- Count total users on host
SELECT COUNT(*) as total_users FROM host_users;

-- Count system vs regular users
SELECT 
    CASE 
        WHEN uid < 1000 THEN 'system'
        WHEN uid >= 1000 AND uid < 65534 THEN 'regular'
        ELSE 'special'
    END as user_type,
    COUNT(*) as count
FROM host_users
GROUP BY user_type;

-- Compare container users vs host users
SELECT 
    hu.username as host_user,
    hu.uid as host_uid,
    u.username as container_user,
    u.uid as container_uid
FROM host_users hu
LEFT JOIN users u ON hu.username = u.username;

-- Find users that exist on host but not in container
SELECT username, uid FROM host_users
WHERE username NOT IN (SELECT username FROM users);

-- Find users that exist in container but not on host
SELECT username, uid FROM users
WHERE username NOT IN (SELECT username FROM host_users);

-- Audit shell usage
SELECT shell, COUNT(*) as user_count
FROM host_users
GROUP BY shell
ORDER BY user_count DESC;
```

### Forensics and Incident Response

```sql
-- Find recently created users (assuming UIDs are assigned sequentially)
SELECT username, uid, directory, shell
FROM host_users 
WHERE uid >= 1000 AND uid < 65534
ORDER BY uid DESC
LIMIT 10;

-- Find users with unusual home directories
SELECT username, uid, directory
FROM host_users 
WHERE directory NOT LIKE '/home/%' 
  AND directory NOT LIKE '/Users/%'
  AND directory NOT IN ('/', '/root', '/nonexistent', '/dev/null', '/var/%', '/usr/%', '/bin', '/sbin')
ORDER BY uid;

-- Find privileged users (UID 0 or in wheel/sudo group context)
SELECT username, uid, shell, directory
FROM host_users 
WHERE uid = 0 OR username IN ('root', 'admin');
```

## Comparison with Built-in Tables

| Feature | `host_users` | `users` |
|---------|--------------|---------|
| Data Source | Host's `/hostfs/etc/passwd` | Container's `/etc/passwd` |
| Use Case | Audit host system from container | Query container's own users |
| Requires `/hostfs` mount | ✅ Yes | ❌ No |
| Available in | Osquery extension | Built-in osquery |

## Performance Considerations

- Fast table - reads a single file from disk
- File is cached by the OS
- Minimal overhead for repeated queries
- No external dependencies or network calls
- Typically small file (hundreds of users at most)

## Security Considerations

- Requires read access to `/hostfs/etc/passwd`
- Should run with appropriate container permissions
- The `/hostfs` mount should be read-only (`:ro`)
- Contains no sensitive data (passwords are in `/etc/shadow`, not `/etc/passwd`)
- User enumeration is possible, but this is generally not sensitive information
- Useful for security auditing and compliance monitoring

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
   docker exec container ls -la /hostfs/etc/passwd
   ```

### Negative UID/GID Values

The `uid_signed` and `gid_signed` fields provide signed interpretations of the IDs, which can handle systems that use signed integers. On most Linux systems, UIDs/GIDs are unsigned, so the unsigned and signed versions will match. macOS systems may use negative values internally, which is why both fields are provided.

## Implementation Details

### File Format

The `/etc/passwd` file follows this format:
```
username:password:uid:gid:gecos:directory:shell
```

Example:
```
root:x:0:0:root:/root:/bin/bash
daemon:x:1:1:daemon:/usr/sbin:/usr/sbin/nologin
alice:x:1000:1000:Alice Smith,,,:/home/alice:/bin/bash
```

### Field Mapping

- `username`: First field (login name)
- Second field (password): Historically stored password hash, now typically `x` or `*` (actual hashes in `/etc/shadow`)
- `uid`: Third field (user ID), parsed as unsigned integer
- `gid`: Fourth field (primary group ID), parsed as unsigned integer
- `description`: Fifth field (GECOS - General Electric Comprehensive Operating System information, typically full name and contact info)
- `directory`: Sixth field (home directory path)
- `shell`: Seventh field (login shell)
- `uid_signed`: Third field parsed as signed integer
- `gid_signed`: Fourth field parsed as signed integer
- `uuid`: Not in `/etc/passwd`, typically empty on Linux

### Platform Differences

**Linux:**
- Standard `/etc/passwd` format
- System users typically have UID < 1000
- Regular users typically have UID >= 1000
- Service accounts often have UID < 100
- Special UID 65534 reserved for `nobody`

**macOS:**
- Also uses `/etc/passwd` for local users
- Many users managed by OpenDirectory instead
- System users often have UID < 500
- Regular users typically have UID >= 500
- The `uuid` field may be populated on macOS systems

### Common UID Ranges

- **0**: Root user
- **1-99**: System accounts (daemons, services)
- **100-999**: System accounts and pseudo-users
- **1000-59999**: Regular user accounts
- **60000-64999**: Reserved
- **65534**: `nobody` user (unprivileged)
- **65535**: Often reserved or invalid

## Related Tables

- `users` - Query container's own user database
- `host_groups` - Query host system's group information
- `host_processes` - Query host system's running processes
- `logged_in_users` (built-in) - See currently logged-in users
- `user_groups` (built-in) - Map users to their group memberships
