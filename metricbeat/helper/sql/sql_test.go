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
	d1 := dest[0].(*string)
	*d1 = m.results[m.index].k

	d2 := dest[1].(*interface{})
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
	if len(dest) != len(m.results) {
		return fmt.Errorf("expected %d results, got %d", len(m.results), len(dest))
	}

	for i, d := range dest {
		dPtr := d.(*interface{})
		*dPtr = m.results[i].v
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
	{k: "varchar", v: []byte("00100")},
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
	switch v := res.v.(type) {
	case nil:
		if ms[res.k] != nil {
			t.Fatalf("Value mismatch for key '%s': expected nil, got %v", res.k, ms[res.k])
		}
	case bool, int64, uint64, float32, float64, string:
		if ms[res.k] != v {
			t.Fatalf("Value mismatch for key '%s': expected %v (%T), got %v (%T)", res.k, v, v, ms[res.k], ms[res.k])
		}
	case int:
		if ms[res.k] != int64(v) {
			t.Fatalf("Value mismatch for key '%s': expected %v (int64), got %v (%T)", res.k, int64(v), ms[res.k], ms[res.k])
		}
	case int32:
		if ms[res.k] != int64(v) {
			t.Fatalf("Value mismatch for key '%s': expected %v (int64), got %v (%T)", res.k, int64(v), ms[res.k], ms[res.k])
		}
	case uint:
		if ms[res.k] != uint64(v) {
			t.Fatalf("Value mismatch for key '%s': expected %v (uint64), got %v (%T)", res.k, uint64(v), ms[res.k], ms[res.k])
		}
	case uint32:
		if ms[res.k] != uint64(v) {
			t.Fatalf("Value mismatch for key '%s': expected %v (uint64), got %v (%T)", res.k, uint64(v), ms[res.k], ms[res.k])
		}
	case []byte:
		msVal, ok := ms[res.k].(string)
		if !ok {
			t.Fatalf("Type mismatch for key '%s': expected string, got %T", res.k, ms[res.k])
		} else if string(v) != msVal {
			t.Fatalf("Value mismatch for key '%s': expected %s, got %s", res.k, string(v), msVal)
		}
	case []interface{}:
		for i, val := range v {
			if ms[res.k].([]interface{})[i] != val {
				t.Fatal()
			}
		}
	case time.Time:
		msVal, ok := ms[res.k].(string)
		if !ok {
			t.Fatalf("Type mismatch for key '%s': expected string, got %T", res.k, ms[res.k])
		} else if v.Format(time.RFC3339Nano) != msVal {
			t.Fatalf("Value mismatch for key '%s': expected %s, got %s", res.k, v.Format(time.RFC3339Nano), msVal)
		}
	default:
		t.Fatalf("Unsupported type for key '%s': %T", res.k, v)
	}
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
