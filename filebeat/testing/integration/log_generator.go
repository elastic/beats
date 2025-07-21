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
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/gofrs/uuid/v5"
	"github.com/klauspost/compress/gzip"
)

// LogGenerator used for generating log files
type LogGenerator interface {
	// GenerateLine generates a single line for a log file.
	// Expected no new line character at the end.
	GenerateLine(filename string, index int) string
	// FileExtension defines the extension of the new file where
	// the generated lines are written.
	FileExtension() string
}

// NewPlainTextGenerator creates is a simple plain text generator.
//
// It's using the given message prefix following by the filename
// and the line number, e.g. `filename:128`
func NewPlainTextGenerator(prefix string) LogGenerator {
	return plainTextGenerator{
		prefix: prefix,
	}
}

type plainTextGenerator struct {
	prefix string
}

func (g plainTextGenerator) GenerateLine(filename string, index int) string {
	return fmt.Sprintf("%s %s:%d", g.prefix, filepath.Base(filename), index)
}

func (g plainTextGenerator) FileExtension() string {
	return ".log"
}

// NewJSONGenerator creates a JSON log line generator.
// Forms a JSON object with a message
// prefixed by the given prefix and followed by the filename
// and the line number, e.g. `filename:128`
func NewJSONGenerator(prefix string) LogGenerator {
	return jsonGenerator{
		prefix: prefix,
	}
}

type jsonGenerator struct {
	prefix string
}

func (g jsonGenerator) GenerateLine(filename string, index int) string {
	message := fmt.Sprintf("%s %s:%d", g.prefix, filepath.Base(filename), index)

	line := struct{ Message string }{Message: message}
	bytes, _ := json.Marshal(line)
	return string(bytes)
}

func (g jsonGenerator) FileExtension() string {
	return ".ndjson"
}

// GenerateLogFiles generate given amount of files with given
// amount of lines in them.
//
// Returns the path value to put in the Filebeat configuration and
// filenames for all created files.
func GenerateLogFiles(t *testing.T, files, lines int, generator LogGenerator) (path string, filenames []string) {
	return generateLogFiles(
		t, files, lines, generator, filenames, GenerateLogFile)
}

// GenerateGZIPLogFiles is the same as GenerateLogFiles, but the files produced
// are GZIP files.
func GenerateGZIPLogFiles(t *testing.T, files, lines int, generator LogGenerator) (path string, filenames []string) {
	return generateLogFiles(
		t, files, lines, generator, filenames, GenerateGZIPLogFile)
}

func generateLogFiles(
	t *testing.T,
	files int,
	lines int,
	generator LogGenerator,
	filenames []string,
	gen func(t *testing.T, filename string, lines int, generator LogGenerator)) (string, []string) {

	t.Logf("generating %d log files with %d lines each...", files, lines)
	logsPath := filepath.Join(t.TempDir(), "logs")
	err := os.MkdirAll(logsPath, 0777)
	if err != nil {
		t.Fatalf("failed to create a directory for logs %q: %s", logsPath, err)
		return "", nil
	}

	filenames = make([]string, 0, files)
	for i := 0; i < files; i++ {
		id, err := uuid.NewV4()
		if err != nil {
			t.Fatalf("failed to generate a unique filename: %s", err)
			return "", nil
		}
		filename := filepath.Join(logsPath, id.String()+generator.FileExtension())
		filenames = append(filenames, filename)
		gen(t, filename, lines, generator)
	}

	t.Logf("finished generating %d log files with %d lines each", files, lines)

	return filepath.Join(logsPath, "*"+generator.FileExtension()), filenames
}

// GenerateLogFile generates a single log file with the given full
// filename, amount of lines using the given generator
// to create each line.
func GenerateLogFile(t *testing.T, filename string, lines int, generator LogGenerator) {
	file, err := os.Create(filename)
	if err != nil {
		t.Fatalf("failed to create a log file: %q", filename)
		return
	}
	defer file.Close()

	writeLines(t, file, filename, lines, generator)
}

// GenerateGZIPLogFile generates a single gzip-compressed log file with the
// given filename, lines, and generator. The file content is identical to the
// one produced by GenerateLogFile, but compressed using gzip.
func GenerateGZIPLogFile(t *testing.T, filename string, lines int, generator LogGenerator) {
	file, err := os.Create(filename)
	if err != nil {
		t.Fatalf("failed to create a gzip log file: %q", filename)
		return
	}
	defer file.Close()

	gw := gzip.NewWriter(file)
	defer gw.Close()

	writeLines(t, gw, filename, lines, generator)
}

// writeLines writes generated lines to the provided writer.
// It is shared between GenerateLogFile and GenerateGZIPLogFile to
// avoid duplicating the core writing logic.
func writeLines(t *testing.T, w io.Writer, filename string, lines int, generator LogGenerator) {
	for i := 1; i <= lines; i++ {
		line := generator.GenerateLine(filename, i) + "\n"
		if _, err := w.Write([]byte(line)); err != nil {
			t.Fatalf("cannot write a generated log line to %s: %s", filename, err)
			return
		}
	}
}
