// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcpbigquery

import (
	"fmt"
	"time"

	"github.com/elastic/go-ucfg"
)

var _ ucfg.Validator = (*config)(nil)

type config struct {
	// Period specifies how often to run the input.
	Period time.Duration `config:"period" validate:"required,positive"`

	// Project ID is the ID of the GCP project that owns the BigQuery dataset.
	ProjectID string `config:"project_id" validate:"required"`

	// CredentialsFile specifies a JSON file containing authentication credentials and key.
	CredentialsFile string `config:"credentials_file"`

	// Queries contains the BigQuery queries to execute, along with options for each.
	Queries []queryConfig `config:"queries" validate:"required"`
}

type queryConfig struct {
	// Query is the SQL query to execute.
	Query string `config:"query" validate:"required"`

	// Cursor configures how state is tracked between queries to simulate cursor behavior.
	Cursor *cursorConfig `config:"cursor"`

	// TimestampField is a field of type TIMESTAMP in the query result to use as the event's @timestamp value.
	TimestampField string `config:"timestamp_field"`

	// IdFields is a list of fields in the query result to use to generate a deterministic ID.
	// This can be useful to avoid duplication when queries may return overlapping results.
	// Note: the ID is only generated if _all_ specified fields are present in the query result.
	IdFields []string `config:"id_fields"`

	// ExpandJsonStrings determines whether to attempt to parse fields of type JSON into objects/arrays instead of
	// leaving them as strings. In the event of parsing failures, we still expand the field into a JSON object with a
	// single field named "original" containing the original string value; this avoids mapping conflicts in
	// Elasticsearch. Defaults to true.
	ExpandJsonStrings *bool `config:"expand_json_strings"`
}

type cursorConfig struct {
	// Field specifies a field in the query result to use for tracking state between queries.
	// This field must be of a type that supports ordering comparisons - the following types are supported:
	// BIGNUMERIC, BYTES, DATE, DATETIME, FLOAT, INTEGER, NUMERIC, STRING, TIME, TIMESTAMP.
	Field string `config:"field"`

	// InitialValue is the starting value for the cursor parameter when there is no previous state.
	// Can be a literal BigQuery value or an expression which returns a single value of the appropriate type.
	// e.g.	"2025-10-01", "123.456789012", "TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 1 DAY)".
	InitialValue string `config:"initial_value"`
}

func defaultConfig() config {
	return config{
		Period: time.Minute,
	}
}

func (c *config) Validate() error {
	for i, query := range c.Queries {
		if query.Cursor != nil && query.Cursor.Field == "" {
			return fmt.Errorf("queries[%d]: cursor field is required", i)
		}
	}
	return nil
}
