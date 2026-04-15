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
	"unsafe"

	sonicDecoder "github.com/bytedance/sonic/decoder"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common/jsontransform"
	"github.com/elastic/elastic-agent-libs/logp"
)

// oracleLines is a varied corpus exercising different JSON shapes.
// Variety matters because sonic may alias some string forms and copy others
// (e.g., keys are always aliased; escaped-value strings are always copied).
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

// TestDecodeBufferSafety is an oracle+regression test for JSONReader.decode().
//
// It verifies two properties:
//
//  1. Correctness: decode() produces the same result as the oracle (a fresh
//     sonic decoder given a safe string copy of the same input).
//
//  2. Buffer safety: overwriting the input []byte immediately after decode()
//     does not corrupt the decoded values. This would catch any attempt to
//     switch decode() back to unsafe.String — see the NOTE below.
//
// NOTE: sonic aliases map keys AND unescaped string values into the input
// buffer when given a no-copy input via unsafe.String. We confirmed this
// empirically: zeroing the buffer after an unsafe.String decode turns every
// key and most values into "XXX...X". The production code uses string(text)
// to make a safe copy before passing to sonic; this test guards that choice.
//
// Run with -race to additionally catch any concurrent-access issues.
func TestDecodeBufferSafety(t *testing.T) {
	// Oracle decoder: always given safe copies — used to generate expected output.
	oracle := sonicDecoder.NewDecoder("")
	oracle.UseNumber()

	// Production reader under test.
	r := &JSONReader{
		cfg:    &Config{OverwriteKeys: true},
		logger: logp.NewLogger("oracle_test"),
	}

	for i, line := range oracleLines {
		t.Run(fmt.Sprintf("line_%d", i), func(t *testing.T) {
			// Oracle: decode from an unambiguously safe copy.
			var expected map[string]interface{}
			oracle.Reset(string(line))
			require.NoError(t, oracle.Decode(&expected), "oracle decode failed")
			jsontransform.TransformNumbers(expected)

			// Production path: give decode() a mutable copy of the bytes.
			buf := make([]byte, len(line))
			copy(buf, line)

			_, got := r.decode(buf)

			// Must match oracle before any tampering.
			require.Equal(t, expected, map[string]interface{}(got),
				"decode() result does not match oracle")

			// Overwrite the mutable buffer with 'X' bytes.
			// If decode() (or sonic internally) aliased any string into buf,
			// those strings now read as "XXX...X" and the comparison below fails.
			for j := range buf {
				buf[j] = 'X'
			}

			// Must still match oracle after the buffer is trashed.
			require.Equal(t, expected, map[string]interface{}(got),
				"decode() strings alias the input buffer: result was corrupted after overwrite. "+
					"If decode() was changed to use unsafe.String, revert to string(text).")
		})
	}
}

// TestDecodeSequentialNoCrossContamination verifies that decoding line N+1
// does not corrupt the results of line N.
//
// This catches the failure mode where a decoder keeps an internal reference
// to the previous input and overwrites prior decoded strings on Reset().
func TestDecodeSequentialNoCrossContamination(t *testing.T) {
	r := &JSONReader{
		cfg:    &Config{OverwriteKeys: true},
		logger: logp.NewLogger("oracle_test"),
	}

	// Oracle for generating expected values.
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

// TestUnsafeStringAliasesInput documents — and proves — why unsafe.String was
// rejected as input to the sonic decoder.
//
// Sonic aliases map keys and unescaped string values directly into the input
// buffer. If the buffer is later overwritten, the decoded map is corrupted.
// This test is expected to FAIL on the unsafe decoder; it exists to make the
// aliasing behaviour explicit and machine-verifiable.
func TestUnsafeStringAliasesInput(t *testing.T) {
	dc := sonicDecoder.NewDecoder("")
	dc.UseNumber()

	line := oracleLines[0] // flat all-string line: aliasing is most visible here

	// Oracle: safe copy.
	var expected map[string]interface{}
	dc.Reset(string(line))
	require.NoError(t, dc.Decode(&expected))
	jsontransform.TransformNumbers(expected)

	// Unsafe path: alias the input buffer.
	buf := make([]byte, len(line))
	copy(buf, line)

	var got map[string]interface{}
	dc.Reset(unsafe.String(unsafe.SliceData(buf), len(buf))) //nolint:gosec
	require.NoError(t, dc.Decode(&got))
	jsontransform.TransformNumbers(got)

	// Must match before overwrite.
	require.Equal(t, expected, got, "unsafe decode produced wrong result")

	// Zero the buffer.
	for j := range buf {
		buf[j] = 'X'
	}

	// This FAILS: all keys (always aliased) and most values (aliased when no
	// escape sequences) are now "XXX...X". This is the proof that unsafe.String
	// must not be used as sonic input in production code.
	aliased := require.New(t)
	aliased.NotEqual(expected, got,
		"expected corruption was not detected — sonic may have changed its aliasing behaviour; "+
			"re-evaluate whether string(text) is still necessary")
}
