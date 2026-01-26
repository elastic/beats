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
- **Result Types**: Generates typed structs with osquery tags for query results
- **time.Time Support**: Use `go_type: time.Time` for proper timestamp handling
- **SQL Validation**: Uses AST-based parsing to validate view queries without a database
- **Column Validation**: Ensures view columns match SELECT output
- **Sensible Defaults**: Automatically applies defaults for common fields
- **Comprehensive Documentation**: Generates Markdown docs for all tables and views

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

**Generated Code:**
```go
// Each package includes:
type Result struct { ... }           // Typed result struct with osquery tags
func Columns() []table.ColumnDefinition { ... }  // For tables
func View() *hooks.View { ... }      // For views
const TableName = "..."              // For tables
```

See the `examples/` directory for usage examples and detailed documentation.
