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
	"path/filepath"
	"regexp"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/pkg/errors"

	"github.com/elastic/beats/dev-tools/mage"
)

var (
	beats = []string{
		"auditbeat",
		"filebeat",
		"metricbeat",
	}

	kubectlCmd = sh.RunCmd("kubectl")
)

// Clean deletes the generated beat-kubernetes.yaml files.
func Clean() error {
	return mage.Clean([]string{
		"*beat-kubernetes.yaml",
	})
}

// IntegTest tests the kubernetes config by deploying it. kubectl is required.
func IntegTest() error {
	if err := haveKubernetes(); err != nil {
		fmt.Println(">> integTest: kubernetes testing (SKIPPED - kubernetes unavailable)")
		return nil
	}

	for _, beat := range beats {
		manifest := beat + "-kubernetes.yaml"
		if err := sh.RunV("kubectl", "create", "-f", manifest); err != nil {
			return errors.Wrapf(err, "failed deploying %v to kubernetes", manifest)
		}
	}
	return nil
}

// Update generates the kubernetes config files.
func Update() error {
	mg.Deps(Clean)

	version, err := mage.BeatQualifiedVersion()
	if err != nil {
		return err
	}

	for _, beat := range beats {
		in := filepath.Join(beat, beat+"-*.yaml")
		out := beat + "-kubernetes.yaml"

		inputs, err := mage.FindFiles(in)
		if err != nil {
			return err
		}

		if err = mage.FileConcat(out, 0644, inputs...); err != nil {
			return err
		}

		if err = mage.FindReplace(out, regexp.MustCompile(`%VERSION%`), version); err != nil {
			return err
		}
	}
	return nil
}

// haveKubernetes returns an error if the 'kubectl version' command returns a
// non-zero exit code.
func haveKubernetes() error {
	err := kubectlCmd("version")
	return errors.Wrap(err, "kubernetes is not available")
}
