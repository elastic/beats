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

package keyspace

import (
	"math"
	"strconv"
	"testing"
)

// getInt64 attempts to normalize various numeric types (and numeric strings) to int64 for assertions.
func getInt64(t *testing.T, v interface{}) int64 {
	switch n := v.(type) {
	case int:
		return int64(n)
	case int8:
		return int64(n)
	case int16:
		return int64(n)
	case int32:
		return int64(n)
	case int64:
		return n
	case uint:
		// check for overflow when converting platform-dependent uint to int64
		if uint64(n) <= uint64(math.MaxInt64) {
			return int64(n)
		}
		t.Fatalf("uint value overflows int64: %v", n)
	case uint8:
		return int64(n)
	case uint16:
		return int64(n)
	case uint32:
		return int64(n)
	case uint64:
		// check for overflow when converting uint64 to int64
		if n <= uint64(math.MaxInt64) {
			return int64(n)
		}
		t.Fatalf("uint64 value overflows int64: %v", n)
	case string:
		if s, err := strconv.ParseInt(n, 10, 64); err == nil {
			return s
		}
	}
	t.Fatalf("unexpected numeric type %T with value %v", v, v)
	return 0
}

func TestParseKeyspaceStats(t *testing.T) {
	input := map[string]string{
		"db0": "keys=795341,expires=0,avg_ttl=0",
		"db1": "keys=10,expires=1,avg_ttl=123,subexpiry=5",
		"db2": "invalid",
	}

	out := parseKeyspaceStats(input)

	// db2 is malformed and should be ignored
	if _, ok := out["db2"]; ok {
		t.Fatalf("expected db2 to be ignored, but it was present: %v", out["db2"])
	}

	// Expect db0 and db1
	if len(out) != 2 {
		t.Fatalf("expected 2 keyspace entries, got %d: %v", len(out), out)
	}

	// Validate db0
	db0, ok := out["db0"]
	if !ok {
		t.Fatalf("db0 missing in output: %v", out)
	}

	if getInt64(t, db0["keys"]) != 795341 {
		t.Fatalf("db0.keys: expected 795341, got %v", db0["keys"])
	}
	if getInt64(t, db0["expires"]) != 0 {
		t.Fatalf("db0.expires: expected 0, got %v", db0["expires"])
	}
	if getInt64(t, db0["avg_ttl"]) != 0 {
		t.Fatalf("db0.avg_ttl: expected 0, got %v", db0["avg_ttl"])
	}
	// subexpiry should be added with default 0
	if getInt64(t, db0["subexpiry"]) != 0 {
		t.Fatalf("db0.subexpiry: expected default 0, got %v", db0["subexpiry"])
	}

	// Validate db1 (with explicit subexpiry)
	db1, ok := out["db1"]
	if !ok {
		t.Fatalf("db1 missing in output: %v", out)
	}
	if getInt64(t, db1["keys"]) != 10 {
		t.Fatalf("db1.keys: expected 10, got %v", db1["keys"])
	}
	if getInt64(t, db1["expires"]) != 1 {
		t.Fatalf("db1.expires: expected 1, got %v", db1["expires"])
	}
	if getInt64(t, db1["avg_ttl"]) != 123 {
		t.Fatalf("db1.avg_ttl: expected 123, got %v", db1["avg_ttl"])
	}
	if getInt64(t, db1["subexpiry"]) != 5 {
		t.Fatalf("db1.subexpiry: expected 5, got %v", db1["subexpiry"])
	}
}
