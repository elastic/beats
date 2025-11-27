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

package javascript

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dop251/goja"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

func TestNew(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("with inline source", func(t *testing.T) {
		cfg, err := config.NewConfigFrom(map[string]any{
			"source": `function process(event) { event.Put("hello", "world"); }`,
		})
		require.NoError(t, err)

		p, err := New(cfg, logptest.NewTestingLogger(t, ""))
		require.NoError(t, err)

		evt := &beat.Event{Fields: mapstr.M{}}
		result, err := p.Run(evt)
		require.NoError(t, err)
		require.NotNil(t, result)

		v, _ := result.GetValue("hello")
		assert.Equal(t, "world", v)
	})

	t.Run("with file", func(t *testing.T) {
		file := writeFile(t, tmpDir, "processor.js", `function process(event) { event.Put("from_file", true); }`)

		cfg, err := config.NewConfigFrom(map[string]any{"file": file})
		require.NoError(t, err)

		p, err := New(cfg, logptest.NewTestingLogger(t, ""))
		require.NoError(t, err)

		evt := &beat.Event{Fields: mapstr.M{}}
		result, err := p.Run(evt)
		require.NoError(t, err)

		v, _ := result.GetValue("from_file")
		assert.Equal(t, true, v)
	})

	t.Run("with multiple files", func(t *testing.T) {
		utilFile := writeFile(t, tmpDir, "util.js", "var multiplier = 2;")
		mainFile := writeFile(t, tmpDir, "main.js", `function process(event) { event.Put("multiplier", multiplier); }`)

		cfg, err := config.NewConfigFrom(map[string]any{"files": []string{utilFile, mainFile}})
		require.NoError(t, err)

		p, err := New(cfg, logptest.NewTestingLogger(t, ""))
		require.NoError(t, err)

		evt := &beat.Event{Fields: mapstr.M{}}
		result, err := p.Run(evt)
		require.NoError(t, err)

		v, _ := result.GetValue("multiplier")
		assert.Equal(t, int64(2), v)
	})

	t.Run("with glob pattern", func(t *testing.T) {
		globDir := filepath.Join(tmpDir, "glob")
		err := os.Mkdir(globDir, 0o755)
		require.NoError(t, err)

		// Create multiple files that should be matched by the glob
		writeFile(t, globDir, "a_utils.js", "var fromGlob = true;")
		writeFile(t, globDir, "b_main.js", `function process(event) { event.Put("from_glob", fromGlob); }`)

		cfg, err := config.NewConfigFrom(map[string]any{"file": filepath.Join(globDir, "*.js")})
		require.NoError(t, err)

		p, err := New(cfg, logptest.NewTestingLogger(t, ""))
		require.NoError(t, err)

		evt := &beat.Event{Fields: mapstr.M{}}
		result, err := p.Run(evt)
		require.NoError(t, err)

		// Verify both files were loaded (b_main.js uses variable from a_util.js)
		v, _ := result.GetValue("from_glob")
		assert.Equal(t, true, v)
	})

	t.Run("with tag", func(t *testing.T) {
		cfg, err := config.NewConfigFrom(map[string]any{
			"tag":    "my-processor",
			"source": `function process(event) { return event; }`,
		})
		require.NoError(t, err)

		p, err := New(cfg, logptest.NewTestingLogger(t, ""))
		require.NoError(t, err)

		assert.Contains(t, p.String(), "id=my-processor")
	})

	t.Run("with invalid config", func(t *testing.T) {
		cfg, err := config.NewConfigFrom(map[string]any{})
		require.NoError(t, err)

		_, err = New(cfg, logptest.NewTestingLogger(t, ""))
		require.ErrorContains(t, err, "javascript must be defined")
	})

	t.Run("with syntax error", func(t *testing.T) {
		cfg, err := config.NewConfigFrom(map[string]any{
			"source": `function process(event { invalid syntax`,
		})
		require.NoError(t, err)

		_, err = New(cfg, logptest.NewTestingLogger(t, ""))
		require.ErrorAs(t, err, new(*goja.CompilerSyntaxError))
	})

	t.Run("with missing process function", func(t *testing.T) {
		cfg, err := config.NewConfigFrom(map[string]any{
			"source": `function notProcess(event) { return event; }`,
		})
		require.NoError(t, err)

		_, err = New(cfg, logptest.NewTestingLogger(t, ""))
		require.ErrorContains(t, err, "process function not found")
	})

	t.Run("file not found", func(t *testing.T) {
		cfg, err := config.NewConfigFrom(map[string]any{"file": filepath.Join(tmpDir, "nonexistent.js")})
		require.NoError(t, err)

		_, err = New(cfg, logptest.NewTestingLogger(t, ""))
		require.ErrorContains(t, err, "no such file or directory")
	})

	t.Run("no sources found with glob", func(t *testing.T) {
		cfg, err := config.NewConfigFrom(map[string]any{"file": filepath.Join(tmpDir, "nomatch", "*.js")})
		require.NoError(t, err)

		_, err = New(cfg, logptest.NewTestingLogger(t, ""))
		require.ErrorContains(t, err, "no sources were found")
	})
}

func TestRunWithStats(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	reg := monitoring.NewRegistry()

	t.Run("tracks successful execution time", func(t *testing.T) {
		p, err := NewFromConfig(Config{
			Tag:    "timing-test",
			Source: `function process(event) { return event; }`,
		}, reg, logger)
		require.NoError(t, err)

		evt := &beat.Event{Fields: mapstr.M{}}
		_, err = p.Run(evt)
		require.NoError(t, err)

		jp, ok := p.(*jsProcessor)
		require.True(t, ok, "expected *jsProcessor type")
		assert.NotNil(t, jp.stats)
		assert.Equal(t, int64(1), jp.stats.processTime.Count())
	})

	t.Run("increments exceptions counter", func(t *testing.T) {
		p, err := NewFromConfig(Config{
			Tag:            "exception-counter",
			Source:         `function process(event) { throw "test error"; }`,
			TagOnException: "_error",
		}, reg, logger)
		require.NoError(t, err)

		evt := &beat.Event{Fields: mapstr.M{}}
		_, err = p.Run(evt)
		require.ErrorContains(t, err, "failed in processor.javascript")

		jp, ok := p.(*jsProcessor)
		require.True(t, ok, "expected *jsProcessor type")
		assert.NotNil(t, jp.stats)
		assert.Equal(t, int64(1), jp.stats.exceptions.Get())
	})
}

func writeFile(t *testing.T, dir, name, contents string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	err := os.WriteFile(path, []byte(contents), 0o644)
	require.NoErrorf(t, err, "failed to write to file %s", path)
	return path
}
