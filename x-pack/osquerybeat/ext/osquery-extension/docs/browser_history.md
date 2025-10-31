# browser_history

Query browser history from multiple browsers with a unified schema. Supports Chrome, Edge, Firefox, and Safari across Linux, macOS, and Windows.

## Platforms

- ✅ Linux
- ✅ macOS  
- ✅ Windows

## Schema

### Universal Fields (Available Across All Browsers)

| Column | Type | Description |
|--------|------|-------------|
| `timestamp` | `BIGINT` | Unix timestamp in seconds since epoch |
| `datetime` | `TEXT` | Human-readable datetime string (RFC3339 format) |
| `url_id` | `BIGINT` | Unique URL identifier |
| `scheme` | `TEXT` | URL scheme/protocol (e.g., "https", "http", "file", "ftp") |
| `domain` | `TEXT` | Domain/hostname extracted from URL (e.g., "github.com", "google.com") |
| `url` | `TEXT` | Full URL visited |
| `title` | `TEXT` | Page title |
| `browser` | `TEXT` | Browser name (chrome, edge, firefox, safari, brave) |
| `parser` | `TEXT` | Parser used to extract the history |
| `user` | `TEXT` | Username/profile owner |
| `profile_name` | `TEXT` | Browser profile name |
| `transition_type` | `TEXT` | Navigation method: how user reached this page (TYPED, LINK, BOOKMARK, RELOAD, REDIRECT, etc.) |
| `referring_url` | `TEXT` | URL that linked to this page (if navigated via link) |
| `visit_id` | `BIGINT` | Unique visit identifier |
| `from_visit_id` | `BIGINT` | Visit ID that led to this visit (navigation chain) |
| `visit_source` | `TEXT` | Data origin: where this visit data came from (browsed/local, synced, imported, extension) |
| `is_hidden` | `INTEGER` | Whether visit is hidden (1) or visible (0) |
| `history_path` | `TEXT` | Path to the browser history database file |
| `custom_data_dir` | `TEXT` | Custom data directory path (optional, for querying non-standard locations) |

### Chromium-Specific Fields (Chrome, Edge, Brave)

| Column | Type | Description |
|--------|------|-------------|
| `ch_visit_duration_ms` | `BIGINT` | Duration of visit in milliseconds (Chromium only) |

### Firefox-Specific Fields

| Column | Type | Description |
|--------|------|-------------|
| `ff_session_id` | `INTEGER` | Firefox session tracking identifier |
| `ff_frecency` | `INTEGER` | Firefox frecency score (frequency + recency algorithm) |

### Safari-Specific Fields

| Column | Type | Description |
|--------|------|-------------|
| `sf_domain_expansion` | `TEXT` | Safari domain classification/expansion |
| `sf_load_successful` | `INTEGER` | Whether page loaded successfully (1) or failed (0) |

## Supported Browsers

| Browser | Linux | macOS | Windows |
|---------|-------|-------|---------|
| Chrome | ✅ | ✅ | ✅ |
| Edge | ✅ | ✅ | ✅ |
| Firefox | ✅ | ✅ | ✅ |
| Safari | ❌ | ✅ | ❌ |

## Auto-Discovery

The table automatically discovers browsers from standard user profile locations:

### Linux
- Chrome: `~/.config/google-chrome/`
- Edge: `~/.config/microsoft-edge/`
- Firefox: `~/.mozilla/firefox/`

### macOS
- Chrome: `~/Library/Application Support/Google/Chrome/`
- Edge: `~/Library/Application Support/Microsoft Edge/`
- Firefox: `~/Library/Application Support/Firefox/`
- Safari: `~/Library/Safari/`

### Windows
- Chrome: `%LOCALAPPDATA%\Google\Chrome\User Data\`
- Edge: `%LOCALAPPDATA%\Microsoft\Edge\User Data\`
- Firefox: `%APPDATA%\Mozilla\Firefox\`

## Configuration

### Custom Data Directories

The table automatically discovers browsers from standard user profile locations shown above. To query history from non-standard locations (useful for forensics, backups, or mounted drives), use the `custom_data_dir` constraint:

```sql
-- Query from custom Chrome profile
SELECT * FROM browser_history 
WHERE custom_data_dir = '/mnt/backup/Users/john/AppData/Local/Google';

-- Query all profiles in a custom location using glob
SELECT * FROM browser_history 
WHERE custom_data_dir GLOB '/forensics/users/*/Library/Application Support/Google';

