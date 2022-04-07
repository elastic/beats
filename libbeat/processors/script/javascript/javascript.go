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
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/pkg/errors"
	"github.com/rcrowley/go-metrics"

	"github.com/elastic/beats/v8/libbeat/beat"
	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/monitoring"
	"github.com/elastic/beats/v8/libbeat/monitoring/adapter"
	"github.com/elastic/beats/v8/libbeat/paths"
	"github.com/elastic/beats/v8/libbeat/processors"
)

type jsProcessor struct {
	Config
	sessionPool *sessionPool
	sourceProg  *goja.Program
	sourceFile  string
	stats       *processorStats
}

// New constructs a new Javascript processor.
func New(c *common.Config) (processors.Processor, error) {
	conf := defaultConfig()
	if err := c.Unpack(&conf); err != nil {
		return nil, err
	}

	return NewFromConfig(conf, monitoring.Default)
}

// NewFromConfig constructs a new Javascript processor from the given config
// object. It loads the sources, compiles them, and validates the entry point.
func NewFromConfig(c Config, reg *monitoring.Registry) (processors.Processor, error) {
	err := c.Validate()
	if err != nil {
		return nil, err
	}

	var sourceFile string
	var sourceCode []byte

	switch {
	case c.Source != "":
		sourceFile = "inline.js"
		sourceCode = []byte(c.Source)
	case c.File != "":
		sourceFile, sourceCode, err = loadSources(c.File)
	case len(c.Files) > 0:
		sourceFile, sourceCode, err = loadSources(c.Files...)
	}
	if err != nil {
		return nil, annotateError(c.Tag, err)
	}

	// Validate processor source code.
	prog, err := goja.Compile(sourceFile, string(sourceCode), true)
	if err != nil {
		return nil, err
	}

	pool, err := newSessionPool(prog, c)
	if err != nil {
		return nil, annotateError(c.Tag, err)
	}

	return &jsProcessor{
		Config:      c,
		sessionPool: pool,
		sourceProg:  prog,
		sourceFile:  sourceFile,
		stats:       getStats(c.Tag, reg),
	}, nil
}

// loadSources loads javascript source from files.
func loadSources(files ...string) (string, []byte, error) {
	var sources []string
	buf := new(bytes.Buffer)

	readFile := func(path string) error {
		if common.IsStrictPerms() {
			if err := common.OwnerHasExclusiveWritePerms(path); err != nil {
				return err
			}
		}

		f, err := os.Open(path)
		if err != nil {
			return errors.Wrapf(err, "failed to open file %v", path)
		}
		defer f.Close()

		if _, err = io.Copy(buf, f); err != nil {
			return errors.Wrapf(err, "failed to read file %v", path)
		}
		return nil
	}

	for _, filePath := range files {
		filePath = paths.Resolve(paths.Config, filePath)

		if hasMeta(filePath) {
			matches, err := filepath.Glob(filePath)
			if err != nil {
				return "", nil, err
			}
			sources = append(sources, matches...)
		} else {
			sources = append(sources, filePath)
		}
	}

	if len(sources) == 0 {
		return "", nil, errors.Errorf("no sources were found in %v",
			strings.Join(files, ", "))
	}

	for _, name := range sources {
		if err := readFile(name); err != nil {
			return "", nil, err
		}
	}

	return strings.Join(sources, ";"), buf.Bytes(), nil
}

func annotateError(id string, err error) error {
	if err == nil {
		return nil
	}
	if id != "" {
		return errors.Wrapf(err, "failed in processor.javascript with id=%v", id)
	}
	return errors.Wrap(err, "failed in processor.javascript")
}

// Run executes the processor on the given it event. It invokes the
// process function defined in the Javascript source.
func (p *jsProcessor) Run(event *beat.Event) (*beat.Event, error) {
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

func getStats(id string, reg *monitoring.Registry) *processorStats {
	if id == "" || reg == nil {
		return nil
	}

	namespace := logName + "." + id
	processorReg := reg.GetRegistry(namespace)
	if processorReg != nil {
		// If a module is reloaded then the namespace could already exist.
		processorReg.Clear()
	} else {
		processorReg = reg.NewRegistry(namespace, monitoring.DoNotReport)
	}

	stats := &processorStats{
		exceptions:  monitoring.NewInt(processorReg, "exceptions"),
		processTime: metrics.NewUniformSample(2048),
	}
	adapter.NewGoMetrics(processorReg, "histogram", adapter.Accept).
		Register("process_time", metrics.NewHistogram(stats.processTime))

	return stats
}
