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

package multipass

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/elastic/beats/v7/dev-tools/mage/target/srvrlesstest/core/process"
	"github.com/elastic/beats/v7/dev-tools/mage/target/srvrlesstest/testing/common"
	"github.com/elastic/beats/v7/dev-tools/mage/target/srvrlesstest/testing/define"
	"github.com/elastic/beats/v7/dev-tools/mage/target/srvrlesstest/testing/runner"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"gopkg.in/yaml.v2"
)

const (
	Ubuntu = "ubuntu"
	Name   = "multipass"
)

type provisioner struct {
	logger common.Logger
}

// NewProvisioner creates the multipass provisioner
func NewProvisioner() common.InstanceProvisioner {
	return &provisioner{}
}

func (p *provisioner) Name() string {
	return Name
}

func (p *provisioner) SetLogger(l common.Logger) {
	p.logger = l
}

func (p *provisioner) Type() common.ProvisionerType {
	return common.ProvisionerTypeVM
}

// Supported returns true if multipass supports this OS.
//
// multipass only supports Ubuntu on the same architecture as the running host.
func (p *provisioner) Supported(os define.OS) bool {
	if os.Type != define.Linux {
		return false
	}
	if os.Distro != Ubuntu {
		return false
	}
	if os.Version != "20.04" && os.Version != "22.04" && os.Version != "24.04" {
		return false
	}
	// multipass only supports the same architecture of the host
	if os.Arch != runtime.GOARCH {
		return false
	}
	return true
}

func (p *provisioner) Provision(ctx context.Context, cfg common.Config, batches []common.OSBatch) ([]common.Instance, error) {
	// this doesn't provision the instances in parallel on purpose
	// multipass cannot handle it, it either results in instances sharing the same IP address
	// or some instances stuck in Starting state
	for _, batch := range batches {
		err := func(batch common.OSBatch) error {
			launchCtx, launchCancel := context.WithTimeout(ctx, 5*time.Minute)
			defer launchCancel()
			err := p.launch(launchCtx, cfg, batch)
			if err != nil {
				return fmt.Errorf("instance %s failed: %w", batch.ID, err)
			}
			return nil
		}(batch)
		if err != nil {
			return nil, err
		}
	}

	var results []common.Instance
	instances, err := p.list(ctx)
	if err != nil {
		return nil, err
	}
	for _, batch := range batches {
		mi, ok := instances[batch.ID]
		if !ok {
			return nil, fmt.Errorf("failed to find %s in multipass list output", batch.ID)
		}
		if mi.State != "Running" {
			return nil, fmt.Errorf("instance %s is not marked as running", batch.ID)
		}
		results = append(results, common.Instance{
			ID:          batch.ID,
			Provisioner: Name,
			Name:        batch.ID,
			IP:          mi.IPv4[0],
			Username:    "ubuntu",
			RemotePath:  "/home/ubuntu/agent",
			Internal:    nil,
		})
	}
	return results, nil
}

// Clean cleans up all provisioned resources.
func (p *provisioner) Clean(ctx context.Context, _ common.Config, instances []common.Instance) error {
	// doesn't execute in parallel for the same reasons in Provision
	// multipass just cannot handle it
	for _, instance := range instances {
		func(instance common.Instance) {
			deleteCtx, deleteCancel := context.WithTimeout(ctx, 5*time.Minute)
			defer deleteCancel()
			err := p.delete(deleteCtx, instance)
			if err != nil {
				// prevent a failure from stopping the other instances and clean
				p.logger.Logf("Delete instance %s failed: %s", instance.Name, err)
			}
		}(instance)
	}
	return nil
}

// launch creates an instance.
func (p *provisioner) launch(ctx context.Context, cfg common.Config, batch common.OSBatch) error {
	// check if instance already exists
	err := p.ensureInstanceNotExist(ctx, batch)
	if err != nil {
		p.logger.Logf(
			"could not check multipass instance %q does not exists, moving on anyway. Err: %v", err)
	}
	args := []string{
		"launch",
		"-c", "2",
		"-d", "50G", // need decent size for all the tests
		"-m", "4G",
		"-n", batch.ID,
		"--cloud-init", "-",
		batch.OS.Version,
	}

	publicKeyPath := filepath.Join(cfg.StateDir, "id_rsa.pub")
	publicKey, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read SSH key to send to multipass instance at %s: %w", publicKeyPath, err)
	}

	var cloudCfg cloudinitConfig
	cloudCfg.SSHAuthorizedKeys = []string{string(publicKey)}
	cloudCfgData, err := yaml.Marshal(&cloudCfg)
	if err != nil {
		return fmt.Errorf("failed to marshal cloud-init configuration: %w", err)
	}

	var output bytes.Buffer
	p.logger.Logf("Launching multipass image %s", batch.ID)
	proc, err := process.Start("multipass", process.WithContext(ctx), process.WithArgs(args), process.WithCmdOptions(runner.AttachOut(&output), runner.AttachErr(&output)))
	if err != nil {
		return fmt.Errorf("failed to run multipass launch: %w", err)
	}
	_, err = proc.Stdin.Write([]byte(fmt.Sprintf("#cloud-config\n%s", cloudCfgData)))
	if err != nil {
		_ = proc.Stdin.Close()
		_ = proc.Kill()
		<-proc.Wait()
		// print the output so its clear what went wrong
		fmt.Fprintf(os.Stdout, "%s\n", output.Bytes())
		return fmt.Errorf("failed to write cloudinit to stdin: %w", err)
	}
	_ = proc.Stdin.Close()
	ps := <-proc.Wait()
	if !ps.Success() {
		// print the output so its clear what went wrong
		fmt.Fprintf(os.Stdout, "%s\n", output.Bytes())
		return fmt.Errorf("failed to run multipass launch: exited with code: %d", ps.ExitCode())
	}
	return nil
}

