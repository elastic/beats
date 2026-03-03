# This file is generated! See ext/osquery-extension/cmd/gentables.
<!-- DO NOT EDIT MANUALLY. Update specs/templates and re-run gentables. -->

# Osquery Extension for Elastic

This osquery extension provides additional custom tables that enhance osquery's capabilities with Elastic-specific functionality. The extension is designed to work seamlessly with Osquerybeat and provides deep system insights across Linux, macOS, and Windows platforms.

## Overview

The extension adds several custom tables to osquery that provide:
- Browser history analysis across multiple browsers
- Host system information access from containers (groups, users, processes)
- Deep file analysis and security auditing on macOS
- Windows Amcache inventory and normalized application view
- Windows Jump List parsing for recent and pinned entries

## Supported Platforms

| Name | Type | Linux | macOS | Windows |
|------|------|-------|-------|---------|
| `elastic_amcache_application` | table | ❌ | ❌ | ✅ |
| `elastic_amcache_application_file` | table | ❌ | ❌ | ✅ |
| `elastic_amcache_application_shortcut` | table | ❌ | ❌ | ✅ |
| `elastic_amcache_applications_view` | view | ❌ | ❌ | ✅ |
| `elastic_amcache_device_pnp` | table | ❌ | ❌ | ✅ |
| `elastic_amcache_driver_binary` | table | ❌ | ❌ | ✅ |
| `elastic_amcache_driver_package` | table | ❌ | ❌ | ✅ |
| `elastic_browser_history` | table | ✅ | ✅ | ✅ |
| `elastic_file_analysis` | table | ❌ | ✅ | ❌ |
| `elastic_host_groups` | table | ✅ | ✅ | ❌ |
| `elastic_host_processes` | table | ✅ | ❌ | ❌ |
| `elastic_host_users` | table | ✅ | ✅ | ❌ |
| `elastic_jumplists` | table | ❌ | ❌ | ✅ |
| `host_groups` | view | ✅ | ✅ | ❌ |
| `host_processes` | view | ✅ | ❌ | ❌ |
| `host_users` | view | ✅ | ✅ | ❌ |

---

## Tables
- [elastic_amcache_application](docs/tables/elastic_amcache_application.md)
- [elastic_amcache_application_file](docs/tables/elastic_amcache_application_file.md)
- [elastic_amcache_application_shortcut](docs/tables/elastic_amcache_application_shortcut.md)
- [elastic_amcache_device_pnp](docs/tables/elastic_amcache_device_pnp.md)
- [elastic_amcache_driver_binary](docs/tables/elastic_amcache_driver_binary.md)
- [elastic_amcache_driver_package](docs/tables/elastic_amcache_driver_package.md)
- [elastic_browser_history](docs/tables/elastic_browser_history.md)
- [elastic_file_analysis](docs/tables/elastic_file_analysis.md)
- [elastic_host_groups](docs/tables/elastic_host_groups.md)
- [elastic_host_processes](docs/tables/elastic_host_processes.md)
- [elastic_host_users](docs/tables/elastic_host_users.md)
- [elastic_jumplists](docs/tables/elastic_jumplists.md)

## Views
- [elastic_amcache_applications_view](docs/views/elastic_amcache_applications_view.md)
- [host_groups](docs/views/host_groups.md)
- [host_processes](docs/views/host_processes.md)
- [host_users](docs/views/host_users.md)

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
