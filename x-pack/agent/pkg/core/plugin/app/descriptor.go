// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package app

import (
	"path"
	"path/filepath"
	"strings"

	"github.com/elastic/beats/v7/x-pack/agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/artifact"
)

// Descriptor defines a program which needs to be run.
// Is passed around operator operations.
type Descriptor struct {
	executionCtx ExecutionContext
	directory    string
	spec         ProcessSpec
}

// NewDescriptor creates a program which satisfies Program interface and can be used with Operator.
func NewDescriptor(binaryName, version string, config *artifact.Config, tags map[Tag]string) *Descriptor {
	binaryName = strings.ToLower(binaryName)
	dir := directory(binaryName, version, config)

	return &Descriptor{
		directory:    dir,
		executionCtx: NewExecutionContext(binaryName, version, tags),
		spec:         spec(dir, binaryName),
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
func (p *Descriptor) Spec() ProcessSpec {
	return p.spec
}

// Directory specifies the root directory of the application within an install path.
func (p *Descriptor) Directory() string {
	return p.directory
}

// IsGrpcConfigurable yields true in case application is grpc configurable.
func (p *Descriptor) IsGrpcConfigurable() bool {
	return p.spec.Configurable == ConfigurableGrpc
}

func defaultSpec(dir string, binaryName string) ProcessSpec {
	if !isKnownBeat(binaryName) {
		return ProcessSpec{
			BinaryPath: path.Join(dir, binaryName),
		}
	}

	return ProcessSpec{
		BinaryPath:   path.Join(dir, binaryName),
		Args:         []string{},
		Configurable: ConfigurableFile, // known unrolled beat will be started with a generated configuration file
	}

}

func spec(directory, binaryName string) ProcessSpec {
	defaultSpec := defaultSpec(directory, binaryName)
	return populateSpec(directory, binaryName, defaultSpec)
}

func directory(binaryName, version string, config *artifact.Config) string {
	if version == "" {
		return filepath.Join(config.InstallPath, binaryName)
	}

	path, err := artifact.GetArtifactPath(binaryName, version, config.OS(), config.Arch(), config.InstallPath)
	if err != nil {
		return ""
	}

	suffix := ".tar.gz"
	if config.OS() == "windows" {
		suffix = ".zip"
	}

	return strings.TrimSuffix(path, suffix)
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

func populateSpec(dir, binaryName string, spec ProcessSpec) ProcessSpec {
	var programSpec program.Spec
	var found bool
	binaryName = strings.ToLower(binaryName)
	for _, prog := range program.Supported {
		if binaryName != strings.ToLower(prog.Name) {
			continue
		}
		found = true
		programSpec = prog
		break
	}

	if !found {
		return spec
	}

	if programSpec.Cmd != "" {
		spec.BinaryPath = filepath.Join(dir, programSpec.Cmd)
	}

	if len(programSpec.Args) > 0 {
		spec.Args = programSpec.Args
	}

	if programSpec.Configurable != "" {
		spec.Configurable = programSpec.Configurable
	}

	return spec
}
