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

package readjson

import (
	"fmt"
	"testing"

	sonicDecoder "github.com/bytedance/sonic/decoder"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common/jsontransform"
	"github.com/elastic/elastic-agent-libs/logp"
)

// oracleLines is a varied corpus exercising different JSON shapes.
// Variety matters because sonic aliases different forms differently:
// map keys are always aliased; plain string values are aliased; escaped-value
// strings are copied by sonic (they require decoding, so a fresh allocation).
var oracleLines = [][]byte{
	// flat, all-string values (common log line shape)
	[]byte(`{"message":"GET /api/users 200","level":"info","timestamp":"2024-01-15T10:30:00Z","method":"GET","path":"/api/users"}`),
	// mixed strings + integers
	[]byte(`{"message":"GET /api/users 200","level":"info","duration":142,"status":200,"bytes_sent":1024}`),
	// strings with escape sequences (sonic copies these; plain substrings are aliased)
	[]byte(`{"message":"line with \"quotes\" and \\backslash","path":"/foo\/bar","tab":"\t end"}`),
	// deeply nested
	[]byte(`{"event":{"kind":"event"},"host":{"hostname":"x-wing","id":"abc123"},"journald":{"pid":42,"process":{"name":"sudo"}}}`),
	// arrays of strings and numbers
	[]byte(`{"args":["sudo","journalctl","--user"],"counts":[1,2,3],"label":"test"}`),
	// unicode
	[]byte(`{"message":"こんにちは世界","emoji":"🎉","level":"info"}`),
	// large realistic journald line
	[]byte(`{"message":"pam_unix(sudo:session): session closed for user root","event":{"kind":"event"},"host":{"hostname":"x-wing","id":"a6a19d57efcf4bf38705c63217a63ba3"},"journald":{"audit":{"login_uid":1000,"session":"1"},"custom":{"syslog_timestamp":"Nov 22 18:10:04 "},"gid":0,"pid":2084586,"process":{"capabilities":"1ffffffffff","command_line":"sudo journalctl --user --rotate","executable":"/usr/bin/sudo","name":"sudo"},"uid":1000}}`),
}

// TestDecodeCorrectnessOracle verifies that decode() produces the same output
// as a reference sonic decoder given a safe string copy of the same input.
//
// Run with -race to additionally catch any concurrent-access issues.
func TestDecodeCorrectnessOracle(t *testing.T) {
	oracle := sonicDecoder.NewDecoder("")
	oracle.UseNumber()

	r := &JSONReader{
		cfg:    &Config{OverwriteKeys: true},
		logger: logp.NewLogger("oracle_test"),
	}

	for i, line := range oracleLines {
		t.Run(fmt.Sprintf("line_%d", i), func(t *testing.T) {
			var expected map[string]interface{}
			oracle.Reset(string(line))
			require.NoError(t, oracle.Decode(&expected), "oracle decode failed")
			jsontransform.TransformNumbers(expected)

			buf := make([]byte, len(line))
			copy(buf, line)
			_, got := r.decode(buf)

			require.Equal(t, expected, map[string]interface{}(got))
		})
	}
}

// TestDecodeAliasesBuffer documents and proves that decode() aliases decoded
// strings directly into the input buffer via sonic's no-copy path.
//
// This aliasing is intentional: it eliminates a per-line string allocation for
// every map key and every unescaped string value. It is safe because all
// production callers pass slices from streambuf.Collect(), which advances its
// internal cursor past consumed bytes — those positions are never overwritten.
// Strings in event.Fields keep the backing array reachable via GC, so aliased
// data remains valid for the event's lifetime.
//
// This test asserts that corruption IS detected after an explicit buffer
// overwrite, confirming sonic is performing the aliasing we rely on. If this
// test starts passing (i.e., corruption is no longer detected), sonic may have
// changed its aliasing behaviour and the unsafe.String path in decode() should
// be re-evaluated.
func TestDecodeAliasesBuffer(t *testing.T) {
	oracle := sonicDecoder.NewDecoder("")
	oracle.UseNumber()

	r := &JSONReader{
		cfg:    &Config{OverwriteKeys: true},
		logger: logp.NewLogger("oracle_test"),
	}

	// Use the flat all-string line — aliasing is most visible because no escape
	// sequences force sonic to copy anything.
	line := oracleLines[0]

	var expected map[string]interface{}
	oracle.Reset(string(line))
	require.NoError(t, oracle.Decode(&expected))
	jsontransform.TransformNumbers(expected)

	buf := make([]byte, len(line))
	copy(buf, line)
	_, got := r.decode(buf)

	require.Equal(t, expected, map[string]interface{}(got), "decode produced wrong result before overwrite")

	// Overwrite the buffer — simulates a caller that reuses the buffer for the
	// next line, which streambuf never does for already-consumed byte positions.
	for j := range buf {
		buf[j] = 'X'
	}

	// Expect corruption: keys and plain string values are aliased, so they now
	// read as "XXX...X". This confirms sonic is aliasing into the buffer.
	require.NotEqual(t, expected, map[string]interface{}(got),
		"aliasing not detected after buffer overwrite: sonic may have changed "+
			"its no-copy behaviour; re-evaluate the unsafe.String path in decode()")
}

