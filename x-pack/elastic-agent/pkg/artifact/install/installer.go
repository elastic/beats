// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package install

import (
	"context"
	"errors"
	"runtime"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/install/atomic"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/install/awaitable"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/install/dir"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/install/hooks"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/install/tar"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/install/zip"
)

var (
	// ErrConfigNotProvided is returned when provided config is nil
	ErrConfigNotProvided = errors.New("config not provided")
)

// Installer is an interface allowing installation of an artifact
type Installer interface {
	// Install installs an artifact and returns
	// location of the installed program
	// error if something went wrong
	Install(ctx context.Context, spec program.Spec, version, installDir string) error
}

// InstallerChecker is an interface that installs but also checks for valid installation.
type InstallerChecker interface {
	Installer

	// Check checks if the installation is good.
	Check(ctx context.Context, spec program.Spec, version, installDir string) error
}

// AwaitableInstallerChecker is an interface that installs, checks but also is awaitable to check when actions are done.
type AwaitableInstallerChecker interface {
	InstallerChecker

	// Waits for its work to be done.
	Wait()
}

// NewInstaller returns a correct installer associated with a
// package type:
// - rpm -> rpm installer
// - deb -> deb installer
// - binary -> zip installer on windows, tar installer on linux and mac
func NewInstaller(config *artifact.Config) (AwaitableInstallerChecker, error) {
	if config == nil {
		return nil, ErrConfigNotProvided
	}

	var installer Installer
	var err error
	if runtime.GOOS == "windows" {
		installer, err = zip.NewInstaller(config)
	} else {
		installer, err = tar.NewInstaller(config)
	}

	if err != nil {
		return nil, err
	}

	atomicInstaller, err := atomic.NewInstaller(installer)
	if err != nil {
		return nil, err
	}

	hooksInstaller, err := hooks.NewInstallerChecker(atomicInstaller, dir.NewChecker())
	if err != nil {
		return nil, err
	}

	return awaitable.NewInstaller(hooksInstaller, hooksInstaller)
}
