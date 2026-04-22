// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build mage

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/magefile/mage/sh"

	devtools "github.com/elastic/beats/v7/dev-tools/mage"
	//mage:import
	_ "github.com/elastic/beats/v7/dev-tools/mage/target/common"
	//mage:import
	_ "github.com/elastic/beats/v7/dev-tools/mage/target/unittest"
)

const (
	distroDir    = "distributions/beats-otel-collector"
	manifestFile = "manifest.yaml"
	binaryName   = "beats-otel-collector"
)

func init() {

	devtools.BeatDescription = "OTel components used by the Elastic Agent"
	devtools.BeatLicense = "Elastic License"
}

// BuildOtelDistro builds the beats-otel-collector distribution using ocb.
func BuildOtelDistro() error {
	if _, err := exec.LookPath("ocb"); err != nil {
		return errors.New("ocb not found: please install ocb https://opentelemetry.io/docs/collector/extend/ocb")
	}

	fmt.Println(">> Building beats-otel-collector distribution")

	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	// ocb must run from the manifest directory so that output_path and replace
	// directives resolve relative to the manifest, not the mage invocation dir.
	dir := filepath.Join(wd, distroDir)
	if err := os.Chdir(dir); err != nil {
		return fmt.Errorf("changing to distribution directory: %w", err)
	}
	defer os.Chdir(wd) //nolint:errcheck

	return sh.RunV("ocb", "--config", manifestFile)
}

// RunOtelDistro runs the beats-otel-collector distribution.
// Set OTEL_ARGS to pass arguments to the binary (e.g. OTEL_ARGS="--config my.yaml").
// When OTEL_ARGS is not set, defaults to --config <distro>/config.yaml.
func RunOtelDistro() error {
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	binary := filepath.Join(wd, distroDir, "_build", binaryName)
	if _, err := os.Stat(binary); os.IsNotExist(err) {
		return fmt.Errorf("binary not found at %s: run 'mage buildOtelDistro' first", binary)
	}

	var args []string
	if raw := os.Getenv("OTEL_ARGS"); raw != "" {
		args = strings.Fields(raw)
	} else {
		args = []string{"--config", filepath.Join(wd, distroDir, "config.yaml")}
	}

	fmt.Printf(">> Running %s %s\n", binary, strings.Join(args, " "))
	return sh.RunV(binary, args...)
}
