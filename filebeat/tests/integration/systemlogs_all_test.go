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

package integration

import (
	"bufio"
	_ "embed"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	cp "github.com/otiai10/copy"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/filebeat_system_module.yml
var systemModuleCfg string

func copyModulesDir(t *testing.T, dst string) {
	pwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("cannot get the current directory: %s", err)
	}
	localModules := filepath.Join(pwd, "../", "../", "module")
	localModulesD := filepath.Join(pwd, "../", "../", "modules.d")

	if err := cp.Copy(localModules, filepath.Join(dst, "module")); err != nil {
		t.Fatalf("cannot copy 'module' folder to test folder: %s", err)
	}
	if err := cp.Copy(localModulesD, filepath.Join(dst, "modules.d")); err != nil {
		t.Fatalf("cannot copy 'modules.d' folder to test folder: %s", err)
	}
}

//nolint:unused,nolintlint // necessary on Linux
func waitForAllFilesets(t *testing.T, outputGlob string, msgAndArgs ...any) {
	require.Eventually(
		t,
		findFilesetNames(t, outputGlob),
		time.Minute,
		10*time.Millisecond,
		msgAndArgs...)
}

//nolint:unused,nolintlint // necessary on Linux
func findFilesetNames(t *testing.T, outputGlob string) func() bool {
	f := func() bool {
		files, err := filepath.Glob(outputGlob)
		if err != nil {
			t.Fatalf("cannot get files list for glob '%s': '%s'", outputGlob, err)
		}

		if len(files) > 1 {
			t.Fatalf(
				"only a single output file is supported, found: %d. Files: %s",
				len(files),
				files,
			)
		}

		foundSyslog := false
		foundAuth := false

		file, err := os.Open(files[0])
		if err != nil {
			t.Fatalf("cannot open '%s': '%s'", files[0], err)
		}
		defer file.Close()

		r := bufio.NewReader(file)
		for {
			line, err := r.ReadBytes('\n')
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				} else {
					t.Fatalf("cannot read '%s': '%s", file.Name(), err)
				}
			}

			data := struct {
				Fileset struct {
					Name string `json:"name"`
				} `json:"fileset"`
			}{}

			if err := json.Unmarshal(line, &data); err != nil {
				t.Fatalf("cannot parse output line as JSON: %s", err)
			}

			switch data.Fileset.Name {
			case "syslog":
				foundSyslog = true
			case "auth":
				foundAuth = true
			}

			if foundAuth && foundSyslog {
				return true
			}
		}

		return false
	}

	return f
}
