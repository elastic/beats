# Generator Examples

This directory contains sample table and view specifications to demonstrate the gentables code generator.

## Sample Files

- `tables/sample_table.yaml` - Example table specification with multiple columns and comprehensive documentation
- `views/sample_view.yaml` - Example view specification demonstrating UNION ALL, CASE expressions, and date calculations

**Note:** The generator accepts both `.yaml` and `.yml` file extensions. You can use whichever you prefer, and both can coexist in the same directory.

## Testing the Generator

To test the generator with these samples, you have two options:

### Option 1: From the gentables directory

```bash
cd /path/to/cmd/gentables

go run . \
  -spec-dir examples/tables,examples/views \
  -out-dir examples/tables/generated \
  -docs-dir examples/docs \
  -views-out-dir examples/views/generated \
  -views-docs-dir examples/docs \
  -verbose
```

### Option 2: From the examples directory

```bash
cd /path/to/cmd/gentables/examples

go run ../main.go \
  -spec-dir tables,views \
  -out-dir tables/generated \
  -docs-dir tables/docs \
  -views-out-dir views/generated \
  -views-docs-dir views/docs \
  -verbose
```

### What Gets Generated

The generator will create:
- `tables/generated/` - Table code packages
  - `sample_custom_table/sample_custom_table.go` - Generated table code
  - `registry_*.go` - Platform-specific registries
- `views/generated/` - View code packages
  - `sample_combined_resources/sample_combined_resources.go` - Generated view code
  - `registry_*.go` - Platform-specific registries
- `tables/docs/` - Table documentation
  - `sample_custom_table.md` - Generated table documentation
- `views/docs/` - View documentation
  - `sample_combined_resources.md` - Generated view documentation

**Note**: Import paths in the generated platform files are calculated automatically based on the output directory locations and the module path detected from `go.mod`.

## Unified Spec Format

Tables and views use **identical YAML format**, differentiated only by the `type` field and the presence of the `query` field for views.

### Specification Format

```yaml
type: table|view                    # Required: "table" or "view"
name: spec_name                     # Required: table or view name
description: Brief description      # Required: brief description
platforms: [linux, darwin, windows] # Optional: defaults to all platforms
implementation_package: pkg/path    # Required for tables: import path of package that registers this table
group: my_group                     # Optional: scopes shared types for this spec

columns:                            # Required: column definitions
  - name: column_name               # Required: column name
    type: TEXT|INTEGER|BIGINT|DOUBLE # Required: osquery column type
    description: Column description # Required: column description
    go_type: string|int32|int64|float64|time.Time # Optional: override Go type
    format: unix|rfc3339            # Optional: format hint for struct tags
    timezone: UTC                   # Optional: timezone hint for struct tags

documentation:                      # Required: documentation
  description: Detailed description # Required: detailed description
  examples:                         # Required: at least one SQL query example
    - title: Example title
      query: SELECT * FROM spec_name;
  notes:                            # Required: at least one note
    - Note text
  related_tables:                   # Optional: defaults to empty list
    - other_table

# View-specific fields:
required_tables:                    # Optional: tables this view depends on
  - table_name

query: |                            # Required for views only
  SELECT column_name FROM table_name;
```

### Key Differences

- **Tables** (`type: table`): Must NOT have a `query` field
- **Views** (`type: view`): Must have a `query` field containing only the SELECT statement(s)

**Note**: The `query` field should contain only the SELECT statement(s). The tool automatically wraps it with `CREATE VIEW ... AS` in the generated code.

## Generated Output

The generator creates individual packages for each table/view with better encapsulation:

### Directory Structure
```
pkg/tables/
├── registry.go                    # STATIC - registry of all tables
└── generated/
    └── sample_custom_table/           # Directory: descriptive with underscores
        └── sample_custom_table.go     # Package: samplecustomtable (idiomatic)

pkg/views/
├── registry.go                    # STATIC - registry of all views
└── generated/
    └── sample_combined_resources/     # Directory: descriptive with underscores
        └── sample_combined_resources.go  # Package: samplecombinedresources (idiomatic)
```

