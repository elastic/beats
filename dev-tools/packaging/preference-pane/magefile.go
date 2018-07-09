// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

// +build mage

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/pkg/errors"

	"github.com/elastic/beats/dev-tools/mage"
)

var builder = preferencePaneBuilder{
	Project:       "beats-preference-pane.xcodeproj",
	Configuration: mage.EnvOr("XCODE_CONFIGURATION", "Release"),
	PackageName:   "BeatsPrefPane.pkg",
	InstallDir:    "/Library/PreferencePanes",
	Identifier:    "co.elastic.beats.preference-pane",
	Version:       "1.0.0",
}

// Default specifies the default build target for mage.
var Default = All

// All build, sign, and package the Beats Preference Pane.
func All() { mg.SerialDeps(Build, Package) }

// Build builds the preference pane source using xcodebuild.
func Build() error { return builder.Build() }

// Package packages the pref pane into BeatsPrefPane.pkg.
func Package() error { return builder.Package() }

// Clean cleans the build artifacts.
func Clean() error { return sh.Rm("build") }

// --- preferencePaneBuilder

type preferencePaneBuilder struct {
	Project       string
	Configuration string
	PackageName   string
	InstallDir    string
	Identifier    string
	Version       string
}

func (b preferencePaneBuilder) SigningInfo() *mage.AppleSigningInfo {
	info, err := mage.GetAppleSigningInfo()
	if err != nil {
		panic(err)
	}

	return info
}

func (b preferencePaneBuilder) Build() error {
	if mage.IsUpToDate("build/Release/Beats.prefPane/Contents/MacOS/Beats",
		"helper", "beats-preference-pane", "beats-preference-pane.xcodeproj") {
		fmt.Println(">> Building MacOS Preference Pane (UP-TO-DATE)")
		return nil
	}

	fmt.Println(">> Building MacOS Preference Pane")
	err := sh.Run("xcodebuild", "build",
		"-project", b.Project,
		"-alltargets",
		"-configuration", b.Configuration,
		// This disables xcodebuild from attempting to codesign.
		// We do that in its own build step.
		"CODE_SIGN_IDENTITY=",
		"CODE_SIGNING_REQUIRED=NO")
	if err != nil {
		return err
	}

	return b.Sign()
}

func (b preferencePaneBuilder) Sign() error {
	if !b.SigningInfo().Sign {
		fmt.Println("Skipping signing of MacOS Preference Pane " +
			"(APPLE_SIGNING_ENABLED not set to true)")
		return nil
	}

	codesign := sh.RunCmd("codesign", "-s", b.SigningInfo().App.ID, "--timestamp")
	targets := []string{
		filepath.Join("build", b.Configuration, "Beats.prefPane/Contents/MacOS/helper"),
		filepath.Join("build", b.Configuration, "Beats.prefPane"),
	}

	fmt.Println(">> Signing MacOS Preference Pane")
	for _, target := range targets {
		if err := codesign(target); err != nil {
			return errors.Wrapf(err, "failed to codesign %v", target)
		}
	}
	return nil
}

func (b preferencePaneBuilder) Package() error {
	output := filepath.Join("build", b.PackageName)
	input := filepath.Join("build", b.Configuration, "Beats.prefPane")

	if mage.IsUpToDate(output, input) {
		fmt.Println(">> Packaging MacOS Preference Pane (UP-TO-DATE)")
		return nil
	}

	fmt.Println(">> Packaging MacOS Preference Pane")
	const pkgroot = "build/pkgroot"
	installDir := filepath.Join(pkgroot, b.InstallDir, filepath.Base(input))
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return err
	}

	if err := mage.Copy(input, installDir); err != nil {
		return err
	}

	pkgbuild := sh.RunCmd("pkgbuild")
	args := []string{
		"--root", pkgroot,
		"--identifier", b.Identifier,
		"--version", b.Version,
	}
	if b.SigningInfo().Sign {
		args = append(args, "--sign", b.SigningInfo().Installer.ID, "--timestamp")
	} else {
		fmt.Println("Skipping signing of MacOS " + b.PackageName +
			" (APPLE_SIGNING_ENABLED not set to true)")
	}
	args = append(args, output)

	return pkgbuild(args...)
}
