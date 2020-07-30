// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build windows

package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
)

func startSubprocess(flags *globalFlags, hasFileContent []byte, stopChan <-chan struct{}, wg *sync.WaitGroup) error {
	reexecPath := filepath.Join(paths.Data(), hashedDirName(hasFileContent), filepath.Base(os.Args[0]))
	argsOverrides := []string{
		"--path.data", paths.Data(),
		"--path.home", filepath.Dir(reexecPath),
		"--path.config", paths.Config(),
	}

	args := append([]string{reexecPath}, os.Args[1:]...)
	args = append(args, argsOverrides...)
	// no support for exec just spin a new child
	cmd := exec.Cmd{
		Path:   reexecPath,
		Args:   args,
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	wg.Add(1)
	go func() {
		<-stopChan
		cmd.Process.Kill()
		wg.Done()
	}()

	if err := cmd.Start(); err != nil {
		return err
	}

	// Wait so agent wont exit and service wont start another instance
	return cmd.Wait()
}
