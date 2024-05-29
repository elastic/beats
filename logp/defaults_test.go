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

package logp_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/elastic/elastic-agent-libs/logp"
)

// TestDefaultConfig tests the default config ensuring the default
// behaviour is to log to files
func TestDefaultConfig(t *testing.T) {
	cfg := logp.DefaultConfig(logp.DefaultEnvironment)

	// Set cfg.Beat to avoid a log file like '-202405029.ndjson'
	cfg.Beat = t.Name()

	// Set Files.Path to an empty folder so we can assert the
	// creation of the log file without being affected by other
	// test runs.
	cfg.Files.Path = t.TempDir()

	if err := logp.Configure(cfg); err != nil {
		t.Fatalf("did not expect an error calling logp.Configure: %s", err)
	}

	// Get the logger and log anything
	logger := logp.L()
	defer logger.Close()

	logger.Info("foo")
	_, fileName, lineNum, _ := runtime.Caller(0)
	lineNum-- // We want the line number from the log
	fileName = filepath.Base(fileName)

	// Assert the log file was created
	glob := fmt.Sprintf("%s-*.ndjson", t.Name())
	glob = filepath.Join(cfg.Files.Path, glob)
	logFiles, err := filepath.Glob(glob)
	if err != nil {
		t.Fatalf("could not list files for glob '%s', err: %s", glob, err)
	}

	if len(logFiles) < 1 {
		t.Fatalf("did not find any log file")
	}

	data, err := os.ReadFile(logFiles[0])
	if err != nil {
		t.Fatalf("cannot open file '%s', err: %s", logFiles[0], err)
	}

	logEntry := struct {
		LogLevel  string `json:"log.level"`
		LogOrigin struct {
			FileName string `json:"file.name"`
			FileLine int    `json:"file.line"`
		} `json:"log.origin"`
		Message     string `json:"message"`
		ServiceName string `json:"service.name"`
	}{}
	if err := json.Unmarshal(data, &logEntry); err != nil {
		t.Fatalf("cannot unmarshal log entry: %s", err)
	}

	if got, expect := logEntry.Message, "foo"; got != expect {
		t.Errorf("expecting message '%s', got '%s'", expect, got)
	}

	if got, expect := logEntry.LogLevel, "info"; got != expect {
		t.Errorf("expecting level '%s', got '%s'", expect, got)
	}

	if got, expect := logEntry.ServiceName, t.Name(); got != expect {
		t.Errorf("expecting service name '%s', got '%s'", expect, got)
	}

	if got, expect := filepath.Base(logEntry.LogOrigin.FileName), fileName; got != expect {
		t.Errorf("expecting log.origin.file.name '%s', got '%s'", expect, got)
	}

	if got, expect := logEntry.LogOrigin.FileLine, lineNum; got != expect {
		t.Errorf("expecting log.origin.file.line '%d', got '%d'", expect, got)
	}

	if t.Failed() {
		t.Log("Original log entry:")
		t.Log(string(data))
	}
}
