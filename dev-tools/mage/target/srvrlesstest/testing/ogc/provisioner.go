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

package ogc

import (
	"bytes"
	"context"
	"fmt"
	"github.com/elastic/beats/v7/dev-tools/mage/target/srvrlesstest/core/process"
	"github.com/elastic/beats/v7/dev-tools/mage/target/srvrlesstest/testing/common"
	"github.com/elastic/beats/v7/dev-tools/mage/target/srvrlesstest/testing/define"
	"github.com/elastic/beats/v7/dev-tools/mage/target/srvrlesstest/testing/runner"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

const (
	// LayoutIntegrationTag is the tag added to all layouts for the integration testing framework.
	LayoutIntegrationTag = "agent-integration"
	Name                 = "ogc"
)

type provisioner struct {
	logger common.Logger
	cfg    Config
}

// NewProvisioner creates the OGC provisioner
func NewProvisioner(cfg Config) (common.InstanceProvisioner, error) {
	err := cfg.Validate()
	if err != nil {
		return nil, err
	}
	return &provisioner{
		cfg: cfg,
	}, nil
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

// Supported returns true when we support this OS for OGC.
func (p *provisioner) Supported(os define.OS) bool {
	_, ok := findOSLayout(os)
	return ok
}

func (p *provisioner) Provision(ctx context.Context, cfg common.Config, batches []common.OSBatch) ([]common.Instance, error) {
	// ensure the latest version
	pullCtx, pullCancel := context.WithTimeout(ctx, 5*time.Minute)
	defer pullCancel()
	err := p.ogcPull(pullCtx)
	if err != nil {
		return nil, err
	}

	// import the calculated layouts
	importCtx, importCancel := context.WithTimeout(ctx, 30*time.Second)
	defer importCancel()
	err = p.ogcImport(importCtx, cfg, batches)
	if err != nil {
		return nil, err
	}

	// bring up all the instances
	upCtx, upCancel := context.WithTimeout(ctx, 30*time.Minute)
	defer upCancel()
	upOutput, err := p.ogcUp(upCtx)
	if err != nil {
		return nil, fmt.Errorf("ogc up failed: %w", err)
	}

	// fetch the machines and run the batches on the machine
	machines, err := p.ogcMachines(ctx)
	if err != nil {
		return nil, err
	}
	if len(machines) == 0 {
		// Print the output so its clear what went wrong.
		// Without this it's unclear where OGC went wrong, it
		// doesn't do a great job of reporting a clean error
		fmt.Fprintf(os.Stdout, "%s\n", upOutput)
		return nil, fmt.Errorf("ogc didn't create any machines")
	}

	// map the machines to instances
	var instances []common.Instance
	for _, b := range batches {
		machine, ok := findMachine(machines, b.ID)
		if !ok {
			// print the output so its clear what went wrong.
			// Without this it's unclear where OGC went wrong, it
			// doesn't do a great job of reporting a clean error
			fmt.Fprintf(os.Stdout, "%s\n", upOutput)
			return nil, fmt.Errorf("failed to find machine for batch ID: %s", b.ID)
		}
		instances = append(instances, common.Instance{
			ID:          b.ID,
			Provisioner: Name,
			Name:        machine.InstanceName,
			IP:          machine.PublicIP,
			Username:    machine.Layout.Username,
			RemotePath:  machine.Layout.RemotePath,
			Internal: map[string]interface{}{
				"instance_id": machine.InstanceID,
			},
		})
	}
	return instances, nil
}

// Clean cleans up all provisioned resources.
func (p *provisioner) Clean(ctx context.Context, cfg common.Config, _ []common.Instance) error {
	return p.ogcDown(ctx)
}

// ogcPull pulls the latest ogc version.
func (p *provisioner) ogcPull(ctx context.Context) error {
	args := []string{
		"pull",
		"docker.elastic.co/observability-ci/ogc:5.0.1",
	}
	var output bytes.Buffer
	p.logger.Logf("Pulling latest ogc image")
	proc, err := process.Start("docker", process.WithContext(ctx), process.WithArgs(args), process.WithCmdOptions(runner.AttachOut(&output), runner.AttachErr(&output)))
	if err != nil {
		return fmt.Errorf("failed to run docker ogcPull: %w", err)
	}
	ps := <-proc.Wait()
	if ps.ExitCode() != 0 {
		// print the output so its clear what went wrong
		fmt.Fprintf(os.Stdout, "%s\n", output.Bytes())
		return fmt.Errorf("failed to run ogc pull: docker run exited with code: %d", ps.ExitCode())
	}
	return nil
}

// ogcImport imports all the required batches into OGC.
func (p *provisioner) ogcImport(ctx context.Context, cfg common.Config, batches []common.OSBatch) error {
	var layouts []Layout
	for _, ob := range batches {
		layouts = append(layouts, osBatchToOGC(cfg.StateDir, ob))
	}
	layoutData, err := yaml.Marshal(struct {
		Layouts []Layout `yaml:"layouts"`
	}{
		Layouts: layouts,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal layouts YAML: %w", err)
	}

	var output bytes.Buffer
	p.logger.Logf("Import layouts into ogc")
	proc, err := p.ogcRun(ctx, []string{"layout", "import"}, true, process.WithCmdOptions(runner.AttachOut(&output), runner.AttachErr(&output)))
	if err != nil {
		return fmt.Errorf("failed to run ogc import: %w", err)
	}
	_, err = proc.Stdin.Write(layoutData)
	if err != nil {
		_ = proc.Stdin.Close()
		_ = proc.Kill()
		<-proc.Wait()
		// print the output so its clear what went wrong
		fmt.Fprintf(os.Stdout, "%s\n", output.Bytes())
		return fmt.Errorf("failed to write layouts to stdin: %w", err)
	}
	_ = proc.Stdin.Close()
	ps := <-proc.Wait()
	if ps.ExitCode() != 0 {
		// print the output so its clear what went wrong
		fmt.Fprintf(os.Stdout, "%s\n", output.Bytes())
		return fmt.Errorf("failed to run ogc import: docker run exited with code: %d", ps.ExitCode())
	}
	return nil
}

// ogcUp brings up all the instances.
func (p *provisioner) ogcUp(ctx context.Context) ([]byte, error) {
	p.logger.Logf("Bring up instances through ogc")
	var output bytes.Buffer
	proc, err := p.ogcRun(ctx, []string{"up", LayoutIntegrationTag}, false, process.WithCmdOptions(runner.AttachOut(&output), runner.AttachErr(&output)))
	if err != nil {
		return nil, fmt.Errorf("failed to run ogc up: %w", err)
	}
	ps := <-proc.Wait()
	if ps.ExitCode() != 0 {
		// print the output so its clear what went wrong
		fmt.Fprintf(os.Stdout, "%s\n", output.Bytes())
		return nil, fmt.Errorf("failed to run ogc up: docker run exited with code: %d", ps.ExitCode())
	}
	return output.Bytes(), nil
}

// ogcDown brings down all the instances.
func (p *provisioner) ogcDown(ctx context.Context) error {
	p.logger.Logf("Bring down instances through ogc")
	var output bytes.Buffer
	proc, err := p.ogcRun(ctx, []string{"down", LayoutIntegrationTag}, false, process.WithCmdOptions(runner.AttachOut(&output), runner.AttachErr(&output)))
	if err != nil {
		return fmt.Errorf("failed to run ogc down: %w", err)
	}
	ps := <-proc.Wait()
	if ps.ExitCode() != 0 {
		// print the output so its clear what went wrong
		fmt.Fprintf(os.Stdout, "%s\n", output.Bytes())
		return fmt.Errorf("failed to run ogc down: docker run exited with code: %d", ps.ExitCode())
	}
	return nil
}

// ogcMachines lists all the instances.
func (p *provisioner) ogcMachines(ctx context.Context) ([]Machine, error) {
	var out bytes.Buffer
	proc, err := p.ogcRun(ctx, []string{"ls", "--as-yaml"}, false, process.WithCmdOptions(runner.AttachOut(&out)))
	if err != nil {
		return nil, fmt.Errorf("failed to run ogc ls: %w", err)
	}
	ps := <-proc.Wait()
	if ps.ExitCode() != 0 {
		return nil, fmt.Errorf("failed to run ogc ls: docker run exited with code: %d", ps.ExitCode())
	}
	var machines []Machine
	err = yaml.Unmarshal(out.Bytes(), &machines)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ogc ls output: %w", err)
	}
	return machines, nil
}

func (p *provisioner) ogcRun(ctx context.Context, args []string, interactive bool, processOpts ...process.StartOption) (*process.Info, error) {
	wd, err := runner.WorkDir()
	if err != nil {
		return nil, err
	}
	tokenName := filepath.Base(p.cfg.ServiceTokenPath)
	clientEmail, err := p.cfg.ClientEmail()
	if err != nil {
		return nil, err
	}
	projectID, err := p.cfg.ProjectID()
	if err != nil {
		return nil, err
	}
	runArgs := []string{"run"}
	if interactive {
		runArgs = append(runArgs, "-i")
	}
	runArgs = append(runArgs,
		"--rm",
		"-e",
		fmt.Sprintf("GOOGLE_APPLICATION_SERVICE_ACCOUNT=%s", clientEmail),
		"-e",
		fmt.Sprintf("GOOGLE_APPLICATION_CREDENTIALS=/root/%s", tokenName),
		"-e",
		fmt.Sprintf("GOOGLE_PROJECT=%s", projectID),
		"-e",
		fmt.Sprintf("GOOGLE_DATACENTER=%s", p.cfg.Datacenter),
		"-v",
		fmt.Sprintf("%s:/root/%s", p.cfg.ServiceTokenPath, tokenName),
		"-v",
		fmt.Sprintf("%s:%s", wd, wd),
		"-w",
		wd,
		"docker.elastic.co/observability-ci/ogc:5.0.1",
		"--",
		"ogc",
		"-v",
	)
	runArgs = append(runArgs, args...)
	opts := []process.StartOption{process.WithContext(ctx), process.WithArgs(runArgs)}
	opts = append(opts, processOpts...)
	return process.Start("docker", opts...)
}

func osBatchToOGC(cacheDir string, batch common.OSBatch) Layout {
	tags := []string{
		LayoutIntegrationTag,
		batch.OS.Type,
		batch.OS.Arch,
	}
	if batch.OS.Type == define.Linux {
		tags = append(tags, strings.ToLower(fmt.Sprintf("%s-%s", batch.OS.Distro, strings.Replace(batch.OS.Version, ".", "-", -1))))
	} else {
		tags = append(tags, strings.ToLower(fmt.Sprintf("%s-%s", batch.OS.Type, strings.Replace(batch.OS.Version, ".", "-", -1))))
	}
	los, _ := findOSLayout(batch.OS.OS)
	return Layout{
		Name:          batch.ID,
		Provider:      los.Provider,
		InstanceSize:  los.InstanceSize,
		RunsOn:        los.RunsOn,
		RemotePath:    los.RemotePath,
		Scale:         1,
		Username:      los.Username,
		SSHPrivateKey: cacheDir + "/id_rsa",
		SSHPublicKey:  cacheDir + "/id_rsa.pub",
		Ports:         []string{"22:22"},
		Tags:          tags,
		Labels: map[string]string{
			"division": "engineering",
			"org":      "ingest",
			"team":     "elastic-agent-control-plane",
			"project":  "elastic-agent",
		},
		Scripts: "path", // not used; but required by OGC
	}
}

func findOSLayout(os define.OS) (LayoutOS, bool) {
	for _, s := range ogcSupported {
		if s.OS == os {
			return s, true
		}
	}
	return LayoutOS{}, false
}

func findMachine(machines []Machine, name string) (Machine, bool) {
	for _, m := range machines {
		if m.Layout.Name == name {
			return m, true
		}
	}
	return Machine{}, false
}
