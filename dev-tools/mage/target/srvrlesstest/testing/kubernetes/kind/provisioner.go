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

package kind

import (
	"bytes"
	"context"
	"fmt"
	"github.com/elastic/beats/v7/dev-tools/mage/target/srvrlesstest/testing/common"
	"github.com/elastic/beats/v7/dev-tools/mage/target/srvrlesstest/testing/define"
	"github.com/elastic/beats/v7/dev-tools/mage/target/srvrlesstest/testing/kubernetes"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
)

const (
	Name = "kind"
)

const clusterCfg string = `
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: ClusterConfiguration
    scheduler:
      extraArgs:
        bind-address: "0.0.0.0"
        secure-port: "10259"
    controllerManager:
      extraArgs:
        bind-address: "0.0.0.0"
        secure-port: "10257"
`

func NewProvisioner() common.InstanceProvisioner {
	return &provisioner{}
}

type provisioner struct {
	logger common.Logger
}

func (p *provisioner) Name() string {
	return Name
}

func (p *provisioner) Type() common.ProvisionerType {
	return common.ProvisionerTypeK8SCluster
}

func (p *provisioner) SetLogger(l common.Logger) {
	p.logger = l
}

func (p *provisioner) Supported(batch define.OS) bool {
	if batch.Type != define.Kubernetes || batch.Arch != runtime.GOARCH {
		return false
	}
	if batch.Distro != "" && batch.Distro != Name {
		// not kind, don't run
		return false
	}
	return true
}

func (p *provisioner) Provision(ctx context.Context, cfg common.Config, batches []common.OSBatch) ([]common.Instance, error) {
	var instances []common.Instance
	for _, batch := range batches {
		k8sVersion := fmt.Sprintf("v%s", batch.OS.Version)
		instanceName := fmt.Sprintf("%s-%s", k8sVersion, batch.Batch.Group)

		agentImageName, err := kubernetes.VariantToImage(batch.OS.DockerVariant)
		if err != nil {
			return nil, err
		}
		agentImageName = fmt.Sprintf("%s:%s", agentImageName, cfg.AgentVersion)
		agentImage, err := kubernetes.AddK8STestsToImage(ctx, p.logger, agentImageName, runtime.GOARCH)
		if err != nil {
			return nil, fmt.Errorf("failed to add k8s tests to image %s: %w", agentImageName, err)
		}

		exists, err := p.clusterExists(instanceName)
		if err != nil {
			return nil, fmt.Errorf("failed to check if cluster exists: %w", err)
		}
		if !exists {
			p.logger.Logf("Provisioning kind cluster %s", instanceName)
			nodeImage := fmt.Sprintf("kindest/node:%s", k8sVersion)
			clusterConfig := strings.NewReader(clusterCfg)

			ret, err := p.kindCmd(clusterConfig, "create", "cluster", "--name", instanceName, "--image", nodeImage, "--config", "-")
			if err != nil {
				return nil, fmt.Errorf("kind: failed to create cluster %s: %s", instanceName, ret.stderr)
			}

			exists, err = p.clusterExists(instanceName)
			if err != nil {
				return nil, err
			}

			if !exists {
				return nil, fmt.Errorf("kind: failed to find cluster %s after successful creation", instanceName)
			}
		} else {
			p.logger.Logf("Kind cluster %s already exists", instanceName)
		}

		kConfigPath, err := p.writeKubeconfig(instanceName)
		if err != nil {
			return nil, err
		}

		c, err := klient.NewWithKubeConfigFile(kConfigPath)
		if err != nil {
			return nil, err
		}

		if err := p.WaitForControlPlane(c); err != nil {
			return nil, err
		}

		if err := p.LoadImage(ctx, instanceName, agentImage); err != nil {
			return nil, err
		}

		instances = append(instances, common.Instance{
			ID:          batch.ID,
			Name:        instanceName,
			Provisioner: Name,
			IP:          "",
			Username:    "",
			RemotePath:  "",
			Internal: map[string]interface{}{
				"config":      kConfigPath,
				"version":     k8sVersion,
				"agent_image": agentImage,
			},
		})
	}

	return instances, nil
}

