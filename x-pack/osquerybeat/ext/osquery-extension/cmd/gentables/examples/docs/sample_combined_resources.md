% This file is generated! See ext/osquery-extension/cmd/gentables.

# sample_combined_resources

View combining active and archived resources with calculated fields and UNION

## Platforms

- ✅ Linux
- ✅ macOS
- ✅ Windows

## Description

This view demonstrates advanced SQL features including:
- UNION ALL for combining multiple query results
- CASE expressions with explicit aliases for calculated fields
- Date calculations using strftime for age and modification tracking
- Complex WHERE conditions with multiple criteria
- Arithmetic expressions with proper aliasing

The view combines active enabled resources with recently archived resources
(archived within the last 30 days), providing a unified view of current and
recent resources with human-readable priority levels and calculated age fields.

## Schema

| Column | Type | Description |
|--------|------|-------------|
| `id` | `BIGINT` | Unique identifier |
| `name` | `TEXT` | Resource name |
| `status` | `TEXT` | Current status |
| `category` | `TEXT` | Resource category |
| `priority_level` | `TEXT` | Human-readable priority level |
| `age_days` | `INTEGER` | Age in days since creation |
| `last_modified_days` | `INTEGER` | Days since last modification |
| `value` | `DOUBLE` | Associated numeric value |

## Required Tables

This view requires the following tables to be available:
- `sample_custom_table`

## View Definition

```sql
CREATE VIEW sample_combined_resources AS
SELECT 
  id,
  name,
  status,
  category,
  CASE priority
    WHEN 4 THEN 'critical'
    WHEN 3 THEN 'high'
    WHEN 2 THEN 'medium'
    ELSE 'low'
  END AS priority_level,
  (strftime('%s', 'now') - created_time) / 86400 AS age_days,
  (strftime('%s', 'now') - updated_time) / 86400 AS last_modified_days,
  value
FROM sample_custom_table
WHERE status = 'active' AND enabled = 1

UNION ALL

SELECT 
  id,
  name,
  status,
  category,
  CASE priority
    WHEN 4 THEN 'critical'
    WHEN 3 THEN 'high'
    WHEN 2 THEN 'medium'
    ELSE 'low'
  END AS priority_level,
  (strftime('%s', 'now') - created_time) / 86400 AS age_days,
  (strftime('%s', 'now') - updated_time) / 86400 AS last_modified_days,
  value
FROM sample_custom_table
WHERE status = 'archived' AND updated_time > (strftime('%s', 'now') - 2592000);
```

## Examples
### Find critical resources requiring attention

```sql
SELECT name, category, priority_level, last_modified_days
FROM sample_combined_resources
WHERE priority_level = 'critical'
  AND last_modified_days > 7
ORDER BY last_modified_days DESC;
```
### Analyze resource distribution by category and priority

```sql
SELECT category, priority_level, COUNT(*) as count, AVG(value) as avg_value
FROM sample_combined_resources
GROUP BY category, priority_level
ORDER BY category, priority_level;
```
### Identify aging resources by status

```sql
SELECT status, 
       COUNT(*) as total,
       AVG(age_days) as avg_age,
       MAX(age_days) as oldest
FROM sample_combined_resources
GROUP BY status;
```

## Notes
- This view uses UNION ALL to combine active and recently archived resources
- CASE expressions require explicit AS aliases for column names
- All date calculations are performed using SQLite date functions
- The view automatically updates when the underlying table changes
- Archived resources older than 30 days are excluded to keep results focused
- Priority levels are converted to human-readable text for easier interpretation

## Related Tables
- `sample_custom_table`
- `sample_resource_history`
