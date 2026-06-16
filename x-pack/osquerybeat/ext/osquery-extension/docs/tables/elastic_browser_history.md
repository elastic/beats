% This file is generated! See ext/osquery-extension/cmd/gentables.

# elastic_browser_history

Browser history from multiple browsers (Chrome, Edge, Firefox, Safari) with unified schema

## Platforms

- ✅ Linux
- ✅ macOS
- ✅ Windows

## Description

Query browser history from multiple browsers with a unified schema. Supports Chrome,
Edge, Firefox, and Safari across Linux, macOS, and Windows. The table auto-discovers
browsers from standard user profile locations. Use the custom_data_dir constraint to
query history from non-standard paths (forensics, backups, or mounted drives).

**Supported browsers:** Chrome, Edge, Firefox on all platforms; Safari on macOS only.
**Auto-discovery paths:** Linux: ~/.config/google-chrome/, microsoft-edge/, ~/.mozilla/firefox/.
macOS: ~/Library/Application Support/ (Google/Chrome/, Microsoft Edge/, Firefox/, Safari/).
Windows: %LOCALAPPDATA% (Google\\Chrome\\User Data\\, Microsoft\\Edge\\User Data\\), %APPDATA%\\Mozilla\\Firefox\\.

**Data model:** One row per visit. The url_id column is unique per URL within a browser
profile; use url_id (with browser and profile_name when grouping across profiles) for
accurate visit counts. transition_type is how the user navigated (TYPED, LINK, BOOKMARK,
etc.); visit_source is where the data came from (browsed, synced, imported, extension).
Chromium-specific fields (ch_*), Firefox (ff_*), and Safari (sf_*) are only populated
for that browser. Database sources: Chrome/Edge/Brave use History (SQLite), Firefox uses
places.sqlite, Safari uses History.db.

## Schema

| Column | Type | Description |
|--------|------|-------------|
| `timestamp` | `BIGINT` | Unix timestamp in seconds (visit time) |
| `datetime` | `TEXT` | Human-readable datetime in RFC3339 format (visit time) |
| `url_id` | `BIGINT` | Unique URL identifier |
| `scheme` | `TEXT` | URL scheme or protocol (https, http, file, ftp) |
| `domain` | `TEXT` | Registrable domain (eTLD+1) from hostname |
| `hostname` | `TEXT` | Full hostname from URL |
| `url` | `TEXT` | Full URL visited |
| `title` | `TEXT` | Page title |
| `browser` | `TEXT` | Browser name (chrome, edge, firefox, safari, brave) |
| `parser` | `TEXT` | Parser used to extract the history |
| `user` | `TEXT` | Username or profile owner |
| `profile_name` | `TEXT` | Browser profile name |
| `transition_type` | `TEXT` | Navigation method (TYPED, LINK, BOOKMARK, RELOAD, REDIRECT, etc.) |
| `referring_url` | `TEXT` | URL that linked to this page when navigated via link |
| `visit_id` | `BIGINT` | Unique visit identifier |
| `from_visit_id` | `BIGINT` | Visit ID that led to this visit (navigation chain) |
| `visit_source` | `TEXT` | Data origin (browsed/local, synced, imported, extension) |
| `is_hidden` | `INTEGER` | Whether visit is hidden (1) or visible (0) |
| `history_path` | `TEXT` | Path to the browser history database file |
| `ch_visit_duration_ms` | `BIGINT` | Duration of visit in milliseconds (Chromium only) |
| `ff_session_id` | `INTEGER` | Firefox session tracking identifier |
| `ff_frecency` | `INTEGER` | Firefox frecency score (frequency and recency) |
| `sf_domain_expansion` | `TEXT` | Safari domain classification or expansion |
| `sf_load_successful` | `INTEGER` | Whether page loaded successfully (1) or failed (0) (Safari) |
| `custom_data_dir` | `TEXT` | Custom data directory path for non-standard locations |

## Examples
### Get all history from all discovered browsers

```sql
SELECT * FROM elastic_browser_history;
```
### Get recent history (last 7 days)

