// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package tablespec

import (
	"encoding/json"
	"fmt"

	"github.com/osquery/osquery-go/plugin/table"

	"github.com/elastic/beats/v7/libbeat/version"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)

func getReadmeURL() string {
	return fmt.Sprintf("https://github.com/elastic/beats/blob/v%s/x-pack/osquerybeat/ext/osquery-extension/README.md", version.GetDefaultVersion())
}

// TableSpec implements the TableSpec interface for the elastic_browser_history table
type TableSpec struct {
	name        string
	desc        string
	platforms   []string
	fullcolumns []FullColumnDefinition
	columns     []table.ColumnDefinition
	generate    func(*logger.Logger) table.GenerateFunc
}

type FullColumnDefinition struct {
	table.ColumnDefinition
	Description string
}

// NewTableSpec creates a new TableSpec instance
func NewTableSpec(name, desc string, platforms []string, fullcolumns []FullColumnDefinition, genfn func(*logger.Logger) table.GenerateFunc) *TableSpec {
	columns := extractColumnDefinitions(fullcolumns)
	return &TableSpec{
		name:        name,
		desc:        desc,
		platforms:   platforms,
		fullcolumns: fullcolumns,
		columns:     columns,
		generate:    genfn,
	}
}

type columnJSON struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Hidden      bool   `json:"hidden"`
	Required    bool   `json:"required"`
	Index       bool   `json:"index"`
}

type tableJSON struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	URL         string       `json:"url"`
	Platforms   []string     `json:"platforms"`
	Evented     bool         `json:"evented"`
	Cacheable   bool         `json:"cacheable"`
	Columns     []columnJSON `json:"columns"`
}

func (s *TableSpec) MarshalJSON() ([]byte, error) {
	columns := make([]columnJSON, len(s.fullcolumns))
	for i, col := range s.fullcolumns {
		columns[i] = columnJSON{
			Name:        col.Name,
			Description: col.Description,
			Type:        string(col.Type),
			Hidden:      false, // Default value, can be enhanced if needed
			Required:    false, // Default value, can be enhanced if needed
			Index:       false, // Default value, can be enhanced if needed
		}
	}

	t := tableJSON{
		Name:        s.name,
		Description: s.desc,
		URL:         getReadmeURL(),
		Platforms:   s.platforms,
		Evented:     false, // Default value, can be enhanced if needed
		Cacheable:   false, // Default value, can be enhanced if needed
		Columns:     columns,
	}

	return json.Marshal(t)
}

// Name returns the table name
func (s TableSpec) Name() string {
	return s.name
}

// Description returns a brief description of the table
func (s TableSpec) Description() string {
	return s.desc
}

// Platforms returns the list of supported platforms
func (s TableSpec) Platforms() []string {
	return s.platforms
}

// Columns returns the column definitions for this table
func (s TableSpec) Columns() []table.ColumnDefinition {
	return s.columns
}

// Generate returns the generate function for this table
func (s TableSpec) Generate(log *logger.Logger) table.GenerateFunc {
	return s.generate(log)
}

func extractColumnDefinitions(fullcols []FullColumnDefinition) []table.ColumnDefinition {
	cols := make([]table.ColumnDefinition, len(fullcols))
	for i, col := range fullcols {
		cols[i] = col.ColumnDefinition
	}
	return cols
}
