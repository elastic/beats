# elastic_file_analysis

Perform comprehensive security analysis of executable files on macOS.

## Platforms

- ❌ Linux
- ✅ macOS (Darwin)
- ❌ Windows

## Description

This table provides deep analysis of executable files and applications on macOS systems. It combines multiple macOS system tools to extract metadata, code signing information, library dependencies, symbols, and embedded strings from binary files.

This is particularly useful for:
- Malware analysis and detection
- Code signing verification
- Security auditing of applications
- Binary forensics and reverse engineering
- Supply chain security assessment
- Application inventory and compliance
- Detecting tampered or suspicious executables

## Schema

| Column | Type | Description |
|--------|------|-------------|
| `path` | `TEXT` | Absolute path to the file being analyzed |
| `mode` | `TEXT` | File permissions (e.g., '0755') |
| `uid` | `BIGINT` | File owner user ID |
| `gid` | `BIGINT` | File owner group ID |
| `size` | `BIGINT` | File size in bytes |
| `atime` | `BIGINT` | Last access time (Unix timestamp) |
| `mtime` | `BIGINT` | Last modification time (Unix timestamp) |
| `ctime` | `BIGINT` | Last status change time (Unix timestamp) |
| `filetype` | `TEXT` | File type and architecture from `file` command |
| `codesign` | `TEXT` | Code signing information from `codesign -dvv` |
| `dependencies` | `TEXT` | Linked libraries from `otool -L` |
| `symbols` | `TEXT` | Exported symbols from `nm` |
| `strings` | `TEXT` | Printable strings from binary (>= 4 characters) |

## How It Works

When you query this table with a file path, it performs a comprehensive analysis by executing multiple macOS command-line tools:

1. **File metadata**: Uses standard file system calls to get size, timestamps, permissions
2. **File type detection**: Runs `file` command to identify binary type and architecture
3. **Code signing**: Executes `codesign -dvv` to extract signing certificate and entitlements
4. **Dependencies**: Uses `otool -L` to list dynamically linked libraries (dylibs)
5. **Symbols**: Runs `nm` to extract exported function names and symbols
6. **Strings**: Executes `strings` command to extract printable text from the binary

All command outputs are captured and stored in their respective columns.

## Examples

### Basic Analysis

```sql
-- Analyze a specific executable
SELECT * FROM elastic_file_analysis 
WHERE path = '/Applications/Safari.app/Contents/MacOS/Safari';

-- Analyze all executables in a directory
SELECT path, filetype, size 
FROM elastic_file_analysis 
WHERE path LIKE '/usr/local/bin/%';

-- Get quick overview (metadata only)
SELECT path, filetype, size, mode, uid, gid
FROM elastic_file_analysis 
WHERE path = '/usr/bin/sudo';
```

### Code Signing Verification

```sql
-- Check code signing status
SELECT 
    path,
    CASE 
        WHEN codesign LIKE '%Signature=adhoc%' THEN 'Ad-hoc signed'
        WHEN codesign LIKE '%Authority=%Apple%' THEN 'Apple signed'
        WHEN codesign LIKE '%Authority=%' THEN 'Developer signed'
        ELSE 'Not signed or invalid'
    END as signing_status
FROM elastic_file_analysis 
WHERE path = '/Applications/Calculator.app/Contents/MacOS/Calculator';

-- Find unsigned executables in /Applications
SELECT path
FROM elastic_file_analysis 
WHERE path LIKE '/Applications/%.app/Contents/MacOS/%'
  AND (codesign = '' OR codesign NOT LIKE '%Signature=%');

-- Extract signing authority
SELECT 
    path,
    substr(codesign, 
           instr(codesign, 'Authority='), 
           instr(substr(codesign, instr(codesign, 'Authority=')), char(10))) as authority
FROM elastic_file_analysis 
WHERE path = '/usr/bin/codesign'
  AND codesign LIKE '%Authority=%';

-- Check for hardened runtime
SELECT path
FROM elastic_file_analysis 
WHERE path LIKE '/Applications/%'
  AND codesign LIKE '%flags=%' 
  AND codesign LIKE '%runtime%';
```

