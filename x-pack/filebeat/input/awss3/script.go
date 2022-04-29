// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/dop251/goja"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/paths"
)

type script struct {
	scriptConfig
	sessionPool *sessionPool
	sourceProg  *goja.Program
	sourceFile  string
}

// newScriptFromConfig constructs a new Javascript script from the given config
// object. It loads the sources, compiles them, and validates the entry point.
func newScriptFromConfig(log *logp.Logger, c *scriptConfig) (*script, error) {
	if c == nil {
		return nil, nil
	}
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
		return nil, err
	}

	// Validate processor source code.
	prog, err := goja.Compile(sourceFile, string(sourceCode), true)
	if err != nil {
		return nil, err
	}

	pool, err := newSessionPool(prog, *c)
	if err != nil {
		return nil, err
	}

	return &script{
		scriptConfig: *c,
		sessionPool:  pool,
		sourceProg:   prog,
		sourceFile:   sourceFile,
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

// run runs the parse function. It receives a raw notification
// as a string and returns a list of S3 Events describing
// which files are going to be downloaded.
func (p *script) run(n string) ([]s3EventV2, error) {
	s := p.sessionPool.Get()
	defer p.sessionPool.Put(s)

	return s.runParseFunc(n)
}

func (p *script) String() string {
	return "script=[type=javascript, sources=" + p.sourceFile + "]"
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
