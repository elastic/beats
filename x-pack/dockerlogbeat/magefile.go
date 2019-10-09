// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build mage

package main

import (
	"os"

	"github.com/elastic/beats/dev-tools/mage"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"

	"github.com/pkg/errors"
)

var hubID = "ossifrage"
var pluginVersion = "0.0.1"
var name = "dockerlogbeat"
var containerName = name + "_container"
var dockerPluginName = hubID + "/" + name
var dockerPlugin = dockerPluginName + ":" + pluginVersion

// Build builds docker rootfs container root
func Build() error {
	mg.Deps(Clean)

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

	os.Mkdir("rootfs", 0755)

	err = sh.RunV("docker", "create", "--name", containerName, "rootfsimage", "true")
	if err != nil {
		return errors.Wrap(err, "error creating container")
	}

	err = sh.RunV("docker", "export", containerName, "-o", "temproot.tar")
	if err != nil {
		return errors.Wrap(err, "error exporting container")
	}

	return sh.RunV("tar", "-xf", "temproot.tar", "-C", "rootfs")
}

// Clean removes working objects and containers
func Clean() error {

	sh.RunV("docker", "rm", "-vf", containerName)
	sh.RunV("docker", "rmi", "rootfsimage")
	sh.Rm("temproot.tar")
	sh.Rm("rootfs")
	sh.RunV("docker", "plugin", "disable", "-f", dockerPlugin)
	sh.RunV("docker", "plugin", "rm", "-f", dockerPlugin)

	return nil
}

// Install installs the beat
func Install() error {
	err := sh.RunV("docker", "plugin", "create", dockerPlugin, ".")
	if err != nil {
		return errors.Wrap(err, "error creating plugin")
	}

	err = sh.RunV("docker", "plugin", "enable", dockerPlugin)
	if err != nil {
		return errors.Wrap(err, "error enabling plugin")
	}

	return nil
}

// Create builds and creates a docker plugin
func Create() {
	mg.SerialDeps(Build, Install)
}