-- Query from mounted container filesystem
SELECT * FROM browser_history 
WHERE custom_data_dir = '/var/lib/docker/overlay2/.../home/user/.config/google-chrome';
```

The `custom_data_dir` should point to the browser's base directory (not the profile directory). The table will discover all profiles within that location.

## Examples

### Basic Queries

```sql
-- Get all history from all discovered browsers
SELECT * FROM browser_history;

-- Get history from specific browser
SELECT * FROM browser_history WHERE browser = 'chrome';

-- Get history from specific profile
SELECT * FROM browser_history 
WHERE browser = 'firefox' AND profile_name = 'default-release';

-- Recent history (last 7 days)
SELECT url, title, browser, datetime 
FROM browser_history 
WHERE timestamp > (strftime('%s', 'now') - 604800)
ORDER BY timestamp DESC;
```

### Advanced Filtering

```sql
-- Search for specific domains (using domain field)
SELECT browser, profile_name, url, title, datetime
FROM browser_history
WHERE domain = 'github.com'
ORDER BY timestamp DESC;

-- Search domains with pattern matching
SELECT browser, domain, url, title
FROM browser_history
WHERE domain LIKE '%.google.com'
ORDER BY timestamp DESC;

-- Filter by URL scheme/protocol
SELECT browser, scheme, url, title, datetime
FROM browser_history
WHERE scheme = 'https'
ORDER BY timestamp DESC;

-- Find non-HTTPS visits (potential security concern)
SELECT browser, url, title, datetime
FROM browser_history
WHERE scheme IN ('http', 'ftp')
ORDER BY timestamp DESC;

-- Most visited URLs (count actual visits)
SELECT url, domain, COUNT(*) as visit_count
FROM browser_history
GROUP BY url
ORDER BY visit_count DESC
LIMIT 20;

-- Most visited domains
SELECT domain, COUNT(*) as visits, COUNT(DISTINCT url_id) as unique_urls
FROM browser_history
WHERE domain != ''
GROUP BY domain
ORDER BY visits DESC
LIMIT 20;

-- Find typed URLs (direct navigation by user)
SELECT url, title, browser, datetime
FROM browser_history
WHERE transition_type = 'TYPED'
ORDER BY timestamp DESC;

-- Analyze transition types (how users navigate)
SELECT transition_type, COUNT(*) as count
FROM browser_history
GROUP BY transition_type
ORDER BY count DESC;

-- Find hidden entries (private or deleted history)
SELECT url, title, browser, datetime
FROM browser_history
WHERE is_hidden = 1;

-- Filter by visit source (data origin)
SELECT url, title, browser, visit_source
FROM browser_history
WHERE visit_source = 'synced'  -- Only show synced history from other devices
ORDER BY timestamp DESC;

-- Combine transition type and visit source for forensic analysis
SELECT 
    transition_type,
    visit_source,
    COUNT(*) as count
FROM browser_history
GROUP BY transition_type, visit_source
ORDER BY count DESC;
```

## Performance Considerations

- Browser history databases can be large (thousands of entries)
- Use timestamp filters to limit results
- Apply browser/profile filters when possible
- Custom data directory queries may be slower due to discovery overhead

## Security Considerations

- Requires read access to browser profile directories
- May contain sensitive browsing information
- Consider data retention policies
- Be aware of privacy regulations (GDPR, CCPA, etc.)
- Recommend running with appropriate access controls

## Troubleshooting

### No Results Returned

1. Check browser profile paths exist
2. Verify read permissions on browser directories
3. Ensure browsers are not running (locked database files)
4. Check for profile names (especially Firefox with random suffixes)

### Incomplete History

- Some browsers may clear history automatically
- Private/incognito mode is not recorded
- Browser may use multiple profiles

### Database Locked Errors

- Browser must be closed before querying
- Or use `custom_data_dir` to query copies of the databases

## Implementation Details

### Database Sources
- **Chrome/Edge/Brave**: Reads from `History` SQLite database
- **Firefox**: Reads from `places.sqlite` database  
- **Safari**: Reads from `History.db` database

### Browser-Specific Fields
- **Chromium fields** (`ch_*`): Only populated for Chrome, Edge, Brave, and other Chromium-based browsers
- **Firefox fields** (`ff_*`): Only populated for Firefox
- **Safari fields** (`sf_*`): Only populated for Safari
- Universal fields are always populated regardless of browser

### Data Model
Browser history databases use a two-table structure:
- **URLs table**: Stores each unique URL with aggregate statistics
- **Visits table**: Stores each individual visit with timestamp and navigation details

**This table returns one row per visit**, providing detailed information about each individual page view.

#### Calculating Aggregates

Since each row represents a single visit, you can easily calculate statistics using SQL:

```sql
-- Count total visits per URL
SELECT url, COUNT(*) as visit_count
FROM browser_history
GROUP BY url;

