// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package sql

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"time"

	"strings"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type DbClient struct {
	*sql.DB
	logger *logp.Logger
}

type sqlRow interface {
	Scan(dest ...interface{}) error
	Next() bool
	Columns() ([]string, error)
	Err() error
}

// NewDBClient gets a client ready to query the database
func NewDBClient(driver, uri string, l *logp.Logger) (*DbClient, error) {
	dbx, err := sql.Open(SwitchDriverName(driver), uri)
	if err != nil {
		return nil, fmt.Errorf("opening connection: %w", err)
	}
	err = dbx.Ping()
	if err != nil {
		if closeErr := dbx.Close(); closeErr != nil {
			// NOTE(SS): Support for wrapping multiple errors is there in Go 1.20+.
			// TODO(SS): When beats module starts using Go 1.20+, use: https://pkg.go.dev/errors#Join
			// and until then, let's use the following workaround.
			return nil, fmt.Errorf(fmt.Sprintf("failed to close with: %s", closeErr.Error())+" after connection test failed: %w", err)
		}
		return nil, fmt.Errorf("testing connection: %w", err)
	}

	return &DbClient{DB: dbx, logger: l}, nil
}

// FetchTableMode scan the rows and publishes the event for querys that return the response in a table format.
func (d *DbClient) FetchTableMode(ctx context.Context, q string) ([]mapstr.M, error) {
	rows, err := d.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	return d.fetchTableMode(rows)
}

// fetchTableMode scan the rows and publishes the event for querys that return the response in a table format.
func (d *DbClient) fetchTableMode(rows sqlRow) ([]mapstr.M, error) {
	// Extracted from
	// https://stackoverflow.com/questions/23507531/is-golangs-sql-package-incapable-of-ad-hoc-exploratory-queries/23507765#23507765
	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("error getting columns: %w", err)
	}

	for k, v := range cols {
		cols[k] = strings.ToLower(v)
	}

	vals := make([]interface{}, len(cols))
	for i := 0; i < len(cols); i++ {
		vals[i] = new(interface{})
	}

	rr := make([]mapstr.M, 0)
	for rows.Next() {
		err = rows.Scan(vals...)
		if err != nil {
			d.logger.Debug(fmt.Errorf("error trying to scan rows: %w", err))
			continue
		}

		r := mapstr.M{}

		for i, c := range cols {
			value := getValue(vals[i].(*interface{})) //nolint:errcheck
			r.Put(c, value)
		}

		rr = append(rr, r)
	}

	if err = rows.Err(); err != nil {
		d.logger.Debug(fmt.Errorf("error trying to read rows: %w", err))
	}

	return rr, nil
}

// FetchVariableMode executes the provided SQL query and returns the results in a key/value format.
func (d *DbClient) FetchVariableMode(ctx context.Context, q string) (mapstr.M, error) {
	rows, err := d.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	return d.fetchVariableMode(rows)
}

// fetchVariableMode scans the provided SQL rows and returns the results in a key/value format.
func (d *DbClient) fetchVariableMode(rows sqlRow) (mapstr.M, error) {
	data := mapstr.M{}

	for rows.Next() {
		var key string
		var val interface{}
		err := rows.Scan(&key, &val)
		if err != nil {
			d.logger.Debug(fmt.Errorf("error trying to scan rows: %w", err))
			continue
		}

		key = strings.ToLower(key)
		data[key] = val
	}

	if err := rows.Err(); err != nil {
		d.logger.Debug(fmt.Errorf("error trying to read rows: %w", err))
	}

	r := mapstr.M{}

	for key, value := range data {
		value := value
		value = getValue(&value)
		r.Put(key, value)
	}

	return r, nil
}

// ReplaceUnderscores takes the root keys of a common.Mapstr and rewrites them replacing underscores with dots. Check tests
// to see an example.
func ReplaceUnderscores(ms mapstr.M) mapstr.M {
	dotMap := mapstr.M{}
	for k, v := range ms {
		dotMap.Put(strings.ReplaceAll(k, "_", "."), v)
	}

	return dotMap
}

func getValue(pval *interface{}) interface{} {
	if pval == nil {
		return nil
	}

	switch v := (*pval).(type) {
	case nil, bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, string, []interface{}:
		return v
	case []byte:
		return string(v)
	case time.Time:
		return v.Format(time.RFC3339Nano)
	default:
		// For any other types, convert to string and try to parse as number
		s := fmt.Sprint(v)
		if len(s) > 1 && s[0] == '0' && s[1] != '.' {
			// Preserve string with leading zeros i.e., 00100 stays 00100
			return s
		}
		if num, err := strconv.ParseFloat(s, 64); err == nil {
			return num
		}
		return s
	}
}

