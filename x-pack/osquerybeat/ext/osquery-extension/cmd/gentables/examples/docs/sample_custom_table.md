% This file is generated! See ext/osquery-extension/cmd/gentables.

# sample_custom_table

Example table showing the generator capabilities with multiple data types

## Platforms

- ✅ Linux
- ✅ macOS
- ✅ Windows

## Description

This table demonstrates the code generator's capabilities with various data types
and comprehensive examples. It serves as a reference for creating new table
specifications and showcases proper documentation practices.

## Schema

| Column | Type | Description |
|--------|------|-------------|
| `id` | `BIGINT` | Unique identifier for the resource |
| `name` | `TEXT` | Resource name or label |
| `status` | `TEXT` | Current status (active, inactive, pending, archived) |
| `created_time` | `BIGINT` | Creation timestamp in UNIX epoch seconds |
| `updated_time` | `BIGINT` | Last update timestamp in UNIX epoch seconds |
| `value` | `DOUBLE` | Associated numeric value or metric |
| `enabled` | `INTEGER` | Whether the resource is enabled (1=yes, 0=no) |
| `category` | `TEXT` | Resource category (system, user, application) |
| `priority` | `INTEGER` | Priority level (1=low, 2=medium, 3=high, 4=critical) |

## Examples
### List all active resources sorted by priority

```sql
SELECT id, name, priority, created_time
FROM sample_custom_table
WHERE status = 'active'
ORDER BY priority DESC, created_time DESC;
```
### Count resources by status and category

```sql
SELECT status, category, COUNT(*) as count
FROM sample_custom_table
GROUP BY status, category
ORDER BY status, category;
```
### Find high-priority enabled resources updated recently

```sql
SELECT name, priority, value, updated_time
FROM sample_custom_table
WHERE enabled = 1 
  AND priority >= 3
  AND updated_time > (strftime('%s', 'now') - 86400)
ORDER BY priority DESC;
```
### Calculate average value by category for active resources

```sql
SELECT category, 
       COUNT(*) as total_count,
       AVG(value) as avg_value,
       MAX(value) as max_value
FROM sample_custom_table
WHERE status = 'active'
GROUP BY category;
```

## Notes
- This is a sample table to demonstrate the code generator's capabilities
- The generator creates Go code with column definitions and table metadata
- Platform-specific build tags are automatically added based on the platforms field
- All timestamps are stored as UNIX epoch seconds for cross-platform compatibility
- Status values should be validated at query time using WHERE clauses

## Related Tables
- `sample_resource_history`
- `sample_resource_tags`
- `system_info`
