// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cursor

import (
	"errors"
	"fmt"
	"strings"
)

// Cursor type constants
const (
	// CursorTypeInteger is the cursor type for integer columns (auto-increment IDs, etc.)
	CursorTypeInteger = "integer"

	// CursorTypeTimestamp is the cursor type for timestamp columns
	CursorTypeTimestamp = "timestamp"

	// CursorTypeDate is the cursor type for date columns
	CursorTypeDate = "date"

	// CursorTypeFloat is the cursor type for floating-point columns (FLOAT, DOUBLE, REAL).
	// Uses Go float64 internally. Subject to IEEE 754 precision limits —
	// boundary rows may be duplicated or skipped at the 15th+ significant digit.
	CursorTypeFloat = "float"

	// CursorTypeDecimal is the cursor type for exact decimal columns (DECIMAL, NUMERIC).
	// Uses shopspring/decimal for arbitrary-precision arithmetic. No data loss at boundaries.
	CursorTypeDecimal = "decimal"
)

// Cursor direction constants
const (
	// CursorDirectionAsc tracks the maximum cursor value (for ascending ORDER BY).
	CursorDirectionAsc = "asc"

	// CursorDirectionDesc tracks the minimum cursor value (for descending ORDER BY).
	CursorDirectionDesc = "desc"
)

// supportedCursorTypes lists all valid cursor types.
var supportedCursorTypes = []string{
	CursorTypeInteger,
	CursorTypeTimestamp,
	CursorTypeDate,
	CursorTypeFloat,
	CursorTypeDecimal,
}

// Config holds the cursor configuration from user's metricbeat.yml
type Config struct {
	Enabled   bool   `config:"enabled"`
	Column    string `config:"column"`
	Type      string `config:"type"`      // "integer", "timestamp", "date", "float", or "decimal"
	Default   string `config:"default"`   // Initial cursor value as string
	Direction string `config:"direction"` // "asc" (default) or "desc"
}

// Validate checks the configuration for errors.
// If cursor is disabled, no validation is performed.
// If cursor is enabled, all fields are required and must be valid.
func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.Column == "" {
		return errors.New("cursor.column is required when cursor is enabled")
	}

	if !isValidCursorType(c.Type) {
		return fmt.Errorf("cursor.type must be one of [%s], got %q",
			strings.Join(supportedCursorTypes, ", "), c.Type)
	}

	if c.Default == "" {
		return errors.New("cursor.default is required when cursor is enabled")
	}

	// Validate default value is parseable as the declared type
	if _, err := ParseValue(c.Default, c.Type); err != nil {
		return fmt.Errorf("cursor.default is invalid for type %q: %w", c.Type, err)
	}

	// Validate direction (defaults to "asc" if empty)
	if c.Direction == "" {
		c.Direction = CursorDirectionAsc
	}
	if !isValidDirection(c.Direction) {
		return fmt.Errorf("cursor.direction must be '%s' or '%s', got %q",
			CursorDirectionAsc, CursorDirectionDesc, c.Direction)
	}

	return nil
}

// isValidCursorType checks if the given type is a supported cursor type
func isValidCursorType(t string) bool {
	switch t {
	case CursorTypeInteger, CursorTypeTimestamp, CursorTypeDate, CursorTypeFloat, CursorTypeDecimal:
		return true
	default:
		return false
	}
}

// isValidDirection checks if the given direction is valid
func isValidDirection(d string) bool {
	switch d {
	case CursorDirectionAsc, CursorDirectionDesc:
		return true
	default:
		return false
	}
}
