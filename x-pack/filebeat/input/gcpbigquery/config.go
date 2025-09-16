package gcpbigquery

import (
	"time"

	"github.com/elastic/go-ucfg"
)

var _ ucfg.Validator = (*config)(nil)

type config struct {
	Period time.Duration `config:"period" validate:"required,positive"`

	// The ID of the GCP project that owns the BigQuery dataset.
	ProjectID string `config:"project_id" validate:"required"`

	// JSON file containing authentication credentials and key.
	CredentialsFile string `config:"credentials_file"`

	// The BigQuery queries to execute, along with options for each.
	Queries []queryConfig `config:"queries" validate:"required"`
}

type queryConfig struct {
	// The SQL query to execute.
	Query string `config:"query" validate:"required"`

	// The name of a field in the target BigQuery table that can be used to simulate cursor pagination, e.g. an incremental ID or timestamp.
	// The following field types are supported: BIGNUMERIC, BYTES, DATE, DATETIME, FLOAT, INTEGER, NUMERIC, STRING, TIME, TIMESTAMP.
	// If not specified, the input will run the configured queries as-is on every poll. If specified, the input will add a WHERE clause
	// to each query to only select rows where the cursor field's value is greater than the last seen value.
	CursorField string `config:"cursor_field"`

	// A TIMESTAMP field in the target BigQuery table to use as the event's @timestamp value.
	TimestampField string `config:"timestamp_field"`

	// Whether to attempt to parse fields of type JSON into objects/arrays instead of leaving them as strings.
	// In the event of parsing failures, we still expand the field into a JSON object with a single field named
	// "original" containing the original string value; this avoids mapping conflicts in Elasticsearch.
	// Defaults to true.
	ExpandJsonStrings *bool `config:"expand_json_strings"`
}

func defaultConfig() config {
	return config{
		Period: time.Minute,
	}
}

func (c *config) Validate() error {
	return nil
}
