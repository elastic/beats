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
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"

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

func TestWith(t *testing.T) {
	tempDir := t.TempDir()

	logSelector := "enabled-log-selector"

	defaultCfg := logp.DefaultConfig(logp.DefaultEnvironment)
	eventsCfg := logp.DefaultEventConfig(logp.DefaultEnvironment)

	defaultCfg.Level = logp.DebugLevel
	defaultCfg.Beat = t.Name()
	defaultCfg.Selectors = []string{logSelector}
	defaultCfg.ToStderr = false
	defaultCfg.ToFiles = true
	defaultCfg.Files.Path = tempDir

	eventsCfg.Level = defaultCfg.Level
	eventsCfg.Beat = defaultCfg.Beat
	eventsCfg.Selectors = defaultCfg.Selectors
	eventsCfg.ToStderr = defaultCfg.ToStderr
	eventsCfg.ToFiles = defaultCfg.ToFiles
	eventsCfg.Files.Path = tempDir

	if err := logp.ConfigureWithTypedOutput(defaultCfg, eventsCfg, "log.type", "event"); err != nil {
		t.Fatalf("could not configure logger: %s", err)
	}

	expectedLines := []string{
		// First/Default logger
		`{"log.level":"info","message":"Very first message"}`,

		// Two messages after calling With
		`{"log.level":"info","message":"a message with extra fields","foo":"bar"}`,
		`{"log.level":"info","message":"another message with extra fields","foo":"bar"}`,

		// A message with the default logger
		`{"log.level":"info","message":"a message without extra fields"}`,

		// Two more messages with a different field
		`{"log.level":"info","message":"a message with an answer","answer":"42"}`,
		`{"log.level":"info","message":"another message with an answer","answer":"42"}`,

		// One last message with the default logger
		`{"log.level":"info","message":"another message without any extra fields"}`,
	}

	logger := logp.L()
	defer logger.Close()

	logger.Info("Very first message")

	// Add a field and write messages
	loggerWithFields := logger.With(strField("foo", "bar"))
	loggerWithFields.Info("a message with extra fields")
	loggerWithFields.Info("another message with extra fields")

	// Use the default logger again
	logger.Info("a message without extra fields")

	// New logger with other fields
	loggerWithFields = logger.With(strField("answer", "42"))
	loggerWithFields.Info("a message with an answer")
	loggerWithFields.Info("another message with an answer")

	// One last message with the default logger
	logger.Info("another message without any extra fields")

	entries := takeAllLogsFromPath(t, tempDir)

	if len(expectedLines) != len(entries) {
		t.Fatalf("expecting %d lines got %d", len(expectedLines), len(entries))
	}

	sort.Slice(entries, func(i, j int) bool {
		t1 := entries[i]["@timestamp"].(string) //nolint: errcheck // We know it's a sting and it is a test
		t2 := entries[j]["@timestamp"].(string) //nolint: errcheck // We know it's a sting and it is a test
		return t1 < t2
	})

	// Now that the slice is sorted, remove some fields, including
	// the @timestamp we used to sort it
	for i := range entries {
		delete(entries[i], "@timestamp")
		delete(entries[i], "log.origin")
		delete(entries[i], "ecs.version")
		delete(entries[i], "service.name")
	}

	strEntries := []string{}
	for _, e := range entries {
		data, _ := json.Marshal(e)
		strEntries = append(strEntries, string(data))
	}

	for i := range strEntries {
		assert.JSONEq(t, strEntries[i], expectedLines[i], "Some log entries are different than expected")
	}
}

