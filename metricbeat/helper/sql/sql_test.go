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
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/tests/resources"
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
	for i, d := range dest {
		d1 := d.(*interface{})
		*d1 = m.results[i].v
	}

	m.totalResults++

	return nil
}

func (m *mockTableMode) Next() bool {
	return m.totalResults < len(m.results)
}

func (m *mockTableMode) Columns() ([]string, error) {
	return []string{"hello", "integer", "signed_integer", "unsigned_integer", "float64", "float32", "null", "boolean", "array", "byte_array", "time"}, nil
}

func (m mockTableMode) Err() error {
	return nil
}

var results = []kv{
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

func checkValue(t *testing.T, res kv, ms common.MapStr) {
	switch v := res.v.(type) {
	case string, bool:
		if ms[res.k] != v {
			t.Fail()
		}
	case nil:
		if ms[res.k] != nil {
			t.Fail()
		}
	case int:
		if ms[res.k] != float64(v) {
			t.Fail()
		}
	case uint:
		if ms[res.k] != float64(v) {
			t.Fail()
		}
	case float32:
		if math.Abs(float64(ms[res.k].(float64)-float64(v))) > 1 {
			t.Fail()
		}
	case float64:
		if ms[res.k] != v {
			t.Fail()
		}
	case []interface{}:
		for i, val := range v {
			if ms[res.k].([]interface{})[i] != val {
				t.Fail()
			}
		}
	case []byte:
		ar := ms[res.k].(string)
		if ar != string(v) {
			t.Fail()
		}
	case time.Time:
		ar := ms[res.k].(string)
		if v.Format(time.RFC3339Nano) != ar {
			t.Fail()
		}
	default:
		if ms[res.k] != res.v {
			t.Fail()
		}
	}
}

func TestToDotKeys(t *testing.T) {
	ms := common.MapStr{"key_value": "value"}
	ms = ReplaceUnderscores(ms)

	if ms["key"].(common.MapStr)["value"] != "value" {
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
