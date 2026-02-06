The sql `query` metricset collects rows returned by a query.

Field names (columns) are returned as lowercase strings. Values are returned as numeric or string.

## Cursor-based Incremental Data Fetching

The cursor feature enables incremental data fetching by tracking the last fetched row value
and using it to retrieve only new data on subsequent collection cycles. This is particularly
useful for:

- Fetching audit logs or events that are continuously appended
- Reducing database load by avoiding full table scans
- Preventing duplicate data ingestion

### Configuration

To enable cursor-based fetching, add a `cursor` configuration block to your metricset:

```yaml
- module: sql
  metricsets: [query]
  hosts: ["postgres://user:pass@localhost:5432/mydb"]
  driver: postgres
  sql_query: "SELECT id, event_data, created_at FROM events WHERE id > :cursor ORDER BY id ASC LIMIT 1000"
  sql_response_format: table
  raw_data.enabled: true
  cursor:
    enabled: true
    column: id
    type: integer
    default: "0"
```

### Cursor Configuration Options

| Option | Required | Description |
|--------|----------|-------------|
| `cursor.enabled` | No | Set to `true` to enable cursor-based fetching. Default: `false` |
| `cursor.column` | Yes (when enabled) | The column name to track for cursor state. Must be present in query results. |
| `cursor.type` | Yes (when enabled) | Data type of the cursor column: `integer`, `timestamp`, `date`, `float`, or `decimal` |
| `cursor.default` | Yes (when enabled) | Initial cursor value used on first run (before any state is persisted) |
| `cursor.direction` | No | Scan direction: `asc` (default, tracks max value) or `desc` (tracks min value) |

### Supported Cursor Types

| Type | Description | Default Format Example |
|------|-------------|----------------------|
| `integer` | Integer values (auto-incrementing IDs, sequence numbers) | `"0"` |
| `timestamp` | Timestamp with timezone (RFC3339 format) | `"2024-01-01T00:00:00Z"` |
| `date` | Date values (YYYY-MM-DD format) | `"2024-01-01"` |
| `float` | Floating-point values (FLOAT, DOUBLE, REAL). IEEE 754 precision limits apply. | `"0.0"` |
| `decimal` | Exact decimal values (DECIMAL, NUMERIC). Arbitrary precision, no data loss. | `"0.00"` |

### Scan Direction

By default, the cursor tracks the maximum value from each batch (ascending scan). For descending scans, set `cursor.direction: desc`:

| Direction | Operator | ORDER BY | Cursor Tracks |
|-----------|----------|----------|---------------|
| `asc` (default) | `>` | `ASC` | Maximum value |
| `desc` | `<` | `DESC` | Minimum value |

### Query Requirements

When cursor is enabled, your SQL query must:

1. **Include the `:cursor` placeholder** exactly once in the query WHERE clause
2. **Include an ORDER BY clause** on the cursor column matching the configured direction
3. **Use `sql_response_format: table`** - cursor requires table mode

### Example Configurations

#### Integer cursor (auto-increment ID)

```yaml
- module: sql
  metricsets: [query]
  hosts: ["mysql://user:pass@localhost:3306/mydb"]
  driver: mysql
  sql_query: "SELECT id, event_type, payload FROM audit_log WHERE id > :cursor ORDER BY id ASC LIMIT 500"
  sql_response_format: table
  raw_data.enabled: true
  cursor:
    enabled: true
    column: id
    type: integer
    default: "0"
```

#### Timestamp cursor (event timestamps)

```yaml
- module: sql
  metricsets: [query]
  hosts: ["postgres://user:pass@localhost:5432/mydb"]
  driver: postgres
  sql_query: "SELECT id, message, created_at FROM logs WHERE created_at > :cursor ORDER BY created_at ASC LIMIT 500"
  sql_response_format: table
  raw_data.enabled: true
  cursor:
    enabled: true
    column: created_at
    type: timestamp
    default: "2024-01-01T00:00:00Z"
```

#### Date cursor (daily partitioned data)

```yaml
- module: sql
  metricsets: [query]
  hosts: ["oracle://user:pass@localhost:1521/MYDB"]
  driver: oracle
  sql_query: "SELECT report_date, metrics FROM daily_reports WHERE report_date > :cursor ORDER BY report_date ASC"
  sql_response_format: table
  raw_data.enabled: true
  cursor:
    enabled: true
    column: report_date
    type: date
    default: "2024-01-01"
```

#### Decimal cursor (exact numeric, financial data)

```yaml
- module: sql
  metricsets: [query]
  hosts: ["postgres://user:pass@localhost:5432/mydb"]
  driver: postgres
  sql_query: "SELECT id, amount, description FROM transactions WHERE amount > :cursor ORDER BY amount ASC LIMIT 500"
  sql_response_format: table
  raw_data.enabled: true
  cursor:
    enabled: true
    column: amount
    type: decimal
    default: "0.00"
```

