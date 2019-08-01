// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package operation

import (
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/elastic/fleet/x-pack/pkg/artifact"
)

// Program defines a program which needs to be run.
// Is passed around operator operations.
type Program interface {
	BinaryName() string
	Version() string
	Config() map[string]interface{}
	Tags() map[Tag]string
	ID() string
	Spec(*artifact.Config) (ProcessSpec, error)
	ExecutionContext() ExecutionContext
	Directory(*artifact.Config) string
}

type program struct {
	executionCtx ExecutionContext
	config       map[string]interface{}
	spec         *ProcessSpec
	specLock     sync.Mutex
}

// NewProgram creates a program which satisfies Program interface and can be used with Operator.
func NewProgram(binaryName, version string, config map[string]interface{}, tags map[Tag]string) Program {
	if config == nil {
		config = make(map[string]interface{})
	}
	return &program{
		executionCtx: NewExecutionContext(binaryName, version, tags),
		config:       config,
	}
}

// NewProgramWithContext creates a program with pregenerated execution context.
func NewProgramWithContext(ctx ExecutionContext, config map[string]interface{}) Program {
	return &program{
		executionCtx: ctx,
		config:       config,
	}
}

func (p *program) BinaryName() string {
	return p.executionCtx.BinaryName
}
func (p *program) Version() string                    { return p.executionCtx.Version }
func (p *program) Tags() map[Tag]string               { return p.executionCtx.Tags }
func (p *program) ID() string                         { return p.executionCtx.ID }
func (p *program) ExecutionContext() ExecutionContext { return p.executionCtx }

func (p *program) Config() map[string]interface{} {
	// TODO: use default from spec and overwrite with what came as an input
	return p.config
}

func (p *program) Spec(config *artifact.Config) (ProcessSpec, error) {
	p.specLock.Lock()
	defer p.specLock.Unlock()

	if p.spec != nil {
		return *p.spec, nil
	}

	dir := p.Directory(config)
	specFile := filepath.Join(dir, p.BinaryName()+".spec")
	f, err := os.Open(specFile)
	if err != nil {

		return getDefaultSpec(dir, p.executionCtx), nil
	}

	decoder := json.NewDecoder(f)
	err = decoder.Decode(&p.spec)
	if err == nil {
		if !filepath.IsAbs(p.spec.BinaryPath) {
			p.spec.BinaryPath = filepath.Join(dir, p.spec.BinaryPath)
			if !filepath.IsAbs(p.spec.BinaryPath) {
				p.spec.BinaryPath, err = filepath.Abs(p.spec.BinaryPath)
				if err != nil {
					return *p.spec, err
				}
			}
		}
	}
	return *p.spec, err
}

func (p *program) Directory(config *artifact.Config) string {
	if p.Version() == "" {
		return filepath.Join(config.InstallPath, p.BinaryName())
	}

	path, err := artifact.GetArtifactPath(p.BinaryName(), p.Version(), config.OS(), config.Arch(), config.InstallPath)
	if err != nil {
		return ""
	}

	suffix := ".tar.gz"
	if config.OS() == "windows" {
		suffix = ".zip"
	}

	return strings.TrimSuffix(path, suffix)
}

func getDefaultSpec(dir string, ctx ExecutionContext) ProcessSpec {
	if !isKnownBeat(ctx.BinaryName) {
		return ProcessSpec{
			BinaryPath: path.Join(dir, ctx.BinaryName),
		}
	}

	return ProcessSpec{
		BinaryPath:   path.Join(dir, ctx.BinaryName),
		Args:         []string{"-e"},
		Configurable: ConfigurableFile, // known unrolled beat will be started with a generated configuration file
	}

}

func isKnownBeat(name string) bool {
	switch name {
	case "filebeat":
		fallthrough
	case "metricbeat":
		return true
	}

	return false
}
