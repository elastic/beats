// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build !windows

package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/reexec"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/configuration"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
)

func startSubprocess(flags *globalFlags, hasFileContent []byte, stopChan <-chan struct{}) error {
	reexecPath := filepath.Join(paths.Data(), hashedDirName(hasFileContent), filepath.Base(os.Args[0]))
	argsOverrides := []string{
		"--path.data", paths.Data(),
		"--path.home", filepath.Dir(reexecPath),
		"--path.config", paths.Config(),
	}

	pathConfigFile := flags.Config()
	rawConfig, err := config.LoadYAML(pathConfigFile)
	if err != nil {
		return errors.New(err,
			fmt.Sprintf("could not read configuration file %s", pathConfigFile),
			errors.TypeFilesystem,
			errors.M(errors.MetaKeyPath, pathConfigFile))
	}

	cfg, err := configuration.NewFromConfig(rawConfig)
	if err != nil {
		return errors.New(err,
			fmt.Sprintf("could not parse configuration file %s", pathConfigFile),
			errors.TypeFilesystem,
			errors.M(errors.MetaKeyPath, pathConfigFile))
	}

	logger, err := logger.NewFromConfig("", cfg.Settings.LoggingConfig)
	if err != nil {
		return err
	}

	rexLogger := logger.Named("reexec")
	rm := reexec.Manager(rexLogger, reexecPath)
	rm.ReExec(argsOverrides...)

	// trigger reexec
	rm.ShutdownComplete()

	return nil
}