// SwitchDriverName switches between driver name and a pretty name for a driver. For example, 'oracle' driver is called
// 'godror' so this detail implementation must be hidden to the user, that should only choose and see 'oracle' as driver
func SwitchDriverName(d string) string {
	switch d {
	case "oracle":
		return "godror"
	case "cockroachdb":
		return "postgres"
	case "cockroach":
		return "postgres"
	case "postgresql":
		return "postgres"
	}

	return d
}

// internal error type for SanitizeError function to support both error message replacing and error wrapping
type sqlSanitizedError struct {
	sanitized string
	err       error
}

func (err *sqlSanitizedError) Error() string {
	return err.sanitized
}

func (err *sqlSanitizedError) Unwrap() error {
	return err.err
}

const redacted = "(redacted)"

var patterns = []struct {
	re   *regexp.Regexp
	repl string
}{
	// scheme://user:password@host -> redact password
	{regexp.MustCompile(`([a-z][a-z0-9+\.\-]*://[^:/?#\s]+):([^@/\s]+)@`), `$1:` + redacted + `@`},
	// user:password@... (no scheme), common in MySQL DSNs like user:pass@tcp(...)
	// Important: Disallow '/' in the password part to avoid matching URI schemes like "postgres://..."
	{regexp.MustCompile(`(^|[\s'\"])([^:/@\s]+):([^@/\s]+)@`), `$1$2:` + redacted + `@`},

	// Key=Value forms (connection strings): handle quoted and unquoted values.
	// Single-quoted values: Password='secret'; Token='abc';
	{regexp.MustCompile(`(?i)\b(password|pwd|pass|passwd|token|secret)\b\s*=\s*'[^']*'`), `$1='` + redacted + `'`},
	// Double-quoted values: Password="secret value";
	{regexp.MustCompile(`(?i)\b(password|pwd|pass|passwd|token|secret)\b\s*=\s*"[^"]*"`), `$1="` + redacted + `"`},
	// Unquoted values until delimiter or whitespace: Password=secret123; PASS=foo
	{regexp.MustCompile(`(?i)\b(password|pwd|pass|passwd|token|secret)\b\s*=\s*[^;,#&\s]+`), `$1=` + redacted},

	// JSON-style fields: {"password":"secret"}
	{regexp.MustCompile(`(?i)"(password|pwd|pass|passwd|token|secret)"\s*:\s*"(?:[^"\\]|\\.)*"`), `"$1":"` + redacted + `"`},

	// Query parameters in URLs: ?password=secret&...
	{regexp.MustCompile(`(?i)([?&])(password|pwd|pass|passwd|token|secret)\s*=\s*([^&#\s]+)`), `$1$2=` + redacted},
}

// SanitizeError replaces all occurrences of 'sensitive' parameter in err.Error() with "(redacted)"
// It also sanitizes common patterns that might contain passwords or sensitive data
func SanitizeError(err error, sensitive string) error {
	if err == nil {
		return nil
	}

	msg := err.Error()

	// First, replace the primary sensitive string if provided (raw, quoted, and URL-encoded forms)
	if s := strings.TrimSpace(sensitive); s != "" {
		// raw
		msg = strings.ReplaceAll(msg, s, redacted)
		// quoted (fmt %q style)
		quoted := fmt.Sprintf("%q", s)
		msg = strings.ReplaceAll(msg, quoted, redacted)
		// URL-encoded (both query and path escaping just to be safe)
		qEsc := url.QueryEscape(s)
		if qEsc != s {
			msg = strings.ReplaceAll(msg, qEsc, redacted)
		}
		pEsc := url.PathEscape(s)
		if pEsc != s && pEsc != qEsc {
			msg = strings.ReplaceAll(msg, pEsc, redacted)
		}
	}

	// Pattern-based sanitization for common secrets in errors (URLs, DSNs, key/value strings, JSON, query params).
	// Order matters: apply more specific URL userinfo patterns first.

	for _, p := range patterns {
		msg = p.re.ReplaceAllString(msg, p.repl)
	}

	return &sqlSanitizedError{
		sanitized: msg,
		err:       err,
	}
}
