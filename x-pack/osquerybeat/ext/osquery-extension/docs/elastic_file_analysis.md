# elastic_file_analysis

Perform deep file analysis using native macOS tools. Provides comprehensive file metadata, code signing information, dependencies, symbols, and strings extraction.

## Platforms

- ❌ Linux
- ✅ macOS
- ❌ Windows

## Schema

| Column | Type | Description |
|--------|------|-------------|
| `path` | `TEXT` | Absolute path to the analyzed file (required constraint) |
| `mode` | `TEXT` | File permissions in octal format (e.g., "755") |
| `uid` | `BIGINT` | Owner user ID |
| `gid` | `BIGINT` | Owner group ID |
| `size` | `BIGINT` | File size in bytes |
| `mtime` | `BIGINT` | Last modification time (Unix timestamp) |
| `file_type` | `TEXT` | File type information from `file` command |
| `code_sign` | `TEXT` | Code signing information from `codesign -dvvv` |
| `dependencies` | `TEXT` | Dynamic library dependencies from `otool -L` |
| `symbols` | `TEXT` | Exported symbols from `nm` command |
| `strings` | `TEXT` | Printable strings from `strings -a` command |

## Required Constraints

This table requires the `path` column in the WHERE clause:

```sql
-- ✅ Correct - path specified
SELECT * FROM elastic_file_analysis WHERE path = '/usr/bin/ls';

-- ❌ Error - path constraint missing
SELECT * FROM elastic_file_analysis;
```

## Examples

### Basic File Analysis

```sql
-- Analyze a single binary
SELECT 
    path,
    file_info,
    signature_info
FROM elastic_file_analysis
WHERE path = '/usr/bin/ls';

-- Analyze application bundle
SELECT *
FROM elastic_file_analysis
WHERE path = '/Applications/Safari.app/Contents/MacOS/Safari';
```

### Code Signing Verification

```sql
-- Check if binary is signed
SELECT 
    path,
    code_sign
FROM elastic_file_analysis
WHERE path = '/Applications/TextEdit.app/Contents/MacOS/TextEdit';

-- Verify Apple-signed binaries
SELECT 
    path,
    code_sign
FROM elastic_file_analysis
WHERE path IN (
    '/usr/bin/ls',
    '/usr/bin/ps',
    '/bin/bash'
)
AND code_sign LIKE '%Apple%';

-- Find unsigned binaries
SELECT path, code_sign
FROM elastic_file_analysis
WHERE path = '/usr/local/bin/my-app'
AND code_sign NOT LIKE '%Signature%';
```

### Dependency Analysis

```sql
-- List all dependencies of a binary
SELECT 
    path,
    dependencies
FROM elastic_file_analysis
WHERE path = '/usr/bin/python3';

-- Find binaries depending on specific library
SELECT path, dependencies
FROM elastic_file_analysis
WHERE path = '/usr/local/bin/my-app'
AND dependencies LIKE '%libssl%';

-- Check for suspicious library dependencies
SELECT path, dependencies
FROM elastic_file_analysis
WHERE path = '/Applications/MyApp.app/Contents/MacOS/MyApp'
AND dependencies LIKE '%/tmp/%';
```

### Symbol Analysis

```sql
-- Extract exported symbols
SELECT 
    path,
    symbols
FROM elastic_file_analysis
WHERE path = '/usr/local/lib/mylib.dylib';

-- Check for specific function exports
SELECT path, symbols
FROM elastic_file_analysis
WHERE path = '/usr/local/lib/mylib.dylib'
AND symbols LIKE '%my_function%';

-- Analyze symbol patterns for malware detection
SELECT path, symbols
FROM elastic_file_analysis
WHERE path = '/tmp/suspicious-binary'
AND (symbols LIKE '%exec%' OR symbols LIKE '%system%');
```

### Strings Extraction

```sql
-- Extract strings from binary (useful for malware analysis)
SELECT 
    path,
    strings
FROM elastic_file_analysis
WHERE path = '/tmp/suspicious-binary';

-- Find URLs in binary
SELECT path, strings
FROM elastic_file_analysis
WHERE path = '/Applications/MyApp.app/Contents/MacOS/MyApp'
AND strings LIKE '%http%';

-- Find configuration or credential patterns
SELECT path, strings
FROM elastic_file_analysis
WHERE path = '/usr/local/bin/my-app'
AND (
    strings LIKE '%password%'
    OR strings LIKE '%api_key%'
    OR strings LIKE '%token%'
);
```

### Security Analysis