### Dependency Analysis

```sql
-- List all library dependencies
SELECT path, dependencies 
FROM elastic_file_analysis 
WHERE path = '/usr/bin/ssh';

-- Find executables using a specific library
SELECT path
FROM elastic_file_analysis 
WHERE path LIKE '/usr/local/bin/%'
  AND dependencies LIKE '%libssl%';

-- Detect suspicious library loading paths
SELECT path, dependencies
FROM elastic_file_analysis 
WHERE path LIKE '/Applications/%'
  AND (dependencies LIKE '%/tmp/%' 
    OR dependencies LIKE '%@executable_path/%'
    OR dependencies LIKE '%@rpath/%');
```

### Symbol Analysis

```sql
-- Check for debugging symbols
SELECT path,
       CASE WHEN symbols LIKE '%_debug_%' THEN 'Has debug symbols' 
            ELSE 'Stripped' 
       END as debug_status
FROM elastic_file_analysis 
WHERE path = '/usr/bin/python3';

-- Find executables with specific function symbols
SELECT path
FROM elastic_file_analysis 
WHERE path LIKE '/usr/bin/%'
  AND symbols LIKE '%_network_%'
  AND symbols LIKE '%_socket_%';

-- Detect obfuscated binaries (few or no symbols)
SELECT path, length(symbols) as symbol_length
FROM elastic_file_analysis 
WHERE path LIKE '/Applications/%/MacOS/%'
  AND length(symbols) < 100;
```

### String Analysis (Forensics)

```sql
-- Search for URLs in binaries
SELECT path
FROM elastic_file_analysis 
WHERE path = '/Applications/Safari.app/Contents/MacOS/Safari'
  AND strings LIKE '%http://%' OR strings LIKE '%https://%';

-- Find hardcoded credentials patterns
SELECT path
FROM elastic_file_analysis 
WHERE path LIKE '/usr/local/bin/%'
  AND (strings LIKE '%password%' 
    OR strings LIKE '%api_key%'
    OR strings LIKE '%secret%');

-- Detect potential C2 indicators
SELECT path
FROM elastic_file_analysis 
WHERE strings LIKE '%POST%' 
  AND strings LIKE '%User-Agent%'
  AND strings LIKE '%http%';

-- Find configuration paths
SELECT path
FROM elastic_file_analysis 
WHERE path = '/usr/sbin/sshd'
  AND strings LIKE '%/etc/%';
```

### Malware Detection

```sql
-- Find recently modified executables with suspicious traits
SELECT 
    path,
    mtime,
    codesign,
    CASE 
        WHEN codesign = '' THEN 'unsigned'
        WHEN codesign LIKE '%adhoc%' THEN 'ad-hoc signed'
        ELSE 'signed'
    END as sign_status
FROM elastic_file_analysis 
WHERE path LIKE '/usr/local/%'
  AND mtime > (strftime('%s', 'now') - 86400 * 7)  -- Last 7 days
  AND (codesign = '' OR codesign LIKE '%adhoc%');

-- Detect executables in unusual locations
SELECT path, filetype, size
FROM elastic_file_analysis 
WHERE (path LIKE '/tmp/%' 
    OR path LIKE '/var/tmp/%'
    OR path LIKE '/dev/shm/%'
    OR path LIKE '/.%')  -- Hidden directories
  AND filetype LIKE '%executable%';

-- Find binaries with suspicious strings
SELECT path
FROM elastic_file_analysis 
WHERE strings LIKE '%/bin/sh%'
  AND strings LIKE '%exec%'
  AND strings LIKE '%/dev/tcp/%';
```

### Application Inventory

