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

package scripttest

import (
	"os"

	"github.com/rogpeppe/go-internal/testscript"
)

// cmdEnvsubst expands environment variables in the named files in-place.
// This is needed because testscript does not expand env vars in txtar file
// content, only in command arguments.
//
// Usage: envsubst <file> [<file>...]
func cmdEnvsubst(script *testscript.TestScript, neg bool, args []string) {
	if len(args) == 0 {
		script.Fatalf("usage: envsubst <file> [<file>...]")
	}
	for _, path := range args {
		abs := script.MkAbs(path)
		data, err := os.ReadFile(abs)
		if err != nil {
			script.Fatalf("envsubst: %v", err)
		}
		expanded := os.Expand(string(data), script.Getenv)
		if err := os.WriteFile(abs, []byte(expanded), 0o666); err != nil {
			script.Fatalf("envsubst: %v", err)
		}
	}
}
