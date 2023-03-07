// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package install

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

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
	// Verify osqueryd or osqueryd.exe exists
	osqFile := osqd.QsquerydPathForPlatform(goos, dir)
	osqExists, err := fileExistsLogged(log, osqFile)
	if err != nil {
		return err
	}
	if !osqExists {
		return fmt.Errorf("%w: %v", os.ErrNotExist, osqFile)
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
