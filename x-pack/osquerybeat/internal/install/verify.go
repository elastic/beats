// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package install

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/fileutil"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/osqd"
	"github.com/elastic/elastic-agent-libs/logp"
)

func execDir() (exedir string, err error) {
	exefp, err := os.Executable()
	if err != nil {
		return "", err
	}
	exedir = filepath.Dir(exefp)
	return exedir, nil
}

// VerifyWithExecutableDirectory verifies installation with the current executable directory
func VerifyWithExecutableDirectory(log *logp.Logger) error {
	exedir, err := execDir()
	if err != nil {
		return err
	}

	return Verify(runtime.GOOS, exedir, log)
}

// Verify verifies installation in the given executable directory
func Verify(goos, dir string, log *logp.Logger) error {
	log.Infof("Install verification for %s", dir)
	// Verify osqueryd or osqueryd.exe exists and is a valid osquery binary.
	_, err := VerifyOsqueryBinary(goos, dir, log)
	if err != nil {
		return err
	}

	// Verify extension file exists
	extFileName := "osquery-extension.ext"
	if goos == "windows" {
		extFileName = "osquery-extension.exe"
	}
	extFile := filepath.Join(dir, extFileName)
	osqExtExists, err := fileExistsLogged(log, extFile)
	if err != nil {
		return err
	}

	if !osqExtExists {
		return fmt.Errorf("%w: %v", os.ErrNotExist, extFileName)
	}
	return nil
}

var osqueryVersionPattern = regexp.MustCompile(`(?i)osqueryd version ([0-9A-Za-z.\-+_]+)`)

func VerifyOsqueryBinary(goos, dir string, log *logp.Logger) (string, error) {
	osqFile := osqd.OsquerydPathForPlatform(goos, dir)
	osqExists, err := fileExistsLogged(log, osqFile)
	if err != nil {
		return "", err
	}
	if !osqExists {
		return "", fmt.Errorf("%w: %v", os.ErrNotExist, osqFile)
	}

	if goos != "windows" {
		info, err := os.Stat(osqFile)
		if err != nil {
			return "", err
		}
		if info.Mode()&0111 == 0 {
			return "", fmt.Errorf("osquery binary is not executable: %s", osqFile)
		}
	}

	// Execute validation only for current runtime OS.
	if goos != runtime.GOOS {
		return "", nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	//nolint:gosec // expected local executable path
	cmd := exec.CommandContext(ctx, osqFile, "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to execute osquery binary %s --version: %w", osqFile, err)
	}
	s := strings.TrimSpace(string(out))
	if s == "" {
		return "", fmt.Errorf("empty output from osquery binary %s --version", osqFile)
	}
	matches := osqueryVersionPattern.FindStringSubmatch(s)
	if len(matches) != 2 {
		return "", fmt.Errorf("unexpected osquery version output from %s: %q", osqFile, s)
	}
	return matches[1], nil
}

func fileExistsLogged(log *logp.Logger, fp string) (bool, error) {
	log.Infof("Check if file exists %s:", fp)
	fpExists, err := fileutil.FileExists(fp)
	if err != nil {
		log.Infof("File exists check failed for %s", fp)
	}
	if !fpExists {
		log.Infof("File %s doesn't exists", fp)
	}
	return fpExists, err
}