-- Count typed visits per URL
SELECT url, COUNT(*) as typed_count
FROM browser_history
WHERE transition_type = 'TYPED'
GROUP BY url;

-- Most visited domains
SELECT domain, COUNT(*) as visits
FROM browser_history
WHERE domain != ''
GROUP BY domain
ORDER BY visits DESC
LIMIT 10;
```

### Field Distinction: transition_type vs visit_source

**`transition_type`** = **HOW** the user navigated (the navigation method)
**`visit_source`** = **WHERE** the visit data originated (data provenance)

**Example**: A visit can be `transition_type = 'TYPED'` and `visit_source = 'synced'`, meaning the user typed the URL on another device, and it synced to this browser.

#### Chromium transition_type Values (Chrome, Edge, Brave)

Based on [Chromium's PageTransition enum](https://source.chromium.org/chromium/chromium/src/+/main:ui/base/page_transition_types.h):

**Core Types** (lower 8 bits):
- `LINK` (0) - User clicked a link
- `TYPED` (1) - User typed URL in address bar
- `AUTO_BOOKMARK` (2) - Generated from autocomplete bookmark suggestion
- `AUTO_SUBFRAME` (3) - Subframe navigation not initiated by user
- `MANUAL_SUBFRAME` (4) - Subframe navigation initiated by user
- `GENERATED` (5) - User typed something that triggered a search
- `AUTO_TOPLEVEL` (6) - Top-level navigation automatically from autocomplete
- `FORM_SUBMIT` (7) - User submitted a form
- `RELOAD` (8) - User reloaded/refreshed the page
- `KEYWORD` (9) - URL generated from search bar keyword
- `KEYWORD_GENERATED` (10) - Keyword search that generated a URL

**Qualifiers** (upper bits, combined with `|`):
- `BLOCKED` - Navigation was blocked by security
- `BACK_FORWARD` - Used back/forward button
- `FROM_ADDRESS_BAR` - Navigation from address bar (distinct from TYPED)
- `HOME_PAGE` - Navigation to home page
- `FROM_API` - Navigation from browser extension/API
- `CHAIN_START` - Start of navigation chain
- `CHAIN_END` - End of navigation chain
- `CLIENT_REDIRECT` - Client-side redirect (JavaScript, meta refresh)
- `SERVER_REDIRECT` - HTTP 3xx server redirect

Example: `TYPED|FROM_ADDRESS_BAR` indicates user typed in address bar.

#### Firefox transition_type Values

Based on [Mozilla's Places visit types](https://searchfox.org/mozilla-central/source/toolkit/components/places/nsINavHistoryService.idl):

- `LINK` (1) - User clicked a link
- `TYPED` (2) - User typed URL (**forensically significant**)
- `BOOKMARK` (3) - From bookmark (**indicates user intent**)
- `EMBED` (4) - Embedded content (iframe, object)
- `REDIRECT_PERMANENT` (5) - HTTP 301 permanent redirect
- `REDIRECT_TEMPORARY` (6) - HTTP 302/307 temporary redirect
- `DOWNLOAD` (7) - Download activity (**forensically significant**)
- `FRAMED_LINK` (8) - Link within iframe
- `RELOAD` (9) - Page reload

#### Chromium visit_source Values

Based on [Chromium's VisitSource enum](https://source.chromium.org/chromium/chromium/src/+/main:components/history/core/browser/history_types.h):

- `synced` (0) - Visit from Chrome Sync (other devices)
- `browsed` (1) - Local browsing activity (default)
- `extension` (2) - Created by browser extension
- `firefox_imported` (3) - Imported from Firefox
- `ie_imported` (4) - Imported from Internet Explorer
- `safari_imported` (5) - Imported from Safari

#### Firefox visit_source Values

Based on Firefox Places database schema:

- `source_organic` (0) - Normal browsing/navigation (default)
- `source_imported` (1) - Imported from another browser
- `source_synced` (2) - Firefox Sync from other devices
- `source_temporary` (3) - Temporary/private browsing artifacts

#### Safari Fields

Safari uses different field names:
- No explicit `transition_type` equivalent
- No explicit `visit_source` field
- Uses `sf_domain_expansion` for domain classification
- Uses `sf_load_successful` for page load status

### Data Normalization
- Handles browser-specific timestamp formats (WebKit microseconds, Unix epoch)
- Normalizes transition types across different browser terminologies
- Converts browser-specific visit tracking to unified schema
- Empty/null values for browser-specific fields when not applicable