```sql
-- List all application executables with metadata
SELECT 
    path,
    size / 1024 / 1024 as size_mb,
    datetime(mtime, 'unixepoch') as modified,
    substr(codesign, instr(codesign, 'TeamIdentifier='), 30) as team_id
FROM elastic_file_analysis 
WHERE path LIKE '/Applications/%.app/Contents/MacOS/%';

-- Compare file types and architectures
SELECT 
    CASE 
        WHEN filetype LIKE '%arm64%' THEN 'ARM64'
        WHEN filetype LIKE '%x86_64%' THEN 'x86_64'
        WHEN filetype LIKE '%universal%' THEN 'Universal'
        ELSE 'Other'
    END as architecture,
    COUNT(*) as count
FROM elastic_file_analysis 
WHERE path LIKE '/Applications/%/MacOS/%'
GROUP BY architecture;
```

## Comparison with Built-in Tables

| Feature | `elastic_file_analysis` | `file` | `hash` |
|---------|-----------------|--------|--------|
| Platform | macOS only | All platforms | All platforms |
| File metadata | ✅ Yes | ✅ Yes | ✅ Yes |
| Code signing | ✅ Yes | ❌ No | ❌ No |
| Dependencies | ✅ Yes | ❌ No | ❌ No |
| Symbols | ✅ Yes | ❌ No | ❌ No |
| Strings | ✅ Yes | ❌ No | ❌ No |
| File type detection | ✅ Yes | ✅ Limited | ❌ No |
| Cryptographic hash | ❌ No | ❌ No | ✅ Yes |
| Performance | Slow (spawns processes) | Fast | Medium |

## Performance Considerations

- **Heavy operation**: Each query spawns multiple child processes (`file`, `codesign`, `otool`, `nm`, `strings`)
- **Large files**: Analysis time increases with file size
- **String extraction**: Can be slow on large binaries
- **Use specific paths**: Avoid wildcard queries on large directories
- **Caching**: Consider caching results for frequently queried files
- **Batch processing**: Query multiple specific paths rather than wildcards
- **Timeout risks**: Very large binaries may timeout (>100MB)

### Optimization Tips

```sql
-- ❌ Avoid: Wildcard queries on large directories
SELECT * FROM elastic_file_analysis WHERE path LIKE '/usr/%';

-- ✅ Better: Specific paths
SELECT * FROM elastic_file_analysis WHERE path IN (
    '/usr/bin/curl',
    '/usr/bin/ssh',
    '/usr/sbin/sshd'
);

-- ✅ Better: Query only needed columns
SELECT path, filetype, codesign FROM elastic_file_analysis WHERE path = '/usr/bin/sudo';
```

## Security Considerations

- **Command injection**: Input paths are validated before executing commands
- **Privilege escalation**: Requires read access to target files
- **Sensitive data exposure**: `strings` output may contain passwords or keys
- **Resource consumption**: Multiple process spawns per query
- **Audit logging**: File analysis queries should be logged
- **Permission denied**: Some system files may be inaccessible

## Troubleshooting

### Empty or Missing Fields

1. **Empty `codesign` field**:
   - File is not signed
   - File is not a Mach-O binary
   - `codesign` command failed (check file permissions)

2. **Empty `dependencies` field**:
   - File has no dynamic library dependencies (statically linked)
   - File is not a binary executable
   - `otool` command failed

3. **Empty `symbols` field**:
   - Binary is stripped (no symbol table)
   - File is not a binary
   - `nm` command failed

4. **Empty `strings` field**:
   - Binary has no printable strings >= 4 characters
   - `strings` command failed

### Common Errors

```sql
-- Check if file exists
SELECT path FROM elastic_file_analysis WHERE path = '/nonexistent/file';
-- Returns empty result if file doesn't exist

-- Verify file is readable
-- Check permissions with file table first
SELECT * FROM file WHERE path = '/usr/bin/sudo';
```

### Understanding Code Signing Output

The `codesign` field contains output from `codesign -dvv`:

