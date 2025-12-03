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
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/rcrowley/go-metrics"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/monitoring/adapter"
	"github.com/elastic/elastic-agent-libs/paths"
)

type jsProcessor struct {
	Config
	sessionPool *sessionPool
	sourceFile  string
	stats       *processorStats
	logger      *logp.Logger
}

// New constructs a new JavaScript processor.
func New(c *config.C, log *logp.Logger) (beat.Processor, error) {
	conf := defaultConfig()
	if err := c.Unpack(&conf); err != nil {
		return nil, err
	}

	return NewFromConfig(conf, monitoring.Default, log)
}

// NewFromConfig constructs a new JavaScript processor from the given config
// object. It loads the sources, compiles them, and validates the entry point.
// For inline sources, initialization happens immediately. For file-based sources,
// initialization is deferred until SetPaths is called.
func NewFromConfig(c Config, reg *monitoring.Registry, logger *logp.Logger) (beat.Processor, error) {
	err := c.Validate()
	if err != nil {
		return nil, err
	}

	processor := &jsProcessor{
		Config: c,
		logger: logger,
		stats:  getStats(c.Tag, reg, logger),
	}

	// For inline sources, we can initialize immediately.
	// For file-based sources, we defer initialization until SetPaths is called.
	if c.Source != "" {
		const inlineSourceFile = "inline.js"

		err = processor.compile(inlineSourceFile, c.Source)
		if err != nil {
			return nil, err
		}
	}

	return processor, nil
}

// SetPaths initializes the processor with the provided paths configuration.
// This method must be called before the processor can be used for file-based sources.
func (p *jsProcessor) SetPaths(path *paths.Path) error {
	if p.Source != "" {
		return nil // inline source already set
	}

	var sourceFile string
	var sourceCode string
	var err error

	switch {
	case p.File != "":
		sourceFile, sourceCode, err = loadSources(path, p.File)
	case len(p.Files) > 0:
		sourceFile, sourceCode, err = loadSources(path, p.Files...)
	}
	if err != nil {
		return annotateError(p.Tag, err)
	}

	return p.compile(sourceFile, sourceCode)
}

// loadSources loads javascript source from files using the provided paths.
func loadSources(pathConfig *paths.Path, files ...string) (string, string, error) {
	buf := new(bytes.Buffer)

	readFile := func(path string) error {
		if common.IsStrictPerms() {
			if err := common.OwnerHasExclusiveWritePerms(path); err != nil {
				return err
			}
		}

		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open file %v: %w", path, err)
		}
		defer f.Close()

		if _, err = io.Copy(buf, f); err != nil {
			return fmt.Errorf("failed to read file %v: %w", path, err)
		}
		return nil
	}

	sources := make([]string, 0, len(files))
	for _, filePath := range files {
		filePath = pathConfig.Resolve(paths.Config, filePath)

		if hasMeta(filePath) {
			matches, err := filepath.Glob(filePath)
			if err != nil {
				return "", "", err
			}
			sources = append(sources, matches...)
		} else {
			sources = append(sources, filePath)
		}
	}

	if len(sources) == 0 {
		return "", "", fmt.Errorf("no sources were found in %v",
			strings.Join(files, ", "))
	}

	for _, name := range sources {
		if err := readFile(name); err != nil {
			return "", "", err
		}
	}

	return strings.Join(sources, ";"), buf.String(), nil
}

func annotateError(id string, err error) error {
	if err == nil {
		return nil
	}
	if id != "" {
		return fmt.Errorf("failed in processor.javascript with id=%v: %w", id, err)
	}
	return fmt.Errorf("failed in processor.javascript: %w", err)
}

func (p *jsProcessor) compile(sourceFile, sourceCode string) error {
	// Validate processor source code.
	prog, err := goja.Compile(sourceFile, sourceCode, true)
	if err != nil {
		return err
	}

	pool, err := newSessionPool(prog, p.Config, p.logger)
	if err != nil {
		return annotateError(p.Tag, err)
	}

	p.sessionPool = pool
	p.sourceFile = sourceFile
	return nil
}

// Run executes the processor on the given it event. It invokes the
// process function defined in the JavaScript source.
func (p *jsProcessor) Run(event *beat.Event) (*beat.Event, error) {
	if p.sessionPool == nil {
		return event, fmt.Errorf("javascript processor not initialized: SetPaths must be called for file-based sources")
	}

	s := p.sessionPool.Get()
	defer p.sessionPool.Put(s)

	var rtn *beat.Event
	var err error

	if p.stats == nil {
		rtn, err = s.runProcessFunc(event)
	} else {
		rtn, err = p.runWithStats(s, event)
	}
	return rtn, annotateError(p.Tag, err)
}

func (p *jsProcessor) runWithStats(s *session, event *beat.Event) (*beat.Event, error) {
	start := time.Now()
	event, err := s.runProcessFunc(event)
	elapsed := time.Since(start)

	p.stats.processTime.Update(int64(elapsed))
	if err != nil {
		p.stats.exceptions.Inc()
	}
	return event, err
}

func (p *jsProcessor) String() string {
	return "script=[type=javascript, id=" + p.Tag + ", sources=" + p.sourceFile + "]"
}

// hasMeta reports whether path contains any of the magic characters
// recognized by Match/Glob.
func hasMeta(path string) bool {
	magicChars := `*?[`
	if runtime.GOOS != "windows" {
		magicChars = `*?[\`
	}
	return strings.ContainsAny(path, magicChars)
}

type processorStats struct {
	exceptions  *monitoring.Int
	processTime metrics.Sample
}

func getStats(id string, reg *monitoring.Registry, logger *logp.Logger) *processorStats {
	if id == "" || reg == nil {
		return nil
	}

	namespace := logName + "." + id
	processorReg := reg.GetRegistry(namespace)
	if processorReg != nil {
		// If a module is reloaded then the namespace could already exist.
		_ = processorReg.Clear()
	} else {
		processorReg = reg.GetOrCreateRegistry(namespace, monitoring.DoNotReport)
	}

	stats := &processorStats{
		exceptions:  monitoring.NewInt(processorReg, "exceptions"),
		processTime: metrics.NewUniformSample(2048),
	}
	_ = adapter.NewGoMetrics(processorReg, "histogram", logger, adapter.Accept).
		Register("process_time", metrics.NewHistogram(stats.processTime))

	return stats
}
