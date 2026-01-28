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
//
// This file was contributed to by generative AI

package testhelpers

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit"
	"github.com/gofrs/uuid/v5"
)

// LineGenerator defines how a line should be rendered and what extension
// the generated file should use.
type LineGenerator interface {
	// GenerateLine generates a single line for a log file.
	// Expected no new line character at the end.
	GenerateLine(filename string, index int) string
	// FileExtension defines the extension of the new file where
	// the generated lines are written.
	FileExtension() string
}

// GeneratorFunc is a helper adapter to allow ordinary functions to be used
// as line generators.
type GeneratorFunc struct {
	Ext string
	Fn  func(filename string, index int) string
}

// GenerateLine generates a single line for a log file.
func (g GeneratorFunc) GenerateLine(filename string, index int) string {
	if g.Fn == nil {
		return ""
	}
	return g.Fn(filename, index)
}

// FileExtension returns the extension for the log file.
func (g GeneratorFunc) FileExtension() string {
	if g.Ext == "" {
		return ".log"
	}
	return g.Ext
}

// NewPlainTextGenerator renders "<prefix> <basename>:<line>" messages.
func NewPlainTextGenerator(prefix string) LineGenerator {
	return GeneratorFunc{
		Ext: ".log",
		Fn: func(filename string, index int) string {
			return fmt.Sprintf("%s %s:%d", prefix, filepath.Base(filename), index)
		},
	}
}

// NewJSONGenerator renders {"message":"<prefix> <basename>:<line>"} entries.
func NewJSONGenerator(prefix string) LineGenerator {
	type message struct {
		Message string `json:"message"`
	}

	return GeneratorFunc{
		Ext: ".ndjson",
		Fn: func(filename string, index int) string {
			data := message{
				Message: fmt.Sprintf("%s %s:%d", prefix, filepath.Base(filename), index),
			}
			encoded, _ := json.Marshal(data)
			return string(encoded)
		},
	}
}

// NewTimestampGenerator renders lines compatible with the legacy helper that
// prefixed RFC3339 timestamps followed by a counter.
func NewTimestampGenerator(prefix string) LineGenerator {
	timestamp := prefix
	if timestamp == "" {
		timestamp = time.Now().Format(time.RFC3339)
		if len(timestamp) != len(time.RFC3339) {
			if len(timestamp) < len(time.RFC3339) {
				padding := len(time.RFC3339) - len(timestamp)
				for range padding {
					timestamp += "-"
				}
			} else {
				timestamp = timestamp[:len(time.RFC3339)]
			}
		}
	}

	return GeneratorFunc{
		Ext: ".log",
		Fn: func(_ string, index int) string {
			// legacy helper printed counters starting at zero
			return fmt.Sprintf("%s           %13d", timestamp, index-1)
		},
	}
}

// CompressionMode determines how generated content is stored on disk.
type CompressionMode int

const (
	// CompressionNone writes plain files.
	CompressionNone CompressionMode = iota
	// CompressionGzip writes GZIP-compressed files in a single operation.
	CompressionGzip
)

// GenerateLogFiles creates the requested number of files with the provided
// line generator. It returns the glob pattern for Filebeat configuration and
// the slice of generated filenames.
func GenerateLogFiles(t *testing.T, files, lines int, generator LineGenerator) (string, []string) {
	t.Helper()
	return generateLogFiles(t, files, lines, generator, CompressionNone)
}

// GenerateGZIPLogFiles behaves like GenerateLogFiles but produces GZIP files.
func GenerateGZIPLogFiles(t *testing.T, files, lines int, generator LineGenerator) (string, []string) {
	t.Helper()
	return generateLogFiles(t, files, lines, generator, CompressionGzip)
}

func generateLogFiles(
	t *testing.T,
	files int,
	lines int,
	generator LineGenerator,
	compression CompressionMode,
) (string, []string) {
	t.Helper()

	if generator == nil {
		generator = NewTimestampGenerator("")
	}

	logsPath := filepath.Join(t.TempDir(), "logs")
	if err := os.MkdirAll(logsPath, 0o755); err != nil {
		t.Fatalf("failed to create log directory %q: %v", logsPath, err)
	}

	paths := make([]string, 0, files)
	for i := 0; i < files; i++ {
		id, err := uuid.NewV4()
		if err != nil {
			t.Fatalf("failed to create unique filename: %v", err)
		}
		filename := filepath.Join(logsPath, id.String()+generator.FileExtension())
		writeLogFile(t, filename, lines, false, generator, compression)
		paths = append(paths, filename)
	}

	glob := filepath.Join(logsPath, "*"+generator.FileExtension())
	return glob, paths
}