```
Executable=/Applications/Safari.app/Contents/MacOS/Safari
Identifier=com.apple.Safari
Format=app bundle with Mach-O universal (x86_64 arm64e)
CodeDirectory v=20500 size=... flags=0x10000(runtime) hashes=...
Signature size=...
Authority=Software Signing
Authority=Apple Code Signing Certification Authority
Authority=Apple Root CA
TeamIdentifier=not set
CDHash=...
```

Key indicators:
- **Authority**: Who signed the binary (Apple, Developer, etc.)
- **flags**: Runtime hardening and other protections
- **TeamIdentifier**: Developer team ID
- **adhoc**: Locally signed, not by Apple or verified developer

### File Type Examples

The `filetype` field shows output from the `file` command:

- `Mach-O 64-bit executable x86_64`
- `Mach-O universal binary with 2 architectures: [x86_64:Mach-O 64-bit executable x86_64] [arm64e:Mach-O 64-bit executable arm64e]`
- `script text executable`
- `data` (not recognized as executable)

## Implementation Details

### Commands Executed

For each file path queried:

1. **File metadata**: `stat()` system calls
2. **File type**: `file <path>`
3. **Code signing**: `codesign -dvv <path> 2>&1`
4. **Dependencies**: `otool -L <path> 2>&1`
5. **Symbols**: `nm <path> 2>&1`
6. **Strings**: `strings <path> 2>&1`

All commands redirect stderr to stdout to capture errors and warnings.

### String Extraction

The `strings` command extracts sequences of printable characters:
- Minimum length: 4 characters (default `strings` behavior)
- Character set: ASCII printable characters (0x20-0x7E)
- Encoding: UTF-8 compatible
- Output size: Can be very large for big binaries

### Security Measures

- Path validation: Prevents command injection
- Read-only operations: No file modification
- Error handling: Command failures don't crash extension
- Timeout protection: Commands are executed with reasonable timeouts

### Platform-Specific Tools

All tools are standard macOS utilities:
- `file` - Part of macOS base system
- `codesign` - macOS developer tools
- `otool` - macOS developer tools (part of Xcode CLI tools)
- `nm` - macOS developer tools (part of Xcode CLI tools)
- `strings` - macOS base system

**Note**: Xcode Command Line Tools must be installed:
```bash
xcode-select --install
```

### Limitations

- **macOS only**: Uses macOS-specific tools and formats
- **Mach-O format**: Designed for macOS executable format
- **Performance**: Not suitable for recursive directory scans
- **Large files**: May timeout on very large binaries
- **Tool availability**: Requires standard macOS utilities
- **No caching**: Each query re-executes all commands

## Related Tables

- `file` (built-in) - Basic file metadata (all platforms)
- `hash` (built-in) - Cryptographic hashes of files
- `processes` (built-in) - Running processes (can correlate with analyzed binaries)
- `host_processes` - Host processes when running in container
- `signature` (built-in, Windows) - Windows code signing verification
- `apps` (built-in, macOS) - Installed macOS applications

## Example Use Cases

### Security Audit Workflow

```sql
-- 1. Find recently modified executables
SELECT path, mtime FROM elastic_file_analysis 
WHERE path LIKE '/usr/local/bin/%' 
  AND mtime > (strftime('%s', 'now') - 86400 * 7);

-- 2. Analyze code signing
SELECT path, codesign FROM elastic_file_analysis 
WHERE path = '/usr/local/bin/suspicious-binary';

-- 3. Check dependencies for unusual libraries
SELECT path, dependencies FROM elastic_file_analysis 
WHERE path = '/usr/local/bin/suspicious-binary';

-- 4. Search for suspicious strings
SELECT path, strings FROM elastic_file_analysis 
WHERE path = '/usr/local/bin/suspicious-binary'
  AND (strings LIKE '%password%' OR strings LIKE '%/tmp/%');
```

### Incident Response

When investigating a suspicious binary:
1. Check code signing to verify authenticity
2. Analyze dependencies for suspicious libraries
3. Extract strings to find C2 domains or credentials
4. Compare symbols against known malware signatures
5. Verify file metadata (timestamps, size)
6. Cross-reference with process table for running instances
