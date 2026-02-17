# Osquery Extension for Elastic

This osquery extension provides additional custom tables that enhance osquery's capabilities with Elastic-specific functionality. The extension is designed to work seamlessly with Osquerybeat and provides deep system insights across Linux, macOS, and Windows platforms.

## Overview

The extension adds several custom tables to osquery that provide:
- Browser history analysis across multiple browsers
- Host system information access from containers (groups, users, processes)
- Deep file analysis and security auditing on macOS

## Supported Platforms

| Table | Linux | macOS | Windows |
|-------|-------|-------|---------|
| `elastic_browser_history` | ✅ | ✅ | ✅ |
| `elastic_host_groups` | ✅ | ✅ | ❌ |
| `host_groups` (view) | ✅ | ✅ | ❌ |
| `elastic_host_users` | ✅ | ✅ | ❌ |
| `host_users` (view) | ✅ | ✅ | ❌ |
| `elastic_host_processes` | ✅ | ❌ | ❌ |
| `host_processes` (view) | ✅ | ❌ | ❌ |
| `elastic_file_analysis` | ❌ | ✅ | ❌ |

---

## Tables

Each table has detailed documentation in its own file:

### 1. [elastic_browser_history](docs/tables/elastic_browser_history.md)
Query browser history from multiple browsers (Chrome, Edge, Firefox, Safari) with unified schema and advanced filtering capabilities.

**Platforms**: Linux, macOS, Windows

### 2. [elastic_host_groups](docs/tables/elastic_host_groups.md)
Query host system group information from the host's `/etc/group` (e.g. when running in a container with hostfs mounted). The [host_groups](docs/views/host_groups.md) view provides backward compatibility.

**Platforms**: Linux, macOS

### 3. [elastic_host_users](docs/tables/elastic_host_users.md)
Query host system user accounts from the host's `/etc/passwd` (e.g. when running in a container with hostfs mounted). The [host_users](docs/views/host_users.md) view provides backward compatibility.

**Platforms**: Linux, macOS

### 4. [elastic_host_processes](docs/tables/elastic_host_processes.md)
Query running process information from the host's `/proc` when running in a container (Linux only). The [host_processes](docs/views/host_processes.md) view provides backward compatibility.

**Platforms**: Linux

### 5. [elastic_file_analysis](docs/tables/elastic_file_analysis.md)
Comprehensive security analysis of executable files on macOS (file type, code signing, dependencies, symbols, strings). Query with a path constraint; uses `file`, `codesign`, `otool`, `nm`, and `strings`.

**Platforms**: macOS

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
osqueryi> .tables
  => elastic_browser_history

# Query the tables
osqueryi> SELECT * FROM elastic_browser_history LIMIT 10;
```

---

## Additional Resources

- **Table Documentation**: [docs/](docs/) - Detailed documentation for each table including configuration, examples, and security considerations
- **Development**: See the main [beats documentation](https://github.com/elastic/beats)
- **Osquery**: [osquery.io](https://osquery.io/)
