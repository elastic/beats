# gentables - Table and View Code Generator

This tool generates Go code and documentation from YAML specifications for osquery tables and views.

## Unified Specification Format

Tables and views are now defined using a unified YAML format in a single directory, differentiated by the `type` field:
- `type: table` - Defines an osquery table
- `type: view` - Defines an osquery view

This simplifies organization and allows both tables and views to be processed in a single pass.

## Isolated Module

This tool has its own `go.mod` to isolate its dependencies from the main osquerybeat module. This keeps the pingcap SQL parser as a build-time-only dependency without adding it to the production code.

## Running the Tool

From the `cmd/gentables` directory:

```bash
go run . -spec-dir <specs_dir> -out-dir <tables_output> -views-out-dir <views_output> -verbose
```

Or via `go generate` from the osquery-extension directory:

```bash
go generate ./...
```

## Dependencies

- `gopkg.in/yaml.v3` - YAML parsing
- `github.com/pingcap/parser` - SQL parser for view validation (isolated from main module)

## Features

- **Unified Spec Format**: Single directory with both tables and views
- **Embedded Templates**: Generator templates are embedded via `embed`
- **Result Types**: Generates typed structs with osquery tags for query results
- **time.Time Support**: Use `go_type: time.Time` for proper timestamp handling
- **SQL Validation**: Uses AST-based parsing to validate view queries without a database
- **Column Validation**: Ensures view columns match SELECT output
- **Sensible Defaults**: Automatically applies defaults for common fields
- **Comprehensive Documentation**: Generates Markdown docs for all tables and views

## Shared Types Across Tables

For types that need to be consistent across multiple tables (e.g., `UserProfile`, `LnkMetadata`), 
create a shared types spec in your specs directory:

```yaml
# shared_types.yaml
type: shared_types
group: my_group
types:
  - name: UserProfile
    description: Windows user profile information
    pointer: true
    columns:
      - name: username
        type: TEXT
        description: The username
      - name: sid
        type: TEXT
        description: The Windows Security Identifier

  - name: LnkMetadata
    description: Windows LNK file metadata
    pointer: true
    columns:
      - name: local_path
        type: TEXT
        description: Target local path
      - name: file_size
        type: INTEGER
        description: Target file size
```

Then reference these shared types in your table specs:

```yaml
# sample_jumplists.yaml
type: table
name: sample_jumplists
shared_types:
  - UserProfile
  - LnkMetadata
columns:
  - name: UserProfile
    type: EMBEDDED
    embedded_type: UserProfile
  # ...

# sample_recent_files.yaml
type: table
name: sample_recent_files
shared_types:
  - UserProfile      # Same type definition as jumplists
  - LnkMetadata      # Same type definition as jumplists
columns:
  - name: UserProfile
    type: EMBEDDED
    embedded_type: UserProfile
  # ...
```

This ensures both tables have identical column definitions for shared fields.
See `examples/tables/shared_types.yaml`, `examples/tables/sample_jumplists.yaml`, and 
`examples/tables/sample_recent_files.yaml` for complete examples.

You may define multiple `type: shared_types` spec files. Each shared types file
must set a `group`, and type names must be unique within that group. Specs that
use `shared_types` must also set the same `group`.

Shared types are generated under the tables output directory:
`<tables_out_dir>/<group>/types.go` with package name derived from the group.

## Templates

Templates are stored under `cmd/gentables/templates/` and embedded into the
binary via `embed`. The generator loads them from the embedded filesystem at
runtime, so you do not need to ship template files separately.

## Generated Output Structure

Each table/view is generated into its own package:

```
pkg/tables/
├── registry.go                      # STATIC - metadata registry for all tables
└── generated/
    └── sample_custom_table/             # Directory (descriptive)
        └── sample_custom_table.go       # package samplecustomtable (idiomatic)

pkg/views/
├── registry.go                      # STATIC - metadata registry for all views
└── generated/
    └── sample_combined_resources/       # Directory (descriptive)
        └── sample_combined_resources.go # package samplecombinedresources (idiomatic)
```

**Package Naming:**
- Directory names preserve the original table/view name (e.g., `sample_custom_table`)
- Package names follow Go idioms - lowercase without underscores (e.g., `samplecustomtable`)

**Grouped Specs:**
If a spec includes `group: my_group`, generated code is placed under:
- Tables: `pkg/tables/generated/my_group/<table_name>/`
- Views: `pkg/views/generated/my_group/<view_name>/`
Shared types for that group are generated at:
`pkg/tables/generated/my_group/types.go`.

**Generated Code:**
```go
// Each package includes:
type Result struct { ... }           // Typed result struct with osquery tags
func Columns() []table.ColumnDefinition { ... }  // For tables
func View() *hooks.View { ... }      // For views
const TableName = "..."              // For tables
```

See the `examples/` directory for usage examples and detailed documentation.