```sql
-- Verify binaries in critical paths
SELECT 
    path,
    mode,
    uid,
    gid,
    code_sign,
    file_type
FROM elastic_file_analysis
WHERE path IN (
    '/usr/bin/sudo',
    '/usr/bin/su',
    '/usr/sbin/sshd'
);

-- Check for setuid binaries (potential privilege escalation)
SELECT path, mode, uid, file_type
FROM elastic_file_analysis
WHERE path = '/usr/local/bin/suspicious-binary'
AND mode LIKE '4%';  -- setuid bit

-- Analyze file permissions and ownership
SELECT path, mode, uid, gid
FROM elastic_file_analysis
WHERE path = '/Applications/MyApp.app/Contents/MacOS/MyApp';
```

### File Metadata Extraction

```sql
-- Get detailed file information
SELECT 
    path,
    file_type,
    size,
    mode,
    uid,
    gid,
    mtime
FROM elastic_file_analysis
WHERE path = '/usr/local/bin/my-binary';

-- Find recently modified binaries
SELECT path, file_type, mtime, size
FROM elastic_file_analysis
WHERE path = '/Applications/MyApp.app/Contents/MacOS/MyApp'
AND CAST(mtime AS INTEGER) > (strftime('%s', 'now') - 86400);  -- Last 24 hours

-- Compare file sizes
SELECT path, size, file_type
FROM elastic_file_analysis
WHERE path IN (
    '/usr/bin/python',
    '/usr/bin/python3'
);
```

## Output Format

The table returns raw text output from macOS system commands:

- **file_type**: Output from `file <path>` command (e.g., "Mach-O 64-bit executable x86_64")
- **code_sign**: Output from `codesign -dvvv <path>` (stderr), includes signing authority, identifier, etc.
- **dependencies**: Output from `otool -L <path>`, lists dynamic library dependencies
- **symbols**: Output from `nm <path>`, lists exported symbols
- **strings**: Output from `strings -a <path>`, extracts printable character sequences

### Example Output

```
file_type: Mach-O 64-bit executable x86_64
code_sign: Executable=/usr/bin/ls
Identifier=com.apple.ls
...
Authority=Software Signing
Authority=Apple Code Signing Certification Authority
...

dependencies: /usr/bin/ls:
	/usr/lib/libutil.dylib (compatibility version 1.0.0)
	/usr/lib/libncurses.5.4.dylib (compatibility version 5.4.0)
	/usr/lib/libSystem.B.dylib (compatibility version 1.0.0)

symbols: (List of function names, one per line)
_main
_some_function
...

strings: (Printable strings found in binary, one per line)
Hello World
/usr/local/
https://example.com
...
```

## Performance Considerations

- File analysis can be CPU and I/O intensive
- Large binaries take longer to analyze
- Symbol and string extraction can be memory-intensive
- Use specific path constraints (avoid wildcards in WHERE clause)
- Consider caching results for frequently analyzed files
- Limit string extraction length for large binaries

## Security Considerations

- Requires read access to target files
- May require elevated privileges for system binaries
- Strings may contain sensitive information (passwords, API keys)
- Be careful when analyzing untrusted binaries
- Consider sandboxing analysis operations
- Code signing verification helps detect tampering

## Use Cases

### Application Security

- Verify code signatures of installed applications
- Detect unsigned or tampered binaries
- Audit entitlements and capabilities

### Malware Analysis

- Extract strings for IOC identification
- Analyze dependencies for suspicious libraries
- Check for code signing anomalies

### Software Inventory

- Catalog installed applications and their versions
- Track library dependencies
- Monitor binary changes

### Compliance Auditing

- Verify all binaries are properly signed
- Check for unauthorized modifications
- Ensure proper file permissions

## Troubleshooting

### Empty Results

- Check that file exists at specified path
- Verify read permissions
- Ensure file is a supported format (Mach-O binary)

### Signature Info Empty

- File may not be code signed
- Check `codesign` tool availability
- Verify file is an executable or bundle

### Permission Denied

- Some system files require elevated privileges
- Run osquery with appropriate permissions
- Check file ownership and permissions

### Incomplete Symbol Information

- Stripped binaries have limited symbols
- Debug symbols may be in separate `.dSYM` files
- Some symbols may be private

## Implementation Details

Uses macOS native tools:
- `file`: File type identification
- `codesign`: Code signature verification
- `otool`: Mach-O binary analysis (dependencies)
- `nm`: Symbol extraction
- `strings`: String extraction

For best results, ensure these tools are available in the system PATH.
