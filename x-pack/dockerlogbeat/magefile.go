// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build mage

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"

	"github.com/elastic/beats/dev-tools/mage"

	devtools "github.com/elastic/beats/dev-tools/mage"
	"github.com/pkg/errors"

	// mage:import
	_ "github.com/elastic/beats/dev-tools/mage/target/common"
	// mage:import
	_ "github.com/elastic/beats/dev-tools/mage/target/unittest"
	// mage:import
	_ "github.com/elastic/beats/dev-tools/mage/target/test"
)

var hubID = "elastic"
var name = "elastic-logging-plugin"
var containerName = name + "_container"
var dockerPluginName = filepath.Join(hubID, name)
var packageStagingDir = "build/package/"
var packageEndDir = "build/distributions/"
var dockerExportPath = filepath.Join(packageStagingDir, "temproot.tar")

func init() {
	devtools.BeatLicense = "Elastic License"
	devtools.BeatDescription = "The Docker Logging Driver is a docker plugin for the Elastic Stack."
}

// getPluginName returns the fully qualified name:version string
func getPluginName() (string, error) {
	version, err := mage.BeatQualifiedVersion()
	if err != nil {
		return "", errors.Wrap(err, "error getting beats version")
	}

	return dockerPluginName + ":" + version, nil
}

// Build builds docker rootfs container root
func Build() error {
	mage.CreateDir(packageStagingDir)
	mage.CreateDir(packageEndDir)

	dockerLogBeatDir, err := os.Getwd()
	if err != nil {
		return errors.Wrap(err, "error getting work dir")
	}

	err = os.Chdir("../..")
	if err != nil {
		return errors.Wrap(err, "error changing directory")
	}

	gv, err := mage.GoVersion()
	if err != nil {
		return errors.Wrap(err, "error determining go version")
	}

	err = sh.RunV("docker", "build", "--build-arg", "versionString"+"="+gv, "--target", "final", "-t", "rootfsimage", "-f", "x-pack/dockerlogbeat/Dockerfile", ".")
	if err != nil {
		return errors.Wrap(err, "error building final container image")
	}

	err = os.Chdir(dockerLogBeatDir)
	if err != nil {
		return errors.Wrap(err, "error returning to dockerlogbeat dir")
	}

	os.Mkdir(filepath.Join(packageStagingDir, "rootfs"), 0755)

	err = sh.RunV("docker", "create", "--name", containerName, "rootfsimage", "true")
	if err != nil {
		return errors.Wrap(err, "error creating container")
	}

	err = sh.RunV("docker", "export", containerName, "-o", dockerExportPath)
	if err != nil {
		return errors.Wrap(err, "error exporting container")
	}

	err = mage.Copy("config.json", filepath.Join(packageStagingDir, "config.json"))
	if err != nil {
		return errors.Wrap(err, "error copying config.json")
	}

	return sh.RunV("tar", "-xf", dockerExportPath, "-C", filepath.Join(packageStagingDir, "rootfs"))
}

// CleanDocker removes working objects and containers
func CleanDocker() error {
	name, err := getPluginName()
	if err != nil {
		return err
	}

	sh.RunV("docker", "rm", "-vf", containerName)
	sh.RunV("docker", "rmi", "rootfsimage")
	sh.Rm(packageStagingDir)
	sh.RunV("docker", "plugin", "disable", "-f", name)
	sh.RunV("docker", "plugin", "rm", "-f", name)

	return nil
}

// Install installs the beat
func Install() error {
	name, err := getPluginName()
	if err != nil {
		return err
	}

	err = sh.RunV("docker", "plugin", "create", name, packageStagingDir)
	if err != nil {
		return errors.Wrap(err, "error creating plugin")
	}

	err = sh.RunV("docker", "plugin", "enable", name)
	if err != nil {
		return errors.Wrap(err, "error enabling plugin")
	}

	return nil
}

// Package builds and creates a docker plugin
func Package() {
	mg.SerialDeps(Build, Install)
}

// Release builds a "release" tarball that can be used later with `docker plugin create`
func Release() error {
	mg.Deps(Build)

	version, err := mage.BeatQualifiedVersion()
	if err != nil {
		return errors.Wrap(err, "error getting beats version")
	}

	tarballName := fmt.Sprintf("%s-%s-%s.tar.gz", name, version, "docker-plugin")

	bashScript := `#!/bin/bash
docker plugin create %s
docker plugin enable %s
	`
	formatted := []byte(fmt.Sprintf(bashScript, name, name))
	err = ioutil.WriteFile(filepath.Join(packageStagingDir, "install.sh"), formatted, 0774)
	if err != nil {
		return errors.Wrap(err, "error writing script")
	}

	outpath := filepath.Join(packageEndDir, tarballName)

	sh.RunV("tar", "zcf", outpath,
		filepath.Join(packageStagingDir, "rootfs"),
		filepath.Join(packageStagingDir, "config.json"),
		filepath.Join(packageStagingDir, "install.sh"))

	return nil
}

// IntegTest is currently a dummy test for the `testsuite` target
func IntegTest() {
	fmt.Printf("There are no Integration tests for The Elastic Log Plugin\n")
}

// Update is currently a dummy test for the `testsuite` target
func Update() {
	fmt.Printf("There is no Update for The Elastic Log Plugin\n")
}