**Package Naming Convention:**
- **Directory names**: Use the original table/view name with underscores (e.g., `sample_custom_table`)
- **Package names**: Idiomatic Go style - lowercase without underscores (e.g., `samplecustomtable`)

This follows Go best practices where package names should be short, lowercase, and without underscores, while directory names can remain descriptive.

### Generated Files
Each table/view package includes:
- `Result` struct with osquery tags
- `Columns()` function (tables) or `View()` function (views)
- `TableName` constant (tables only)

### Usage
```go
// Import using descriptive directory path, idiomatic package alias
import samplecustomtable "github.com/.../pkg/tables/generated/sample_custom_table"

// Access the types and functions
var result samplecustomtable.Result
columns := samplecustomtable.Columns()
name := samplecustomtable.TableName
```

**Registry registration is automatic via generated registry files:**
- Tables are registered in `pkg/tables/generated/registry.go`
- Views are registered in `pkg/views/generated/registry.go`
- No `init()` functions or manual registration needed

## Automatic Registration

### implementation_package Field (Tables Only, Required)

Every table must specify `implementation_package`: the import path of the Go package that registers the table (via `RegisterGenerateFunc()` in `init()`). This guarantees each table has a single, explicit registration point.

**Example 1 – dedicated implementation package** (implementation lives in a separate package):

```yaml
type: table
name: my_table
platforms: [linux, darwin, windows]

implementation_package: github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/myimpl

columns:
  - name: id
    type: BIGINT
    description: Unique identifier
  # ... more columns
```

**Example 2 – implementation in generated package** (implementation lives alongside generated code in the same package):

```yaml
type: table
name: sample_jumplists
group: jumplists

implementation_package: github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/tables/generated/jumplists/sample_jumplists

columns:
  # ...
```

**How it works:**

1. **Generator creates platform import files** – Imports `implementation_package` in `registry_linux.go`, etc.
2. **Package init() registers** – That package’s `init()` calls `RegisterGenerateFunc()`
3. **Static registry registers tables** – Generated `registry.go` calls all table registrations
4. **Main.go calls registry** – Calls `tables.RegisterTables()` and `views.RegisterViews()`

## Osquery Struct Tags

The generator automatically creates `Result` structs with osquery tags for proper serialization:

### Basic Tags
All columns get an osquery tag with their column name:
```go
type Result struct {
    Id   int64  `osquery:"id"`   // Basic column
    Name string `osquery:"name"` // Text column
}
```

### Format and Timezone Tags
Use the optional `format` and `timezone` fields in your column specs to add additional tags:

**YAML Spec:**
```yaml
columns:
  - name: created_time
    type: BIGINT
    description: Creation timestamp
    format: unix      # Adds format:"unix" tag
    timezone: UTC     # Adds tz:"UTC" tag
```

**Generated Go:**
```go
type Result struct {
    CreatedTime int64 `osquery:"created_time" format:"unix" tz:"UTC"`
}
```

### Supported Format Values
- `unix` - UNIX epoch timestamp (seconds since 1970-01-01)
- `rfc3339` - RFC3339 formatted timestamp (ISO 8601)

### Timezone Values
- `UTC` - Coordinated Universal Time
- Any IANA timezone name (e.g., `America/New_York`)

These tags are used by the osquery encoding package for proper serialization and deserialization of query results.

### Using time.Time in Result Structs

For timestamp fields, you can use `go_type: time.Time` to generate `time.Time` fields instead of int64/string:

**YAML Spec:**
```yaml
columns:
  - name: timestamp
    type: BIGINT
    description: Event timestamp
    go_type: time.Time    # Override default int64 with time.Time
    format: unix
    timezone: UTC
```

**Generated Go:**
```go
type Result struct {
    Timestamp time.Time `osquery:"timestamp" format:"unix" tz:"UTC"`
}
```

The encoding package automatically converts between `time.Time` and the appropriate format based on the tags.

## Column Documentation for Views

Both tables and views must document their columns in the spec. For views, this is especially important because:

