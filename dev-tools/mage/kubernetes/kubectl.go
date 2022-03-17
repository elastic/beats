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

package kubernetes

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

// KubectlApply applies the manifest file to the kubernetes cluster.
//
// KUBECONFIG must be in `env` to target a specific cluster.
func KubectlApply(env map[string]string, stdout, stderr io.Writer, filepath string) error {
	_, err := sh.Exec(
		env,
		stdout,
		stderr,
		"kubectl",
		"apply",
		"-f",
		filepath,
	)
	return err
}

// KubectlDelete deletes the resources from the manifest file from the kubernetes cluster.
//
// KUBECONFIG must be in `env` to target a specific cluster.
func KubectlDelete(env map[string]string, stdout, stderr io.Writer, filepath string) error {
	_, err := sh.Exec(
		env,
		stdout,
		stderr,
		"kubectl",
		"delete",
		"-f",
		filepath,
	)
	return err
}

// KubectlApplyInput applies the manifest string to the kubernetes cluster.
//
// KUBECONFIG must be in `env` to target a specific cluster.
func KubectlApplyInput(env map[string]string, stdout, stderr io.Writer, manifest string) error {
	return kubectlIn(env, stdout, stderr, manifest, "apply", "-f", "-")
}

// KubectlDeleteInput deletes the resources from the manifest string from the kubernetes cluster.
//
// KUBECONFIG must be in `env` to target a specific cluster.
func KubectlDeleteInput(env map[string]string, stdout, stderr io.Writer, manifest string) error {
	return kubectlIn(env, stdout, stderr, manifest, "delete", "-f", "-")
}

// KubectlWait waits for a condition to occur for a resource in the kubernetes cluster.
//
// KUBECONFIG must be in `env` to target a specific cluster.
func KubectlWait(env map[string]string, stdout, stderr io.Writer, waitFor, resource string, labels string) error {
	_, err := sh.Exec(
		env,
		stdout,
		stderr,
		"kubectl",
		"wait",
		"--timeout=300s",
		fmt.Sprintf("--for=%s", waitFor),
		resource,
		"-l",
		labels,
	)
	return err
}

func kubectlIn(env map[string]string, stdout, stderr io.Writer, input string, args ...string) error {
	c := exec.Command("kubectl", args...)
	c.Env = os.Environ()
	for k, v := range env {
		c.Env = append(c.Env, k+"="+v)
	}
	c.Stdout = stdout
	c.Stderr = stderr
	c.Stdin = strings.NewReader(input)

	if mg.Verbose() {
		fmt.Println("exec:", "kubectl", strings.Join(args, " "))
	}

	return c.Run()
}

func kubectlStart(env map[string]string, stdout, stderr io.Writer, args ...string) (*exec.Cmd, error) {
	c := exec.Command("kubectl", args...)
	c.Env = os.Environ()
	for k, v := range env {
		c.Env = append(c.Env, k+"="+v)
	}
	c.Stdout = stdout
	c.Stderr = stderr
	c.Stdin = nil

	if mg.Verbose() {
		fmt.Println("exec:", "kubectl", strings.Join(args, " "))
	}

	if err := c.Start(); err != nil {
		return nil, err
	}
	return c, nil
}
