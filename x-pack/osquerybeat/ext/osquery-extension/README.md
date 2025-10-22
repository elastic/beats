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

Each table has detailed documentation in its own file:

### 1. [browser_history](docs/browser_history.md)
Query browser history from multiple browsers (Chrome, Edge, Firefox, Safari) with unified schema and advanced filtering capabilities.

**Platforms:** Linux, macOS, Windows

### 2. [host_users](docs/host_users.md)
Query user information from `/etc/passwd`. Useful for reading user information from alternative filesystem roots (e.g., container inspection, mounted drives).

**Platforms:** Linux, macOS

### 3. [host_groups](docs/host_groups.md)
Query group information from `/etc/group`. Reads group information from the configured filesystem root.

**Platforms:** Linux, macOS

### 4. [host_processes](docs/host_processes.md)
Query detailed process information by reading directly from `/proc` filesystem. Provides more control and can read from alternative filesystem roots.

**Platforms:** Linux

### 5. [elastic_file_analysis](docs/elastic_file_analysis.md)
Perform deep file analysis using native system tools. Provides comprehensive file metadata, code signing information, dependencies, symbols, and strings extraction.

**Platforms:** macOS

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

---

## Additional Resources

- **Table Documentation**: [docs/](docs/) - Detailed documentation for each table including configuration, examples, and security considerations
- **Development**: See the main [beats documentation](https://github.com/elastic/beats)
- **Osquery**: [osquery.io](https://osquery.io/)