// WriteLogFile writes count lines to path. When append is true, it appends
// to an existing file preserving its contents. The optional prefix parameter
// mirrors the legacy helper behavior.
func WriteLogFile(t *testing.T, path string, count int, append bool, prefix ...string) {
	t.Helper()
	generator := NewTimestampGenerator("")
	if len(prefix) > 0 {
		generator = NewTimestampGenerator(strings.Join(prefix, ""))
	}
	writeLogFile(t, path, count, append, generator, CompressionNone)
}

// WriteLogFileWithGenerator writes count lines using the provided generator.
func WriteLogFileWithGenerator(t *testing.T, path string, count int, append bool, generator LineGenerator) {
	t.Helper()
	if generator == nil {
		generator = NewTimestampGenerator("")
	}
	writeLogFile(t, path, count, append, generator, CompressionNone)
}

// AppendLogLines appends the requested amount of lines to an existing file.
func AppendLogLines(t *testing.T, path string, count int, generator LineGenerator) {
	t.Helper()
	WriteLogFileWithGenerator(t, path, count, true, generator)
}

// WriteNLogFiles creates nFiles files under baseDir/logs with nLines each and
// returns the directory containing them.
func WriteNLogFiles(t *testing.T, baseDir string, nFiles, nLines int) string {
	t.Helper()

	if baseDir == "" {
		baseDir = t.TempDir()
	}

	basePath := filepath.Join(baseDir, "logs")
	if err := os.MkdirAll(basePath, 0o755); err != nil {
		t.Fatalf("cannot create folder to store logs: %v", err)
	}

	generator := GeneratorFunc{
		Ext: ".log",
		Fn: func(string, int) string {
			return gofakeit.HackerPhrase()
		},
	}

	for fCount := 0; fCount < nFiles; fCount++ {
		path := filepath.Join(basePath, fmt.Sprintf("%06d.log", fCount))
		writeLogFile(t, path, nLines, false, generator, CompressionNone)
	}

	return basePath
}

// StartAppendingLogFile starts a goroutine that appends a new line at the given
// interval. The returned stop function terminates the writer and closes the file.
func StartAppendingLogFile(t *testing.T, path string, interval time.Duration, generator LineGenerator) (stop func()) {
	t.Helper()

	if generator == nil {
		generator = NewTimestampGenerator("")
	}

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("could not create log file %q: %v", path, err)
	}

	var once sync.Once
	done := make(chan struct{})
	ticker := time.NewTicker(interval)

	go func() {
		defer file.Close()
		for idx := 1; ; idx++ {
			select {
			case <-ticker.C:
				line := generator.GenerateLine(path, idx) + "\n"
				if _, err := file.WriteString(line); err != nil {
					t.Errorf("could not write data to %q: %v", path, err)
					return
				}
				if err := file.Sync(); err != nil {
					t.Errorf("could not sync file %q: %v", path, err)
					return
				}
			case <-done:
				return
			}
		}
	}()

	return func() {
		once.Do(func() {
			close(done)
			ticker.Stop()
		})
	}
}

func writeLogFile(
	t *testing.T,
	filename string,
	lines int,
	append bool,
	generator LineGenerator,
	compression CompressionMode,
) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		t.Fatalf("failed to create directory for %q: %v", filename, err)
	}

	flag := os.O_CREATE | os.O_WRONLY
	if append {
		flag |= os.O_APPEND
	} else {
		flag |= os.O_TRUNC
	}

	file, err := os.OpenFile(filename, flag, 0o644)
	if err != nil {
		t.Fatalf("failed to open log file %q: %v", filename, err)
	}
	defer file.Close()

	switch compression {
	case CompressionNone:
		writeLines(t, file, filename, lines, generator)
	case CompressionGzip:
		if append {
			t.Fatalf("cannot append to gzip compressed file %q", filename)
		}
		gw := gzip.NewWriter(file)
		writeLines(t, gw, filename, lines, generator)
		if err := gw.Close(); err != nil {
			t.Fatalf("failed to finish gzip writer for %q: %v", filename, err)
		}
	default:
		t.Fatalf("unknown compression mode %d", compression)
	}
}

func writeLines(t *testing.T, w io.Writer, filename string, lines int, generator LineGenerator) {
	t.Helper()
	for i := 1; i <= lines; i++ {
		line := generator.GenerateLine(filename, i) + "\n"
		if _, err := w.Write([]byte(line)); err != nil {
			t.Fatalf("failed to write line %d to %q: %v", i, filename, err)
		}
	}
}
