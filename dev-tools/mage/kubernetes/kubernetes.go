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
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/magefile/mage/mg"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v8/dev-tools/mage"
)

func init() {
	mage.RegisterIntegrationTester(&KubernetesIntegrationTester{})
}

type KubernetesIntegrationTester struct {
}

// Name returns kubernetes name.
func (d *KubernetesIntegrationTester) Name() string {
	return "kubernetes"
}

// Use determines if this tester should be used.
func (d *KubernetesIntegrationTester) Use(dir string) (bool, error) {
	kubernetesFile := filepath.Join(dir, "kubernetes.yml")
	if _, err := os.Stat(kubernetesFile); !os.IsNotExist(err) {
		return true, nil
	}
	return false, nil
}

// HasRequirements ensures that the required kubectl are installed.
func (d *KubernetesIntegrationTester) HasRequirements() error {
	if err := mage.HaveKubectl(); err != nil {
		return err
	}
	return nil
}

// StepRequirements returns the steps required for this tester.
func (d *KubernetesIntegrationTester) StepRequirements() mage.IntegrationTestSteps {
	return mage.IntegrationTestSteps{&mage.MageIntegrationTestStep{}, &KindIntegrationTestStep{}}
}

// Test performs the tests with kubernetes.
func (d *KubernetesIntegrationTester) Test(dir string, mageTarget string, env map[string]string) error {
	stdOut := ioutil.Discard
	stdErr := ioutil.Discard
	if mg.Verbose() {
		stdOut = os.Stdout
		stdErr = os.Stderr
	}

	manifestPath := filepath.Join(dir, "kubernetes.yml")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		// defensive, as `Use` should cause this runner not to be used if no file.
		return fmt.Errorf("no kubernetes.yml")
	}

	kubeConfig := env["KUBECONFIG"]
	if kubeConfig == "" {
		kubeConfig = env["KUBE_CONFIG"]
	}
	if kubeConfig == "" {
		fmt.Println("Skip running tests inside of kubernetes no KUBECONFIG defined.")
		return nil
	}

	if mg.Verbose() {
		fmt.Println(">> Applying module manifest to cluster...")
	}

	// Determine the path to use inside the pod.
	repo, err := mage.GetProjectRepoInfo()
	if err != nil {
		return err
	}
	magePath := filepath.Join("/go/src", repo.CanonicalRootImportPath, repo.SubDir, "build/mage-linux-amd64")

	// Apply the manifest from the dir. This is the requirements for the tests that will
	// run inside the cluster.
	if err := KubectlApply(env, stdOut, stdErr, manifestPath); err != nil {
		return errors.Wrapf(err, "failed to apply manifest %s", manifestPath)
	}
	defer func() {
		if mg.Verbose() {
			fmt.Println(">> Deleting module manifest from cluster...")
		}
		if err := KubectlDelete(env, stdOut, stdErr, manifestPath); err != nil {
			log.Printf("%s", errors.Wrapf(err, "failed to apply manifest %s", manifestPath))
		}
	}()

	err = waitKubeStateMetricsReadiness(env, stdOut, stdErr)
	if err != nil {
		return err
	}

	// Pass all environment variables inside the pod, except for KUBECONFIG as the test
	// should use the environment set by kubernetes on the pod.
	insideEnv := map[string]string{}
	for envKey, envVal := range env {
		if envKey != "KUBECONFIG" && envKey != "KUBE_CONFIG" {
			insideEnv[envKey] = envVal
		}
	}

	destDir := filepath.Join("/go/src", repo.CanonicalRootImportPath)
	workDir := filepath.Join(destDir, repo.SubDir)
	remote, err := NewKubeRemote(kubeConfig, "default", kubernetesClusterName(), workDir, destDir, repo.RootDir)
	if err != nil {
		return err
	}
	// Uses `os.Stdout` directly as its output should always be shown.
	err = remote.Run(insideEnv, os.Stdout, stdErr, magePath, mageTarget)
	if err != nil {
		return err
	}
	return nil
}

// InsideTest performs the tests inside of environment.
func (d *KubernetesIntegrationTester) InsideTest(test func() error) error {
	return test()
}

// waitKubeStateMetricsReadiness waits until kube-state-metrics Pod is ready to receive requests
func waitKubeStateMetricsReadiness(env map[string]string, stdOut, stdErr io.Writer) error {
	checkKubeStateMetricsReadyAttempts := 10
	readyAttempts := 1
	for {
		err := KubectlWait(env, stdOut, stdErr, "condition=ready", "pod", "app=kube-state-metrics")
		if err != nil {
			if mg.Verbose() {
				fmt.Println("Kube-state-metrics is not ready yet...retrying")
			}
		} else {
			break
		}
		if readyAttempts > checkKubeStateMetricsReadyAttempts {
			return errors.Wrapf(err, "Timeout waiting for kube-state-metrics")
		}
		time.Sleep(6 * time.Second)
		readyAttempts += 1
	}
	// kube-state-metrics ready, return with no error
	return nil
}

// kubernetesClusterName generates a name for the Kubernetes cluster.
func kubernetesClusterName() string {
	commit, err := mage.CommitHash()
	if err != nil {
		panic(errors.Wrap(err, "failed to construct kind cluster name"))
	}

	version, err := mage.BeatQualifiedVersion()
	if err != nil {
		panic(errors.Wrap(err, "failed to construct kind cluster name"))
	}
	version = strings.NewReplacer(".", "-").Replace(version)

	clusterName := "{{.BeatName}}-{{.Version}}-{{.ShortCommit}}-{{.StackEnvironment}}"
	clusterName = mage.MustExpand(clusterName, map[string]interface{}{
		"StackEnvironment": mage.StackEnvironment,
		"ShortCommit":      commit[:10],
		"Version":          version,
	})

	// The cluster name may be used as a component of Kubernetes resource names.
	// kind does this, for example.
	//
	// Since Kubernetes resources are required to have names that are valid DNS
	// names, we should ensure that the cluster name also meets this criterion.
	subDomainPattern := `^[A-Za-z0-9](?:[A-Za-z0-9\-]{0,61}[A-Za-z0-9])?$`
	// Note that underscores, in particular, are not permitted.
	matched, err := regexp.MatchString(subDomainPattern, clusterName)
	if err != nil {
		panic(errors.Wrap(err, "error while validating kind cluster name"))
	}
	if !matched {
		panic("constructed invalid kind cluster name")
	}

	return clusterName
}