func TestConcurrency(t *testing.T) {
	tempDir := t.TempDir()

	logSelector := "enabled-log-selector"

	defaultCfg := logp.DefaultConfig(logp.DefaultEnvironment)
	eventsCfg := logp.DefaultEventConfig(logp.DefaultEnvironment)

	defaultCfg.Level = logp.DebugLevel
	defaultCfg.Beat = t.Name()
	defaultCfg.Selectors = []string{logSelector}
	defaultCfg.ToStderr = false
	defaultCfg.ToFiles = true
	defaultCfg.Files.Path = tempDir

	eventsCfg.Level = defaultCfg.Level
	eventsCfg.Beat = defaultCfg.Beat
	eventsCfg.Selectors = defaultCfg.Selectors
	eventsCfg.ToStderr = defaultCfg.ToStderr
	eventsCfg.ToFiles = defaultCfg.ToFiles
	eventsCfg.Files.Path = tempDir

	expectedLines := []string{
		`{"id":0,"log.level":"info","message":"count: 000","sort_field":0}`,
		`{"id":0,"log.level":"info","message":"count: 001","sort_field":10000}`,
		`{"id":0,"log.level":"info","message":"count: 002","sort_field":20000}`,
		`{"id":0,"log.level":"info","message":"count: 003","sort_field":30000}`,
		`{"id":0,"log.level":"info","message":"count: 004","sort_field":40000}`,
		`{"id":0,"log.level":"info","message":"count: 005","sort_field":50000}`,
		`{"id":0,"log.level":"info","message":"count: 006","sort_field":60000}`,
		`{"id":0,"log.level":"info","message":"count: 007","sort_field":70000}`,
		`{"id":0,"log.level":"info","message":"count: 008","sort_field":80000}`,
		`{"id":0,"log.level":"info","message":"count: 009","sort_field":90000}`,
		`{"id":100,"log.level":"info","message":"count: 100","sort_field":1000000}`,
		`{"id":100,"log.level":"info","message":"count: 101","sort_field":1010000}`,
		`{"id":100,"log.level":"info","message":"count: 102","sort_field":1020000}`,
		`{"id":100,"log.level":"info","message":"count: 103","sort_field":1030000}`,
		`{"id":100,"log.level":"info","message":"count: 104","sort_field":1040000}`,
		`{"id":100,"log.level":"info","message":"count: 105","sort_field":1050000}`,
		`{"id":100,"log.level":"info","message":"count: 106","sort_field":1060000}`,
		`{"id":100,"log.level":"info","message":"count: 107","sort_field":1070000}`,
		`{"id":100,"log.level":"info","message":"count: 108","sort_field":1080000}`,
		`{"id":100,"log.level":"info","message":"count: 109","sort_field":1090000}`,
		`{"id":200,"log.level":"info","message":"count: 200","sort_field":2000000}`,
		`{"id":200,"log.level":"info","message":"count: 201","sort_field":2010000}`,
		`{"id":200,"log.level":"info","message":"count: 202","sort_field":2020000}`,
		`{"id":200,"log.level":"info","message":"count: 203","sort_field":2030000}`,
		`{"id":200,"log.level":"info","message":"count: 204","sort_field":2040000}`,
		`{"id":200,"log.level":"info","message":"count: 205","sort_field":2050000}`,
		`{"id":200,"log.level":"info","message":"count: 206","sort_field":2060000}`,
		`{"id":200,"log.level":"info","message":"count: 207","sort_field":2070000}`,
		`{"id":200,"log.level":"info","message":"count: 208","sort_field":2080000}`,
		`{"id":200,"log.level":"info","message":"count: 209","sort_field":2090000}`,
		`{"id":300,"log.level":"info","message":"count: 300","sort_field":3000000}`,
		`{"id":300,"log.level":"info","message":"count: 301","sort_field":3010000}`,
		`{"id":300,"log.level":"info","message":"count: 302","sort_field":3020000}`,
		`{"id":300,"log.level":"info","message":"count: 303","sort_field":3030000}`,
		`{"id":300,"log.level":"info","message":"count: 304","sort_field":3040000}`,
		`{"id":300,"log.level":"info","message":"count: 305","sort_field":3050000}`,
		`{"id":300,"log.level":"info","message":"count: 306","sort_field":3060000}`,
		`{"id":300,"log.level":"info","message":"count: 307","sort_field":3070000}`,
		`{"id":300,"log.level":"info","message":"count: 308","sort_field":3080000}`,
		`{"id":300,"log.level":"info","message":"count: 309","sort_field":3090000}`,
		`{"id":400,"log.level":"info","message":"count: 400","sort_field":4000000}`,
		`{"id":400,"log.level":"info","message":"count: 401","sort_field":4010000}`,
		`{"id":400,"log.level":"info","message":"count: 402","sort_field":4020000}`,
		`{"id":400,"log.level":"info","message":"count: 403","sort_field":4030000}`,
		`{"id":400,"log.level":"info","message":"count: 404","sort_field":4040000}`,
		`{"id":400,"log.level":"info","message":"count: 405","sort_field":4050000}`,
		`{"id":400,"log.level":"info","message":"count: 406","sort_field":4060000}`,
		`{"id":400,"log.level":"info","message":"count: 407","sort_field":4070000}`,
		`{"id":400,"log.level":"info","message":"count: 408","sort_field":4080000}`,
		`{"id":400,"log.level":"info","message":"count: 409","sort_field":4090000}`,
	}

	if err := logp.ConfigureWithTypedOutput(defaultCfg, eventsCfg, "log.type", "event"); err != nil {
		t.Fatalf("could not configure logger: %s", err)
	}

	wg := sync.WaitGroup{}

	for i := 0; i < 500; i += 100 {
		wg.Add(1)
		go func(id int) {
			logger := logp.L().With("id", id)
			defer wg.Done()
			time.Sleep(time.Millisecond * 100)
			for j := 0; j < 10; j++ {
				logger.Infow(fmt.Sprintf("count: %03d", id+j), "sort_field", (id+j)*10000)
			}
		}(i)
	}

	wg.Wait()

	entries := takeAllLogsFromPath(t, tempDir)
	if len(expectedLines) != len(entries) {
		t.Fatalf("expecting %d lines got %d", len(expectedLines), len(entries))
	}

	sort.Slice(entries, func(i, j int) bool {
		t1 := entries[i]["sort_field"].(float64) //nolint: errcheck // We know it's a float64 and it is a test
		t2 := entries[j]["sort_field"].(float64) //nolint: errcheck // We know it's a float64 and it is a test
		return t1 < t2
	})

	// Now that the slice is sorted, remove some fields, including
	// the @timestamp we used to sort it
	for i := range entries {
		delete(entries[i], "@timestamp")
		delete(entries[i], "log.origin")
		delete(entries[i], "ecs.version")
		delete(entries[i], "service.name")
	}

	strEntries := []string{}
	for _, e := range entries {
		data, _ := json.Marshal(e)
		strEntries = append(strEntries, string(data))
	}

	for i := range strEntries {
		assert.JSONEq(t, strEntries[i], expectedLines[i], "Some log entries are different than expected")
	}

	// Get a logger and close it so the file descriptors are released.
	// This is specially important on Windows
	logp.L().Close()
}

func strField(key, val string) zapcore.Field {
	return zapcore.Field{Type: zapcore.StringType, Key: key, String: val}
}

func takeAllLogsFromPath(t *testing.T, path string) []map[string]any {
	entries := []map[string]any{}

	glob := filepath.Join(path, "*.ndjson")
	files, err := filepath.Glob(glob)
	if err != nil {
		t.Fatalf("cannot get files for glob '%s': %s", glob, err)
	}

	for _, filePath := range files {
		f, err := os.Open(filePath)
		if err != nil {
			t.Fatalf("cannot open file '%s': %s", filePath, err)
		}
		defer f.Close()

		sc := bufio.NewScanner(f)
		for sc.Scan() {
			m := map[string]any{}
			bytes := sc.Bytes()
			if err := json.Unmarshal(bytes, &m); err != nil {
				t.Fatalf("cannot unmarshal log entry: '%s', err: %s", string(bytes), err)
			}

			entries = append(entries, m)
		}
	}

	return entries
}