1. **Clear Documentation** - Users can see what columns are available without analyzing the SQL
2. **Type Information** - Explicit type declarations help users understand data types
3. **Schema Validation** - Column specs document the expected schema of the view
4. **Better IDE Support** - Generated code includes column information in comments

The `columns` field must list all columns that the view returns, matching the SELECT statement in the query.

### Validation

The generator uses pingcap SQL parser to validate view specifications:

- **SQL Syntax Validation** - The SELECT statement must be syntactically valid SQL
- **Column Matching** - All specified columns must match the SELECT's output schema
- **Bidirectional Check** - Validates that specified columns exist in the output, and all output columns are documented
- **UNION Support** - Validates UNION and UNION ALL queries by extracting columns from the first SELECT
- **SELECT * Handling** - Views using `SELECT *` skip validation with a warning (assumes columns are correct)

**Alias Requirements:**

The following **do NOT require** an `AS` alias:
- Simple column references: `SELECT id, name FROM table`
- Qualified columns: `SELECT t.id, t.name FROM table t`
- Wildcards: `SELECT *` or `SELECT t.*`

The following **REQUIRE** an `AS` alias:
- Aggregate functions: `COUNT(*)` → `COUNT(*) AS total_count`
- Scalar functions: `UPPER(name)` → `UPPER(name) AS name_upper`
- CASE expressions: `CASE WHEN ... END` → `CASE WHEN ... END AS status_label`
- Arithmetic expressions: `value * 2` → `value * 2 AS doubled_value`
- Any complex expression that isn't a plain column reference

**How it works:**
1. Parses the SELECT statement into an Abstract Syntax Tree (AST)
2. Extracts output column names from the AST
3. Simple columns use their name; complex expressions require explicit aliases
4. Compares the extracted columns with the specified columns

**Advantages:**
- No database needed - purely static analysis
- Works for any valid SQL regardless of source table schemas
- Fast and reliable validation
- Forces explicit, intentional column naming for complex expressions

**Note:** 
- The `query` field should contain ONLY the SELECT statement
- The generator automatically wraps it in `CREATE VIEW view_name AS` when generating code
- This validates SQL syntax and output schema, but not source table column existence (verified at runtime by osquery)

## Defaults

The generator applies sensible defaults for commonly omitted fields:

- **`platforms`**: Defaults to `["linux", "darwin", "windows"]` if not specified
- **`documentation.related_tables`**: Defaults to empty array if not specified

This allows you to write minimal specs for cross-platform tables/views without repetitive boilerplate.

## Required vs Optional Fields Summary

### Required for ALL specs (tables and views):
- ✅ `type` - Must be "table" or "view"
- ✅ `name` - Spec name
- ✅ `description` - Brief description
- ✅ `columns` - At least one column with:
  - ✅ `name` - Column name
  - ✅ `type` - Column type (TEXT, INTEGER, BIGINT, DOUBLE, BOOLEAN)
  - ✅ `description` - Column description
- ✅ `documentation.description` - Detailed description
- ✅ `documentation.examples` - At least one example query
- ✅ `documentation.notes` - At least one note

### Required for TABLES only:
- ✅ `implementation_package` - Import path of the package that registers this table (via `RegisterGenerateFunc()` in `init()`)

### Optional for ALL specs:
- ⚪ `platforms` - Defaults to `["linux", "darwin", "windows"]`
- ⚪ `columns[].go_type` - Explicit Go type override (for example, "time.Time")
- ⚪ `columns[].format` - Format hint for osquery tags (for example, "unix", "rfc3339")
- ⚪ `columns[].timezone` - Timezone hint for osquery tags (for example, "UTC")
- ⚪ `documentation.related_tables` - Defaults to empty array
- ⚪ `required_tables` - Only applies to views; optional
- ⚪ `group` - Required if `shared_types` is set

### View-specific required:
- ✅ `query` - Must contain SELECT statement(s) only

### Table-specific rules:
- ❌ `query` - Must NOT be present

## Notes

These examples are for demonstration purposes only and are not processed by the actual build. They are kept separate from the real table/view specs in `../../specs/`.
