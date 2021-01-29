// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package source

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/elastic/beats/v7/libbeat/logp"

	"github.com/otiai10/copy"
)

type LocalSource struct {
	OrigPath    string `config:"path"`
	workingPath string
	BaseSource
}

var ErrNoPath = fmt.Errorf("local source defined with no path specified")

func ErrInvalidPath(path string) error {
	return fmt.Errorf("local source has invalid path '%s'", path)
}

func (l *LocalSource) Validate() error {
	if l.OrigPath == "" {
		return ErrNoPath
	}

	s, err := os.Stat(l.OrigPath)
	base := ErrInvalidPath(l.OrigPath)
	if err != nil {
		return fmt.Errorf("%s: %w", base, err)
	}
	if !s.IsDir() {
		return fmt.Errorf("%s: path points to a non-directory", base)
	}

	return nil
}

func (l *LocalSource) Fetch() (err error) {
	if l.workingPath != "" {
		return nil
	}
	l.workingPath, err = ioutil.TempDir("/tmp", "elastic-synthetics-")
	if err != nil {
		return fmt.Errorf("could not create tmp dir: %w", err)
	}

	err = copy.Copy(l.OrigPath, l.workingPath)
	if err != nil {
		return fmt.Errorf("could not copy suite: %w", err)
	}

	dir, err := getSuiteDir(l.workingPath)
	if err != nil {
		return err
	}

	if os.Getenv("ELASTIC_SYNTHETICS_OFFLINE") != "true" {
		// Ensure all deps installed
		err = runSimpleCommand(exec.Command("npm", "install"), dir)
		if err != nil {
			return err
		}

		// Update playwright, needs to run separately to ensure post-install hook is run that downloads
		// chrome. See https://github.com/microsoft/playwright/issues/3712
		err = runSimpleCommand(exec.Command("npm", "install", "playwright-chromium"), dir)
		if err != nil {
			return err
		}
	}

	return nil
}

func (l *LocalSource) Workdir() string {
	return l.workingPath
}

func (l *LocalSource) Close() error {
	if l.workingPath != "" {
		return os.RemoveAll(l.workingPath)
	}

	return nil
}

func getSuiteDir(suiteFile string) (string, error) {
	path, err := filepath.Abs(suiteFile)
	if err != nil {
		return "", err
	}
	stat, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	if stat.IsDir() {
		return suiteFile, nil
	}

	return filepath.Dir(suiteFile), nil
}

func runSimpleCommand(cmd *exec.Cmd, dir string) error {
	cmd.Dir = dir
	logp.Info("Running %s in %s", cmd, dir)
	output, err := cmd.CombinedOutput()
	logp.Info("Ran %s got %s", cmd, string(output))
	return err
}
