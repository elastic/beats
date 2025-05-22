// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package application_pool

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"math/rand/v2"
)

func Run(commands string) (*string, *string, error) {
	rInt := rand.Int()

	tempDir := os.TempDir()
	baseFilename := fmt.Sprintf("command-%d.ps1", rInt)
	filename := filepath.Join(tempDir, baseFilename)

	defer os.Remove(filename)

	err := os.WriteFile(filename, []byte(commands), os.FileMode(0700))
	if err != nil {
		return nil, nil, fmt.Errorf("error writing command file %s: %w", filename, err)
	}

	var stderr bytes.Buffer
	var stdout bytes.Buffer

	cmd := exec.Command("powershell.exe", "-ExecutionPolicy", "Bypass", "-NoLogo", "-NonInteractive", "-NoProfile", "-File", filename)

	cmd.Stderr = &stderr
	cmd.Stdout = &stdout

	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("error starting command with script %s: %w", filename, err)
	}

	if err := cmd.Wait(); err != nil {
		return nil, nil, fmt.Errorf("error waiting for command with script %s: %w", filename, err)
	}

	stdOutStr := stdout.String()
	stdErrStr := stderr.String()

	return &stdOutStr, &stdErrStr, nil
}