// TestStreambufLifetimeInvariant is the primary safety proof for the
// unsafe.String aliasing used in decode().
//
// It simulates the exact streambuf memory model used in production:
//
//  1. streambuf.Collect() returns b.data[mark:mark+count] — a slice of an
//     internal backing array.
//  2. streambuf.Reset() advances the cursor: b.data = b.data[mark:].
//  3. Subsequent appends land at positions ≥ the new cursor — AFTER the
//     bytes that were already consumed and decoded.
//  4. Therefore old aliased positions are never overwritten; strings stored in
//     event.Fields remain valid as long as the GC can reach them.
//
// The test encodes this sequence directly and asserts that results decoded from
// the first Collect() slice are uncorrupted after a second append+decode cycle.
func TestStreambufLifetimeInvariant(t *testing.T) {
	r := &JSONReader{
		cfg:    &Config{OverwriteKeys: true},
		logger: logp.NewLogger("lifetime_test"),
	}
	oracle := sonicDecoder.NewDecoder("")
	oracle.UseNumber()

	line0 := oracleLines[0] // flat all-string: aliasing most visible
	line1 := oracleLines[6] // large realistic journald line

	var exp0 map[string]interface{}
	oracle.Reset(string(line0))
	require.NoError(t, oracle.Decode(&exp0))
	jsontransform.TransformNumbers(exp0)

	var exp1 map[string]interface{}
	oracle.Reset(string(line1))
	require.NoError(t, oracle.Decode(&exp1))
	jsontransform.TransformNumbers(exp1)

	// --- Simulate streambuf ---

	// Allocate a single backing array large enough for both lines, mirroring
	// the streambuf internal buffer before any growth reallocation.
	backing := make([]byte, 0, len(line0)+len(line1))

	// Step 1: append line0 and take a Collect()-style slice.
	backing = append(backing, line0...)
	slice0 := backing[:len(line0)]

	_, got0 := r.decode(slice0)
	require.Equal(t, exp0, map[string]interface{}(got0), "line0 initial decode mismatch")

	// Step 2: advance past line0 (simulates Reset: b.data = b.data[mark:]),
	// then append line1 — lands at positions ≥ len(line0), AFTER line0's bytes.
	backing = backing[len(line0):]
	backing = append(backing, line1...)
	slice1 := backing[:len(line1)]

	_, got1 := r.decode(slice1)
	require.Equal(t, exp1, map[string]interface{}(got1), "line1 initial decode mismatch")

	// Step 3: verify line0 results are uncorrupted.
	//
	// If decode() aliased line0's strings into bytes [0:len(line0)] of the
	// original backing array, and the append in step 2 had overwritten those
	// bytes, got0 would be corrupted here. It is not — append extended the
	// array past those positions, leaving aliased strings intact.
	//
	// This mirrors exactly what streambuf does: new log lines are appended
	// after the mark, never over already-consumed bytes.
	require.Equal(t, exp0, map[string]interface{}(got0),
		"line0 decoded values corrupted after line1 was appended and decoded: "+
			"the streambuf lifetime invariant does not hold for this input")
}

// TestDecodeSequentialNoCrossContamination verifies that decoding line N+1
// does not corrupt the results of line N.
//
// Each line is decoded from an independent buffer. sonic aliases strings into
// each buffer's backing array; those arrays remain alive via GC as long as the
// decoded strings in the results maps hold pointers to them. Resetting the
// shared decoder (r.dec) to a new input does not touch old buffers.
func TestDecodeSequentialNoCrossContamination(t *testing.T) {
	r := &JSONReader{
		cfg:    &Config{OverwriteKeys: true},
		logger: logp.NewLogger("oracle_test"),
	}

	oracle := sonicDecoder.NewDecoder("")
	oracle.UseNumber()

	type entry struct {
		expected map[string]interface{}
		got      map[string]interface{}
	}
	results := make([]entry, len(oracleLines))

	// Decode ALL lines, accumulating results from both the oracle and production.
	for i, line := range oracleLines {
		var expected map[string]interface{}
		oracle.Reset(string(line))
		require.NoError(t, oracle.Decode(&expected))
		jsontransform.TransformNumbers(expected)

		buf := make([]byte, len(line))
		copy(buf, line)
		_, got := r.decode(buf)

		results[i] = entry{expected, map[string]interface{}(got)}
	}

	// After all decodes (r.dec has been Reset many times), verify every prior
	// result is still uncorrupted.
	for i := range oracleLines {
		require.Equal(t, results[i].expected, results[i].got,
			"line %d result was contaminated by subsequent decodes", i)
	}
}
