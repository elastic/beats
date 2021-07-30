// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package app

import (
	"path/filepath"
	"strings"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact"
)

// Descriptor defines a program which needs to be run.
// Is passed around operator operations.
type Descriptor struct {
	spec         program.Spec
	executionCtx ExecutionContext
	directory    string
	process      ProcessSpec
}

// NewDescriptor creates a program which satisfies Program interface and can be used with Operator.
func NewDescriptor(spec program.Spec, version string, config *artifact.Config, tags map[Tag]string) *Descriptor {
	dir := directory(spec, version, config)
	return &Descriptor{
		spec:         spec,
		directory:    dir,
		executionCtx: NewExecutionContext(spec.ServicePort, spec.Cmd, version, tags),
		process:      specification(dir, spec),
	}
}

// ServicePort is the port the service will connect to gather GRPC information. When this is not
// 0 then the application is ran using the `service` application type, versus a `process` application.
func (p *Descriptor) ServicePort() int {
	return p.executionCtx.ServicePort
}

// ArtifactName is the name of the artifact to download from the artifact store. E.g beats/filebeat.
func (p *Descriptor) ArtifactName() string {
	return p.spec.Artifact
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

// Spec returns a program specification with resolved binary path.
func (p *Descriptor) Spec() program.Spec {
	return p.spec
}

// ProcessSpec returns a process specification with resolved binary path.
func (p *Descriptor) ProcessSpec() ProcessSpec {
	return p.process
}

// Directory specifies the root directory of the application within an install path.
func (p *Descriptor) Directory() string {
	return p.directory
}

func specification(dir string, spec program.Spec) ProcessSpec {
	return ProcessSpec{
		BinaryPath:    filepath.Join(dir, spec.Cmd),
		Args:          spec.Args,
		Configuration: nil,
	}
}

func directory(spec program.Spec, version string, config *artifact.Config) string {
	if version == "" {
		return filepath.Join(config.InstallPath, spec.Cmd)
	}

	path, err := artifact.GetArtifactPath(spec, version, config.OS(), config.Arch(), config.InstallPath)
	if err != nil {
		return ""
	}

	suffix := ".tar.gz"
	if config.OS() == "windows" {
		suffix = ".zip"
	}

	return strings.TrimSuffix(path, suffix)
}