func (p *provisioner) LoadImage(ctx context.Context, clusterName string, image string) error {
	ret, err := p.kindCmd(nil, "load", "docker-image", "--name", clusterName, image)
	if err != nil {
		return fmt.Errorf("kind: load docker-image %s failed: %w: %s", image, err, ret.stderr)
	}
	return nil
}

func (p *provisioner) WaitForControlPlane(client klient.Client) error {
	r, err := resources.New(client.RESTConfig())
	if err != nil {
		return err
	}
	for _, sl := range []metav1.LabelSelectorRequirement{
		{Key: "component", Operator: metav1.LabelSelectorOpIn, Values: []string{"etcd", "kube-apiserver", "kube-controller-manager", "kube-scheduler"}},
		{Key: "k8s-app", Operator: metav1.LabelSelectorOpIn, Values: []string{"kindnet", "kube-dns", "kube-proxy"}},
	} {
		selector, err := metav1.LabelSelectorAsSelector(
			&metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					sl,
				},
			},
		)
		if err != nil {
			return err
		}
		err = wait.For(conditions.New(r).ResourceListMatchN(&v1.PodList{}, len(sl.Values), func(object k8s.Object) bool {
			pod, ok := object.(*v1.Pod)
			if !ok {
				return false
			}

			for _, cond := range pod.Status.Conditions {
				if cond.Type != v1.PodReady {
					continue
				}

				return cond.Status == v1.ConditionTrue
			}

			return false
		}, resources.WithLabelSelector(selector.String())))
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *provisioner) Clean(ctx context.Context, cfg common.Config, instances []common.Instance) error {
	// doesn't execute in parallel for the same reasons in Provision
	// multipass just cannot handle it
	for _, instance := range instances {
		func(instance common.Instance) {
			err := p.deleteCluster(instance.ID)
			if err != nil {
				// prevent a failure from stopping the other instances and clean
				p.logger.Logf("Delete instance %s failed: %s", instance.Name, err)
			}
		}(instance)
	}

	return nil
}

func (p *provisioner) clusterExists(name string) (bool, error) {
	ret, err := p.kindCmd(nil, "get", "clusters")
	if err != nil {
		return false, err
	}

	for _, c := range strings.Split(ret.stdout, "\n") {
		if c == name {
			return true, nil
		}
	}
	return false, nil
}

func (p *provisioner) writeKubeconfig(name string) (string, error) {
	kubecfg := fmt.Sprintf("%s-kubecfg", name)

	ret, err := p.kindCmd(nil, "get", "kubeconfig", "--name", name)
	if err != nil {
		return "", fmt.Errorf("kind get kubeconfig: stderr: %s: %w", ret.stderr, err)
	}

	file, err := os.CreateTemp("", fmt.Sprintf("kind-cluster-%s", kubecfg))
	if err != nil {
		return "", fmt.Errorf("kind kubeconfig file: %w", err)
	}
	defer file.Close()

	if n, err := io.WriteString(file, ret.stdout); n == 0 || err != nil {
		return "", fmt.Errorf("kind kubecfg file: bytes copied: %d: %w]", n, err)
	}

	return file.Name(), nil
}

type cmdResult struct {
	stdout string
	stderr string
}

func (p *provisioner) kindCmd(stdIn io.Reader, args ...string) (cmdResult, error) {

	var stdout, stderr bytes.Buffer
	cmd := exec.Command("kind", args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if stdIn != nil {
		cmd.Stdin = stdIn
	}
	err := cmd.Run()
	return cmdResult{
		stdout: stdout.String(),
		stderr: stderr.String(),
	}, err
}

func (p *provisioner) deleteCluster(name string) error {
	_, err := p.kindCmd(nil, "delete", "cluster", "--name", name)
	return err
}