#### Float cursor (approximate numeric)

```yaml
- module: sql
  metricsets: [query]
  hosts: ["mysql://user:pass@localhost:3306/mydb"]
  driver: mysql
  sql_query: "SELECT id, score FROM scores WHERE score > :cursor ORDER BY score ASC LIMIT 500"
  sql_response_format: table
  raw_data.enabled: true
  cursor:
    enabled: true
    column: score
    type: float
    default: "0.0"
```

Note: Float cursors use IEEE 754 `float64` representation. For exact precision at boundaries (for example, financial data), use the `decimal` type instead.

#### Descending scan (processing historical data backwards)

```yaml
- module: sql
  metricsets: [query]
  hosts: ["postgres://user:pass@localhost:5432/mydb"]
  driver: postgres
  sql_query: "SELECT id, event_data FROM events WHERE id < :cursor ORDER BY id DESC LIMIT 500"
  sql_response_format: table
  raw_data.enabled: true
  cursor:
    enabled: true
    column: id
    type: integer
    default: "999999999"
    direction: desc
```

With `direction: desc`, the cursor tracks the minimum value from each batch, suitable for scanning data in reverse chronological order.

### State Persistence

Cursor state is persisted to disk using Metricbeat's statestore at:
`{data.path}/sql-cursor/`

The state persists across Metricbeat restarts, allowing incremental fetching to continue
from where it left off. State is keyed by a hash of:
- Module ID
- Full database URI/DSN (includes database name)
- Query string
- Cursor column name

This ensures that different query configurations maintain separate cursor states, including
different databases on the same server.

### Important: Choosing `>` vs `>=` for Cursor Queries

The choice of comparison operator in your WHERE clause affects data completeness:

**Use `>` (greater than) when:**
- The cursor column has unique, monotonically increasing values (auto-increment IDs, sequences)
- No two rows can share the same cursor value
- Example: `WHERE id > :cursor ORDER BY id ASC`

**Use `>=` (greater than or equal) when:**
- The cursor column may have duplicate values (timestamps, dates, scores)
- Late-arriving rows may be inserted with the same value as the current cursor
- Example: `WHERE created_at >= :cursor ORDER BY created_at ASC`

The `>=` operator causes the last row from each batch to be re-fetched on the next cycle (a duplicate), but ensures no data is lost when multiple rows share the same cursor value. If using `>=`, configure Elasticsearch document IDs or use an ingest pipeline to deduplicate.

```yaml
# Safe for timestamps -- accepts duplicates, prevents data loss
sql_query: "SELECT id, data, created_at FROM events WHERE created_at >= :cursor ORDER BY created_at ASC LIMIT 500"

# Safe for unique IDs -- no duplicates possible
sql_query: "SELECT id, data FROM events WHERE id > :cursor ORDER BY id ASC LIMIT 500"
```

### Error Handling

The cursor feature follows an "at-least-once" delivery model:
- Events are emitted **before** the cursor state is updated
- If a failure occurs after emitting events but before updating state, those events will be
  re-fetched on the next cycle
- This ensures no data loss, but may result in occasional duplicates

### Driver-specific Notes

**MySQL:** When using timestamp cursors, include `parseTime=true` in your DSN to ensure the driver
correctly handles `time.Time` parameters:
```
hosts: ["root:pass@tcp(localhost:3306)/mydb?parseTime=true"]
```

**Oracle:** Set the session timezone to UTC for timestamp cursors. The `godror` driver may convert
Go UTC timestamps to the Oracle session timezone, causing incorrect comparisons. Use the
`alterSession` DSN parameter or consult the Oracle integration documentation.

**Decimal columns:** The `decimal` cursor type passes the cursor value as a string to the database
driver. Most drivers (PostgreSQL, MySQL, MSSQL) implicitly cast strings to DECIMAL for comparison.
If your driver doesn't, use an explicit cast: `WHERE price > CAST(:cursor AS DECIMAL(10,2))`.

### Limitations

- Only one `:cursor` placeholder is allowed per query
- The cursor column **must** be included in the SELECT clause. If omitted, the cursor will not
  advance and an error will be logged on the first fetch
- NULL cursor values are skipped (only non-NULL values contribute to cursor progression)
- String, UUID, and ULID columns are not supported as cursor types. Workaround: add an integer
  or timestamp column for cursor tracking, or use a database function to convert to a sortable value
- All matching rows are loaded into memory before events are emitted. Use LIMIT to control memory
  usage (recommended: 500-5000 rows per cycle). For wide rows with large text columns, use a lower LIMIT
- Float cursors are subject to IEEE 754 precision limits. For exact boundary comparisons
  (for example, financial data), use the `decimal` type instead
- If a collection cycle takes longer than the configured `period`, subsequent cycles are skipped
  until the current one completes
