// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package app

import (
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/elastic/beats/x-pack/agent/pkg/artifact"
)

// Descriptor defines a program which needs to be run.
// Is passed around operator operations.
type Descriptor struct {
	executionCtx ExecutionContext
	spec         *ProcessSpec
	specLock     sync.Mutex
}

// NewDescriptor creates a program which satisfies Program interface and can be used with Operator.
func NewDescriptor(binaryName, version string, tags map[Tag]string) *Descriptor {
	return &Descriptor{
		executionCtx: NewExecutionContext(binaryName, version, tags),
	}
}

// NewDescriptorWithContext creates a program with pregenerated execution context.
func NewDescriptorWithContext(ctx ExecutionContext, config map[string]interface{}) *Descriptor {
	return &Descriptor{
		executionCtx: ctx,
	}
}

// BinaryName is the name of the binary. E.g filebeat.
func (p *Descriptor) BinaryName() string {
	return p.executionCtx.BinaryName
}

// Version specifies a version of the applications e.g '7.2.0'.
func (p *Descriptor) Version() string { return p.executionCtx.Version }

// Tags is a collection of tags used to specify application more precisely.
// Two descriptor with same binary name and version but with different tags will
// result in two different instances of the application.
func (p *Descriptor) Tags() map[Tag]string { return p.executionCtx.Tags }

// ID is a unique representation of the application.
func (p *Descriptor) ID() string { return p.executionCtx.ID }

// ExecutionContext returns execution context of the application.
func (p *Descriptor) ExecutionContext() ExecutionContext { return p.executionCtx }

// Spec returns a Process Specification with resolved binary path.
func (p *Descriptor) Spec(config *artifact.Config) (ProcessSpec, error) {
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

// Directory specifies the root direcvory of the application within an install path.
func (p *Descriptor) Directory(config *artifact.Config) string {
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

// IsGrpcConfigurable yields true in case application is grpc configurable.
func (p *Descriptor) IsGrpcConfigurable(config *artifact.Config) (bool, error) {
	spec, err := p.Spec(config)
	if err != nil {
		return false, err
	}

	return spec.Configurable == ConfigurableGrpc, nil
}

func getDefaultSpec(dir string, ctx ExecutionContext) ProcessSpec {
	if !isKnownBeat(ctx.BinaryName) {
		return ProcessSpec{
			BinaryPath: path.Join(dir, ctx.BinaryName),
		}
	}

	return ProcessSpec{
		BinaryPath:   path.Join(dir, ctx.BinaryName),
		Args:         []string{},
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
