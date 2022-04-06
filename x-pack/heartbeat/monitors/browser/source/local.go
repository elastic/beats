// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package source

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"syscall"

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
	// ensure the used synthetics version dep used in suite does not
	// exceed our supported range
	err = validatePackageJSON(path.Join(l.OrigPath, "package.json"))
	if err != nil {
		return err
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
	defer func() {
		if err != nil {
			err := l.Close() // cleanup the dir if this function returns an err
			if err != nil {
				logp.Warn("could not cleanup dir: %s", err)
			}
		}
	}()

	err = copy.Copy(l.OrigPath, l.workingPath)
	if err != nil {
		return fmt.Errorf("could not copy suite: %w", err)
	}

	dir, err := getAbsoluteSuiteDir(l.workingPath)
	if err != nil {
		return err
	}

	if !Offline() {
		err = setupOnlineDir(dir)
		return err
	}

	return nil
}

// setupOnlineDir is run in environments with internet access and attempts to make sure the node env
// is setup correctly.
func setupOnlineDir(dir string) (err error) {
	// If we're not offline remove the node_modules folder so we can do a fresh install, this minimizes
	// issues with dependencies being broken.
	modDir := path.Join(dir, "node_modules")
	_, statErr := os.Stat(modDir)
	if os.IsExist(statErr) {
		err := os.RemoveAll(modDir)
		if err != nil {
			return fmt.Errorf("could not remove node_modules from '%s': %w", dir, err)
		}
	}

	// Ensure all deps installed
	err = runSimpleCommand(exec.Command("npm", "install"), dir)
	if err != nil {
		return err
	}

	return err
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

func getAbsoluteSuiteDir(suiteFile string) (string, error) {
	absPath, err := filepath.Abs(suiteFile)
	if err != nil {
		return "", err
	}
	stat, err := os.Stat(absPath)
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
	logp.Info("Ran %s (%d) got '%s': (%s) as (%d/%d)", cmd, cmd.ProcessState.ExitCode(), string(output), err, syscall.Getuid(), syscall.Geteuid())
	return err
}