```sql
SELECT url, title, browser, datetime
FROM elastic_browser_history
WHERE timestamp > (strftime('%s', 'now') - 604800)
ORDER BY timestamp DESC;
```
### Filter by browser and profile

```sql
SELECT browser, profile_name, url, title, datetime
FROM elastic_browser_history
WHERE browser = 'firefox' AND profile_name = 'default-release'
ORDER BY timestamp DESC;
```
### Filter by browser and domain

```sql
SELECT browser, profile_name, url, title, datetime
FROM elastic_browser_history
WHERE browser = 'chrome' AND domain = 'github.com'
ORDER BY timestamp DESC;
```
### Query from custom data directory

```sql
SELECT * FROM elastic_browser_history
WHERE custom_data_dir = '/mnt/backup/Users/john/AppData/Local/Google';
```
### Custom data directory with GLOB

```sql
SELECT * FROM elastic_browser_history
WHERE custom_data_dir GLOB '/forensics/users/*/Library/Application Support/Google';
```
### Search by domain and hostname

```sql
SELECT browser, domain, hostname, url, title
FROM elastic_browser_history
WHERE domain LIKE '%.google.com'
ORDER BY timestamp DESC;
```
### Find non-HTTPS visits

```sql
SELECT browser, url, title, datetime
FROM elastic_browser_history
WHERE scheme IN ('http', 'ftp')
ORDER BY timestamp DESC;
```
### Most visited URLs (group by url_id for consistency)

```sql
SELECT url, domain, COUNT(*) as visit_count
FROM elastic_browser_history
GROUP BY url_id
ORDER BY visit_count DESC
LIMIT 20;
```
### Most visited domains

```sql
SELECT domain, COUNT(*) as visits
FROM elastic_browser_history
WHERE domain != ''
GROUP BY domain
ORDER BY visits DESC
LIMIT 20;
```
### Most visited hostnames

```sql
SELECT hostname, COUNT(*) as visits
FROM elastic_browser_history
WHERE hostname != ''
GROUP BY hostname
ORDER BY visits DESC
LIMIT 20;
```
### Find typed URLs (direct navigation)

```sql
SELECT url, title, browser, datetime
FROM elastic_browser_history
WHERE transition_type = 'TYPED'
ORDER BY timestamp DESC;
```
### Find hidden entries

```sql
SELECT url, title, browser, datetime
FROM elastic_browser_history
WHERE is_hidden = 1;
```
### Filter by visit source (e.g. synced from other devices)

```sql
SELECT url, title, browser, visit_source
FROM elastic_browser_history
WHERE visit_source = 'synced'
ORDER BY timestamp DESC;
```
### Correct aggregation across profiles (include browser and profile_name)

```sql
SELECT browser, profile_name, url, COUNT(*) as visits
FROM elastic_browser_history
GROUP BY browser, profile_name, url_id
ORDER BY visits DESC
LIMIT 20;
```

## Notes
- Chrome, Edge, and Brave use Chromium parsers; Safari is macOS only. Universal columns are always populated; ch_*, ff_*, and sf_* only for that browser.
- custom_data_dir must point to the browser base directory (not the profile directory); the table discovers all profiles under that path.
- Use url_id for grouping visits to the same URL; url_id is only consistent within the same profile, so include browser and profile_name when grouping across profiles.
- transition_type is how the user navigated (TYPED, LINK, BOOKMARK, RELOAD, etc.); visit_source is where the data came from (browsed, synced, imported, extension).
- Performance: use timestamp and browser/profile filters to limit results; custom_data_dir queries can be slower due to discovery. History databases can be large.
- Security: requires read access to browser profile directories; data may be sensitive; consider retention and privacy regulations (e.g. GDPR, CCPA).
- Troubleshooting: if no results, check profile paths and read permissions; close the browser to avoid database locked errors, or query copies via custom_data_dir.

## Related Tables
- `users`
- `chrome_extensions`
- `firefox_addons`
- `safari_extensions`
