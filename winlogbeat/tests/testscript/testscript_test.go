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
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"

	"github.com/elastic/beats/v7/winlogbeat/cmd"
)

var (
	update   = flag.Bool("update", false, "update txtar scripts with actual output")
	keepWork = flag.Bool("keep", false, "keep testscript work directories after test")
)

func TestMain(m *testing.M) {
	testscript.Main(m, map[string]func(){
		"winlogbeat": func() {
			if err := cmd.RootCmd.Execute(); err != nil {
				os.Exit(1)
			}
		},
	})
}

// TestScripts runs all txtar test scripts under testdata/. Each subdirectory
// becomes a subtest, allowing targeted runs such as:
//
//	go test ./tests/testscript/... -run TestScripts/export
//	go test ./tests/testscript/... -run TestScripts/config
//	go test ./tests/testscript/... -run TestScripts/eventlog
//	go test ./tests/testscript/... -run TestScripts/evtx
func TestScripts(t *testing.T) {
	evtxTestdata, err := filepath.Abs(filepath.Join("..", "..", "sys", "wineventlog", "testdata"))
	if err != nil {
		t.Fatalf("resolve evtx testdata path: %v", err)
	}

	params := testscript.Params{
		Cmds: customCommands(),
		Setup: func(env *testscript.Env) error {
			env.Setenv("EVTX_TESTDATA", evtxTestdata)
			return setupTest(env)
		},
		UpdateScripts: *update,
		TestWork:      *keepWork,
	}
	for _, sub := range []string{"export", "config", "eventlog", "evtx"} {
		t.Run(sub, func(t *testing.T) {
			p := params
			p.Dir = filepath.Join("testdata", sub)
			testscript.Run(t, p)
		})
	}
}