func (p *provisioner) ensureInstanceNotExist(ctx context.Context, batch common.OSBatch) error {
	var output bytes.Buffer
	var stdErr bytes.Buffer
	proc, err := process.Start("multipass",
		process.WithContext(ctx),
		process.WithArgs([]string{"list", "--format", "json"}),
		process.WithCmdOptions(
			runner.AttachOut(&output),
			runner.AttachErr(&stdErr)))
	if err != nil {
		return fmt.Errorf("multipass list failed to run: %w", err)
	}

	state := <-proc.Wait()
	if !state.Success() {
		msg := fmt.Sprintf("multipass list exited with non-zero status: %s",
			state.String())
		p.logger.Logf(msg)
		p.logger.Logf("output: %s", output.String())
		p.logger.Logf("stderr: %s", stdErr.String())
		return errors.New(msg)
	}
	list := struct {
		List []struct {
			Ipv4    []string `json:"ipv4"`
			Name    string   `json:"name"`
			Release string   `json:"release"`
			State   string   `json:"state"`
		} `json:"list"`
	}{}
	err = json.NewDecoder(&output).Decode(&list)
	if err != nil {
		return fmt.Errorf("could not decode mutipass list output: %w", err)
	}

	for _, i := range list.List {
		if i.Name == batch.ID {
			p.logger.Logf("multipass trying to delete instance %s", batch.ID)

			output.Reset()
			stdErr.Reset()
			proc, err = process.Start("multipass",
				process.WithContext(ctx),
				process.WithArgs([]string{"delete", "--purge", batch.ID}),
				process.WithCmdOptions(
					runner.AttachOut(&output),
					runner.AttachErr(&stdErr)))
			if err != nil {
				return fmt.Errorf(
					"multipass instance %q already exist, state %q. Could not delete it: %w",
					batch.ID, i.State, err)
			}
			state = <-proc.Wait()
			if !state.Success() {
				msg := fmt.Sprintf("failed to delete and purge multipass instance %s: %s",
					batch.ID,
					state.String())
				p.logger.Logf(msg)
				p.logger.Logf("output: %s", output.String())
				p.logger.Logf("stderr: %s", stdErr.String())
				return errors.New(msg)
			}

			break
		}
	}

	return nil
}

// delete deletes an instance.
func (p *provisioner) delete(ctx context.Context, instance common.Instance) error {
	args := []string{
		"delete",
		"-p",
		instance.ID,
	}

	var output bytes.Buffer
	p.logger.Logf("Deleting instance %s", instance.Name)
	proc, err := process.Start("multipass", process.WithContext(ctx), process.WithArgs(args), process.WithCmdOptions(runner.AttachOut(&output), runner.AttachErr(&output)))
	if err != nil {
		// print the output so its clear what went wrong
		fmt.Fprintf(os.Stdout, "%s\n", output.Bytes())
		return fmt.Errorf("failed to run multipass delete: %w", err)
	}
	ps := <-proc.Wait()
	if ps.ExitCode() != 0 {
		// print the output so its clear what went wrong
		fmt.Fprintf(os.Stdout, "%s\n", output.Bytes())
		return fmt.Errorf("failed to run multipass delete: exited with code: %d", ps.ExitCode())
	}
	return nil
}

// list all the instances.
func (p *provisioner) list(ctx context.Context) (map[string]instance, error) {
	cmd := exec.CommandContext(ctx, "multipass", "list", "--format", "yaml")
	result, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run multipass list: %w", err)
	}

	// yaml output from multipass gives a list of instances for each instance name,
	// even though there is only ever 1 entry in the list
	var instancesMulti map[string][]instance
	err = yaml.Unmarshal(result, &instancesMulti)
	if err != nil {
		return nil, fmt.Errorf("failed to parse multipass list output: %w", err)
	}
	instances := map[string]instance{}
	for name, multi := range instancesMulti {
		instances[name] = multi[0]
	}

	return instances, nil
}

type instance struct {
	State   string   `yaml:"state"`
	IPv4    []string `yaml:"ipv4"`
	Release string   `yaml:"release"`
}

type cloudinitConfig struct {
	SSHAuthorizedKeys []string `yaml:"ssh_authorized_keys"`
}
