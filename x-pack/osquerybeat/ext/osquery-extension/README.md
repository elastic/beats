# Osquery Extension for Elastic

This osquery extension provides additional custom tables that enhance osquery's capabilities with Elastic-specific functionality. The extension is designed to work seamlessly with Osquerybeat and provides deep system insights across Linux, macOS, and Windows platforms.

## Overview

The extension adds several custom tables to osquery that provide:
- Browser history analysis across multiple browsers

## Supported Platforms

| Table | Linux | macOS | Windows |
|-------|-------|-------|---------|
| `browser_history` | ✅ | ✅ | ✅ |

---

## Tables

Each table has detailed documentation in its own file:

### 1. [browser_history](docs/browser_history.md)
Query browser history from multiple browsers (Chrome, Edge, Firefox, Safari) with unified schema and advanced filtering capabilities.

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
  => browser_history

# Query the tables
osqueryi> SELECT * FROM browser_history LIMIT 10;
```

---

## Additional Resources

- **Table Documentation**: [docs/](docs/) - Detailed documentation for each table including configuration, examples, and security considerations
- **Development**: See the main [beats documentation](https://github.com/elastic/beats)
- **Osquery**: [osquery.io](https://osquery.io/)
