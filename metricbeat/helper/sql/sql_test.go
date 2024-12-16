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
	"database/sql/driver"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/tests/resources"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type kv struct {
	k string
	v interface{}
}

type mockVariableMode struct {
	index   int
	results []kv
}

func (m *mockVariableMode) Scan(dest ...interface{}) error {
	d1 := dest[0].(*string) //nolint:errcheck // false positive
	*d1 = m.results[m.index].k

	d2 := dest[1].(*interface{}) //nolint:errcheck // false positive
	*d2 = m.results[m.index].v

	m.index++

	return nil
}

func (m *mockVariableMode) Next() bool {
	return m.index < len(m.results)
}

func (m mockVariableMode) Columns() ([]string, error) {
	return []string{"key", "value"}, nil
}

func (m mockVariableMode) Err() error {
	return nil
}

type mockTableMode struct {
	results      []kv
	totalResults int
}

func (m *mockTableMode) Scan(dest ...interface{}) error {
	for i, d := range dest {
		d1 := d.(*interface{}) //nolint:errcheck // false positive
		*d1 = m.results[i].v
	}

	m.totalResults++

	return nil
}

func (m *mockTableMode) Next() bool {
	return m.totalResults < len(m.results)
}

func (m *mockTableMode) Columns() ([]string, error) {
	cols := make([]string, len(m.results))
	for i, r := range m.results {
		cols[i] = r.k
	}
	return cols, nil
}

func (m mockTableMode) Err() error {
	return nil
}

var results = []kv{
	{k: "string", v: "000400"},
	{k: "varchar", v: "00100"},
	{k: "hello", v: "world"},
	{k: "integer", v: int(10)},
	{k: "signed_integer", v: int(-10)},
	{k: "unsigned_integer", v: uint(100)},
	{k: "float64", v: float64(-13.2)},
	{k: "float32", v: float32(13.2)},
	{k: "null", v: nil},
	{k: "boolean", v: true},
	{k: "array", v: []interface{}{0, 1, 2}},
	{k: "byte_array", v: []byte("byte_array")},
	{k: "time", v: time.Now()},
}

func TestFetchVariableMode(t *testing.T) {
	db := DbClient{}

	ms, err := db.fetchVariableMode(&mockVariableMode{results: results})
	if err != nil {
		t.Fatal(err)
	}

	for _, res := range results {
		checkValue(t, res, ms)
	}
}

func TestFetchTableMode(t *testing.T) {
	db := DbClient{}

	mss, err := db.fetchTableMode(&mockTableMode{results: results})
	if err != nil {
		t.Fatal(err)
	}

	for _, ms := range mss {
		for _, res := range results {
			checkValue(t, res, ms)
		}
	}
}

func checkValue(t *testing.T, res kv, ms mapstr.M) {
	t.Helper()

	actual := ms[res.k]
	switch v := res.v.(type) {
	case string, bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		if actual != v {
			t.Errorf("key %q: expected %v (%T), got %v (%T)", res.k, v, v, actual, actual)
		}
	case nil:
		if actual != nil {
			t.Errorf("key %q: expected nil, got %v (%T)", res.k, actual, actual)
		}
	case []interface{}:
		actualSlice := actual.([]interface{})
		if len(v) != len(actualSlice) {
			t.Errorf("key %q: slice length mismatch: expected %d, got %d", res.k, len(v), len(actualSlice))
			return
		}
		for i, val := range v {
			if actualSlice[i] != val {
				t.Errorf("key %q: slice mismatch at index %d: expected %v, got %v", res.k, i, val, actualSlice[i])
			}
		}
	case []byte:
		actualStr := actual.(string)
		if actualStr != string(v) {
			t.Errorf("key %q: expected %q (string), got %q", res.k, string(v), actualStr)
		}
	case time.Time:
		actualStr := actual.(string)
		expectedStr := v.Format(time.RFC3339Nano)
		if expectedStr != actualStr {
			t.Errorf("key %q: expected time %q, got %q", res.k, expectedStr, actualStr)
		}
	case CustomType:
		// Handle custom types that should be converted to string
		expectedStr := fmt.Sprint(v)
		if num, err := strconv.ParseFloat(expectedStr, 64); err == nil {
			if actual != num {
				t.Errorf("key %q: expected %v (float64), got %v (%T)", res.k, num, actual, actual)
			}
		} else {
			actualStr := actual.(string)
			if actualStr != expectedStr {
				t.Errorf("key %q: expected %q (string), got %q", res.k, expectedStr, actualStr)
			}
		}
	default:
		if actual != res.v {
			t.Errorf("key %q: expected %v (%T), got %v (%T)", res.k, res.v, res.v, actual, actual)
		}
	}
}

// CustomType for testing custom type handling
type CustomType struct {
	value string //nolint:unused // unused checker is buggy
}

func TestToDotKeys(t *testing.T) {
	ms := mapstr.M{"key_value": "value"}
	ms = ReplaceUnderscores(ms)

	if ms["key"].(mapstr.M)["value"] != "value" {
		t.Fail()
	}
}

func TestNewDBClient(t *testing.T) {
	t.Run("create and close", func(t *testing.T) {
		goroutines := resources.NewGoroutinesChecker()
		defer goroutines.Check(t)

		client, err := NewDBClient("dummy", "localhost", nil)
		require.NoError(t, err)

		err = client.Close()
		require.NoError(t, err)
	})

	t.Run("unavailable", func(t *testing.T) {
		goroutines := resources.NewGoroutinesChecker()
		defer goroutines.Check(t)

		_, err := NewDBClient("dummy", "unavailable", nil)
		require.Error(t, err)
	})
}

func init() {
	sql.Register("dummy", dummyDriver{})
}

type dummyDriver struct{}

func (dummyDriver) Open(name string) (driver.Conn, error) {
	if name == "error" {
		return nil, fmt.Errorf("error")
	}

	return &dummyConnection{name: name}, nil
}

type dummyConnection struct {
	name string
}

// Ensure that this dummy connection implements the pinger interface, used by the helper.
var _ driver.Pinger = &dummyConnection{}

func (*dummyConnection) Prepare(query string) (driver.Stmt, error) { return nil, nil }
func (*dummyConnection) Close() error                              { return nil }
func (*dummyConnection) Begin() (driver.Tx, error)                 { return nil, nil }
func (c *dummyConnection) Ping(context.Context) error {
	if c.name == "unavailable" {
		return fmt.Errorf("database unavailable")
	}
	return nil
}
