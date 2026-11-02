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

//go:build !integration

package log

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"

	"github.com/elastic/beats/v7/filebeat/channel"
	"github.com/elastic/beats/v7/filebeat/input/file"
	"github.com/elastic/beats/v7/filebeat/input/inputtest"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/reader"
	"github.com/elastic/beats/v7/libbeat/reader/readfile"
	"github.com/elastic/beats/v7/libbeat/reader/readfile/encoding"
	"github.com/elastic/beats/v7/libbeat/tests/resources"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestReadLine(t *testing.T) {
	absPath, err := filepath.Abs("../../tests/files/logs/")
	// All files starting with tmp are ignored
	logFile := absPath + "/tmp" + strconv.Itoa(rand.Int()) + ".log"

	assert.NotNil(t, absPath)
	assert.NoError(t, err)

	if err != nil {
		t.Fatalf("Error creating the absolute path: %s", absPath)
	}

	file, err := os.Create(logFile)
	defer file.Close()
	defer os.Remove(logFile)

	assert.NoError(t, err)
	assert.NotNil(t, file)

	firstLineString := "9Characte\n"
	secondLineString := "This is line 2\n"

	length, err := file.WriteString(firstLineString)
	assert.NoError(t, err)
	assert.NotNil(t, length)

	length, err = file.WriteString(secondLineString)
	assert.NoError(t, err)
	assert.NotNil(t, length)

	file.Sync()

	// Open file for reading
	readFile, err := os.Open(logFile)
	defer readFile.Close()
	assert.NoError(t, err)

	source := File{File: readFile}

	h := Harvester{
		logger: logptest.NewTestingLogger(t, "harvester"),
		config: config{
			LogConfig: LogConfig{
				CloseInactive: 500 * time.Millisecond,
				Backoff:       100 * time.Millisecond,
				MaxBackoff:    1 * time.Second,
				BackoffFactor: 2,
			},
			BufferSize:     100,
			MaxBytes:       1000,
			LineTerminator: readfile.LineFeed,
		},
		source: source,
	}

	var ok bool
	h.encodingFactory, ok = encoding.FindEncoding(h.config.Encoding)
	assert.True(t, ok)

	h.encoding, err = h.encodingFactory(readFile)
	assert.NoError(t, err)

	r, err := h.newLogFileReader()
	assert.NoError(t, err)

	// Read third line
	_, text, bytesread, _, err := readLine(r)
	t.Logf("received line: '%s'\n", text)
	assert.NoError(t, err)
	assert.Equal(t, text, firstLineString[0:len(firstLineString)-1])
	assert.Equal(t, bytesread, len(firstLineString))

	// read second line
	_, text, bytesread, _, err = readLine(r)
	t.Logf("received line: '%s'\n", text)
	assert.Equal(t, text, secondLineString[0:len(secondLineString)-1])
	assert.Equal(t, bytesread, len(secondLineString))
	assert.NoError(t, err)

	// Read third line, which doesn't exist
	_, text, bytesread, _, err = readLine(r)
	t.Logf("received line: '%s'\n", text)
	assert.Equal(t, "", text)
	assert.Equal(t, bytesread, 0)
	assert.Equal(t, err, ErrInactive)
}

// readLine reads a full line into buffer and returns it.
// In case of partial lines, readLine does return an error and an empty string
// This could potentially be improved / replaced by https://github.com/elastic/beats/libbeat/tree/master/common/streambuf
func readLine(reader reader.Reader) (time.Time, string, int, mapstr.M, error) {
	message, err := reader.Next()

	// Full line read to be returned
	if message.Bytes != 0 && err == nil {
		return message.Ts, string(message.Content), message.Bytes, message.Fields, err
	}

	return time.Time{}, "", 0, nil, err
}

// TestHarvesterStopWhileReadingMultiline checks that Stop returns and leaves no
// goroutines behind when the harvester is in the middle of a multiline backlog.
func TestHarvesterStopWhileReadingMultiline(t *testing.T) {
	goroutines := resources.NewGoroutinesChecker()
	defer goroutines.Check(t)

	var sb strings.Builder
	for i := range 1_000 {
		fmt.Fprintf(&sb, "event %d\n", i)
		for range 50 {
			sb.WriteString("  continuation line\n")
		}
	}
	path := writeLogFile(t, sb.String())
	logger := logptest.NewTestingLogger(t, "")
	settings := map[string]any{
		"multiline.pattern": `^\s`,
		"multiline.match":   "after",
	}

	// Each trial stops the harvester at a different point while it reads.
	for trial := range 50 {
		// Large enough for every event in the file, so OnEvent never blocks.
		events := make(chan beat.Event, 1024)
		h := newTestHarvester(t, logger, path, settings, NewEventCapturer(events))

		go func() { _ = h.Run() }()
		select {
		case <-events:
		case <-time.After(10 * time.Second):
			require.FailNowf(t, "harvester published no event", "trial %d", trial)
		}

		stopped := make(chan struct{})
		go func() {
			h.Stop()
			close(stopped)
		}()
		select {
		case <-stopped:
		case <-time.After(10 * time.Second):
			require.FailNowf(t, "Harvester.Stop did not return", "trial %d", trial)
		}
	}
}

// TestHarvesterRunClosedReader checks that Run treats a closed reader as a normal stop.
func TestHarvesterRunClosedReader(t *testing.T) {
	cases := []struct {
		name string
		err  error
	}{
		{name: "log file closed", err: ErrClosed},
		{name: "line reader closed", err: os.ErrClosed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			logger, observed := logptest.NewTestingLoggerWithObserver(t, "")
			path := writeLogFile(t, "line\n")
			h := newTestHarvester(t, logger, path, map[string]any{}, inputtest.Outlet{})
			h.reader = errReader{err: tc.err}

			assert.NoError(t, h.Run())
			assert.Equal(t, 1, observed.FilterMessage("Reader was closed. Closing.").Len(),
				"Run must log that the reader was closed")
			assert.Empty(t, observed.FilterLevelExact(zapcore.ErrorLevel).All(),
				"Run must not log a closed reader as an error")
		})
	}
}

func writeLogFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.log")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func newTestHarvester(t *testing.T, logger *logp.Logger, path string, settings map[string]any, out channel.Outleter) *Harvester {
	t.Helper()
	fi, err := os.Stat(path)
	require.NoError(t, err)
	identifier, err := file.NewStateIdentifier(nil, logger)
	require.NoError(t, err)

	cfg := conf.MustNewConfigFrom(settings)
	require.NoError(t, cfg.SetString("paths", 0, path))
	h, err := NewHarvester(
		logger, cfg,
		file.NewState(fi, path, "log", nil, identifier),
		file.NewStates(logger),
		func(file.State) bool { return true },
		func() channel.Outleter { return out },
	)
	require.NoError(t, err)
	require.NoError(t, h.Setup())
	return h
}

type errReader struct{ err error }

func (r errReader) Next() (reader.Message, error) { return reader.Message{}, r.err }

func (errReader) Close() error { return nil }
