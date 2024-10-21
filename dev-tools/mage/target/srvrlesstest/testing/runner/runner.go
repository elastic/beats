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

package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/elastic/beats/v7/dev-tools/mage/target/srvrlesstest/testing"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v2"

	"github.com/elastic/beats/v7/dev-tools/mage/target/srvrlesstest/testing/common"
	"github.com/elastic/beats/v7/dev-tools/mage/target/srvrlesstest/testing/define"
	tssh "github.com/elastic/beats/v7/dev-tools/mage/target/srvrlesstest/testing/ssh"
	"github.com/elastic/beats/v7/dev-tools/mage/target/srvrlesstest/testing/supported"
)

// Result is the complete result from the runner.
type Result struct {
	// Tests is the number of tests ran.
	Tests int
	// Failures is the number of tests that failed.
	Failures int
	// Output is the raw test output.
	Output []byte
	// XMLOutput is the XML Junit output.
	XMLOutput []byte
	// JSONOutput is the JSON output.
	JSONOutput []byte
}

// State represents the state storage of what has been provisioned.
type State struct {
	// Instances stores provisioned and prepared instances.
	Instances []StateInstance `yaml:"instances"`

	// Stacks store provisioned stacks.
	Stacks []common.Stack `yaml:"stacks"`
}

// StateInstance is an instance stored in the state.
type StateInstance struct {
	common.Instance

	// Prepared set to true when the instance is prepared.
	Prepared bool `yaml:"prepared"`
}

// Runner runs the tests on remote instances.
type Runner struct {
	cfg    common.Config
	logger common.Logger
	ip     common.InstanceProvisioner
	sp     common.StackProvisioner

	batches []common.OSBatch

	batchToStack   map[string]stackRes
	batchToStackCh map[string]chan stackRes
	batchToStackMx sync.Mutex

	stateMx sync.Mutex
	state   State
}

// NewRunner creates a new runner based on the provided batches.
func NewRunner(cfg common.Config, ip common.InstanceProvisioner, sp common.StackProvisioner, batches ...define.Batch) (*Runner, error) {
	err := cfg.Validate()
	if err != nil {
		return nil, err
	}
	platforms, err := cfg.GetPlatforms()
	if err != nil {
		return nil, err
	}

	osBatches, err := supported.CreateBatches(batches, platforms, cfg.Groups, cfg.Matrix, cfg.SingleTest)
	if err != nil {
		return nil, err
	}
	osBatches = filterSupportedOS(osBatches, ip)

	logger := &runnerLogger{
		writer:    os.Stdout,
		timestamp: cfg.Timestamp,
	}
	ip.SetLogger(logger)
	sp.SetLogger(logger)

	r := &Runner{
		cfg:            cfg,
		logger:         logger,
		ip:             ip,
		sp:             sp,
		batches:        osBatches,
		batchToStack:   make(map[string]stackRes),
		batchToStackCh: make(map[string]chan stackRes),
	}

	err = r.loadState()
	if err != nil {
		return nil, err
	}
	return r, nil
}

// Logger returns the logger used by the runner.
func (r *Runner) Logger() common.Logger {
	return r.logger
}

// Run runs all the tests.
func (r *Runner) Run(ctx context.Context) (Result, error) {
	// validate tests can even be performed
	err := r.validate()
	if err != nil {
		return Result{}, err
	}

	// prepare
	prepareCtx, prepareCancel := context.WithTimeout(ctx, 10*time.Minute)
	defer prepareCancel()
	sshAuth, repoArchive, err := r.prepare(prepareCtx)
	if err != nil {
		return Result{}, err
	}

	// start the needed stacks
	err = r.startStacks(ctx)
	if err != nil {
		return Result{}, err
	}

	// only send to the provisioner the batches that need to be created
	var instances []StateInstance
	var batches []common.OSBatch
	for _, b := range r.batches {
		if !b.Skip {
			i, ok := r.findInstance(b.ID)
			if ok {
				instances = append(instances, i)
			} else {
				batches = append(batches, b)
			}
		}
	}
	if len(batches) > 0 {
		provisionedInstances, err := r.ip.Provision(ctx, r.cfg, batches)
		if err != nil {
			return Result{}, err
		}
		for _, i := range provisionedInstances {
			instances = append(instances, StateInstance{
				Instance: i,
				Prepared: false,
			})
		}
	}

	var results map[string]common.OSRunnerResult
	switch r.ip.Type() {
	case common.ProvisionerTypeVM:
		// use SSH to perform all the required work on the instances
		results, err = r.runInstances(ctx, sshAuth, repoArchive, instances)
		if err != nil {
			return Result{}, err
		}
	case common.ProvisionerTypeK8SCluster:
		results, err = r.runK8sInstances(ctx, instances)
		if err != nil {
			return Result{}, err
		}

	default:
		return Result{}, fmt.Errorf("invalid provisioner type %d", r.ip.Type())
	}

	// merge the results
	return r.mergeResults(results)
}

// Clean performs a cleanup to ensure anything that could have been left running is removed.
func (r *Runner) Clean() error {
	r.stateMx.Lock()
	defer r.stateMx.Unlock()

	var instances []common.Instance
	for _, i := range r.state.Instances {
		instances = append(instances, i.Instance)
	}
	r.state.Instances = nil
	stacks := make([]common.Stack, len(r.state.Stacks))
	copy(stacks, r.state.Stacks)
	r.state.Stacks = nil
	err := r.writeState()
	if err != nil {
		return err
	}

	var g errgroup.Group
	g.Go(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		return r.ip.Clean(ctx, r.cfg, instances)
	})
	for _, stack := range stacks {
		g.Go(func(stack common.Stack) func() error {
			return func() error {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
				defer cancel()
				return r.sp.Delete(ctx, stack)
			}
		}(stack))
	}
	return g.Wait()
}

func (r *Runner) runK8sInstances(ctx context.Context, instances []StateInstance) (map[string]common.OSRunnerResult, error) {
	results := make(map[string]common.OSRunnerResult)
	var resultsMx sync.Mutex
	var err error
	for _, instance := range instances {
		batch, ok := findBatchByID(instance.ID, r.batches)
		if !ok {
			err = fmt.Errorf("unable to find batch with ID: %s", instance.ID)
			continue
		}

		logger := &batchLogger{wrapped: r.logger, prefix: instance.ID}
		// start with the ExtraEnv first preventing the other environment flags below
		// from being overwritten
		env := map[string]string{}
		for k, v := range r.cfg.ExtraEnv {
			env[k] = v
		}

		// ensure that we have all the requirements for the stack if required
		if batch.Batch.Stack != nil {
			// wait for the stack to be ready before continuing
			logger.Logf("Waiting for stack to be ready...")
			stack, stackErr := r.getStackForBatchID(batch.ID)
			if stackErr != nil {
				err = stackErr
				continue
			}
			env["ELASTICSEARCH_HOST"] = stack.Elasticsearch
			env["ELASTICSEARCH_USERNAME"] = stack.Username
			env["ELASTICSEARCH_PASSWORD"] = stack.Password
			env["KIBANA_HOST"] = stack.Kibana
			env["KIBANA_USERNAME"] = stack.Username
			env["KIBANA_PASSWORD"] = stack.Password
			logger.Logf("Using Stack with Kibana host %s, credentials available under .integration-cache", stack.Kibana)
		}

		// set the go test flags
		env["GOTEST_FLAGS"] = r.cfg.TestFlags
		env["KUBECONFIG"] = instance.Instance.Internal["config"].(string)
		env["TEST_BINARY_NAME"] = r.cfg.BinaryName
		env["K8S_VERSION"] = instance.Instance.Internal["version"].(string)
		env["AGENT_IMAGE"] = instance.Instance.Internal["agent_image"].(string)

		prefix := fmt.Sprintf("%s-%s", instance.Instance.Internal["version"].(string), batch.ID)

		// run the actual tests on the host
		result, runErr := batch.OS.Runner.Run(ctx, r.cfg.VerboseMode, nil, logger, r.cfg.AgentVersion, prefix, batch.Batch, env)
		if runErr != nil {
			logger.Logf("Failed to execute tests on instance: %s", err)
			err = fmt.Errorf("failed to execute tests on instance %s: %w", instance.Name, err)
		}
		resultsMx.Lock()
		results[batch.ID] = result
		resultsMx.Unlock()
	}
	if err != nil {
		return nil, err
	}
	return results, nil
}

// runInstances runs the batch on each instance in parallel.
func (r *Runner) runInstances(ctx context.Context, sshAuth ssh.AuthMethod, repoArchive string, instances []StateInstance) (map[string]common.OSRunnerResult, error) {
	g, ctx := errgroup.WithContext(ctx)
	results := make(map[string]common.OSRunnerResult)
	var resultsMx sync.Mutex
	for _, i := range instances {
		func(i StateInstance) {
			g.Go(func() error {
				batch, ok := findBatchByID(i.ID, r.batches)
				if !ok {
					return fmt.Errorf("unable to find batch with ID: %s", i.ID)
				}
				logger := &batchLogger{wrapped: r.logger, prefix: i.ID}
				result, err := r.runInstance(ctx, sshAuth, logger, repoArchive, batch, i)
				if err != nil {
					logger.Logf("Failed for instance %s (@ %s): %s\n", i.ID, i.IP, err)
					return err
				}
				resultsMx.Lock()
				results[batch.ID] = result
				resultsMx.Unlock()
				return nil
			})
		}(i)
	}
	err := g.Wait()
	if err != nil {
		return nil, err
	}
	return results, nil
}

// runInstance runs the batch on the machine.
func (r *Runner) runInstance(ctx context.Context, sshAuth ssh.AuthMethod, logger common.Logger, repoArchive string, batch common.OSBatch, instance StateInstance) (common.OSRunnerResult, error) {
	sshPrivateKeyPath, err := filepath.Abs(filepath.Join(r.cfg.StateDir, "id_rsa"))
	if err != nil {
		return common.OSRunnerResult{}, fmt.Errorf("failed to determine OGC SSH private key path: %w", err)
	}

	logger.Logf("Starting SSH; connect with `ssh -i %s %s@%s`", sshPrivateKeyPath, instance.Username, instance.IP)
	client := tssh.NewClient(instance.IP, instance.Username, sshAuth, logger)
	connectCtx, connectCancel := context.WithTimeout(ctx, 10*time.Minute)
	defer connectCancel()
	err = client.Connect(connectCtx)
	if err != nil {
		logger.Logf("Failed to connect to instance %s: %s", instance.IP, err)
		return common.OSRunnerResult{}, fmt.Errorf("failed to connect to instance %s: %w", instance.Name, err)
	}
	defer client.Close()
	logger.Logf("Connected over SSH")

	if !instance.Prepared {
		// prepare the host to run the tests
		logger.Logf("Preparing instance")
		err = batch.OS.Runner.Prepare(ctx, client, logger, batch.OS.Arch, r.cfg.GOVersion)
		if err != nil {
			logger.Logf("Failed to prepare instance: %s", err)
			return common.OSRunnerResult{}, fmt.Errorf("failed to prepare instance %s: %w", instance.Name, err)
		}

		// now its prepared, add to state
		instance.Prepared = true
		err = r.addOrUpdateInstance(instance)
		if err != nil {
			return common.OSRunnerResult{}, fmt.Errorf("failed to save instance state %s: %w", instance.Name, err)
		}
	}

	// copy the required files (done every run)
	err = batch.OS.Runner.Copy(ctx, client, logger, repoArchive, r.getBuilds(batch))
	if err != nil {
		logger.Logf("Failed to copy files instance: %s", err)
		return common.OSRunnerResult{}, fmt.Errorf("failed to copy files to instance %s: %w", instance.Name, err)
	}
	// start with the ExtraEnv first preventing the other environment flags below
	// from being overwritten
	env := map[string]string{}
	for k, v := range r.cfg.ExtraEnv {
		env[k] = v
	}

	// ensure that we have all the requirements for the stack if required
	if batch.Batch.Stack != nil {
		// wait for the stack to be ready before continuing
		logger.Logf("Waiting for stack to be ready...")
		stack, err := r.getStackForBatchID(batch.ID)
		if err != nil {
			return common.OSRunnerResult{}, err
		}
		env["ELASTICSEARCH_HOST"] = stack.Elasticsearch
		env["ELASTICSEARCH_USERNAME"] = stack.Username
		env["ELASTICSEARCH_PASSWORD"] = stack.Password
		env["KIBANA_HOST"] = stack.Kibana
		env["KIBANA_USERNAME"] = stack.Username
		env["KIBANA_PASSWORD"] = stack.Password
		logger.Logf("Using Stack with Kibana host %s, credentials available under .integration-cache", stack.Kibana)
	}

	// set the go test flags
	env["GOTEST_FLAGS"] = r.cfg.TestFlags
	env["TEST_BINARY_NAME"] = r.cfg.BinaryName

	// run the actual tests on the host
	result, err := batch.OS.Runner.Run(ctx, r.cfg.VerboseMode, client, logger, r.cfg.AgentVersion, batch.ID, batch.Batch, env)
	if err != nil {
		logger.Logf("Failed to execute tests on instance: %s", err)
		return common.OSRunnerResult{}, fmt.Errorf("failed to execute tests on instance %s: %w", instance.Name, err)
	}

	// fetch any diagnostics
	if r.cfg.DiagnosticsDir != "" {
		err = batch.OS.Runner.Diagnostics(ctx, client, logger, r.cfg.DiagnosticsDir)
		if err != nil {
			logger.Logf("Failed to fetch diagnostics: %s", err)
		}
	} else {
		logger.Logf("Skipping diagnostics fetch as DiagnosticsDir was not set")
	}

	return result, nil
}

// validate ensures that required builds of Elastic Agent exist
func (r *Runner) validate() error {
	var requiredFiles []string
	for _, b := range r.batches {
		if !b.Skip {
			for _, build := range r.getBuilds(b) {
				if !slices.Contains(requiredFiles, build.Path) {
					requiredFiles = append(requiredFiles, build.Path)
				}
				if !slices.Contains(requiredFiles, build.SHA512Path) {
					requiredFiles = append(requiredFiles, build.SHA512Path)
				}
			}
		}
	}
	var missingFiles []string
	for _, file := range requiredFiles {
		_, err := os.Stat(file)
		if os.IsNotExist(err) {
			missingFiles = append(missingFiles, file)
		} else if err != nil {
			return err
		}
	}
	if len(missingFiles) > 0 {
		return fmt.Errorf("missing required Elastic Agent package builds for integration runner to execute: %s", strings.Join(missingFiles, ", "))
	}
	return nil
}

// getBuilds returns the build for the batch.
func (r *Runner) getBuilds(b common.OSBatch) []common.Build {
	var builds []common.Build
	formats := []string{"targz", "zip", "rpm", "deb"}
	binaryName := "elastic-agent"

	var packages []string
	for _, p := range r.cfg.Packages {
		if slices.Contains(formats, p) {
			packages = append(packages, p)
		}
	}
	if len(packages) == 0 {
		packages = formats
	}

	// This is for testing beats in serverless environment
	if strings.HasSuffix(r.cfg.BinaryName, "beat") {
		var serverlessPackages []string
		for _, p := range packages {
			if slices.Contains([]string{"targz", "zip"}, p) {
				serverlessPackages = append(serverlessPackages, p)
			}
		}
		packages = serverlessPackages
	}

	if r.cfg.BinaryName != "" {
		binaryName = r.cfg.BinaryName
	}

	for _, f := range packages {
		arch := b.OS.Arch
		if arch == define.AMD64 {
			arch = "x86_64"
		}
		suffix, err := testing.GetPackageSuffix(b.OS.Type, b.OS.Arch, f)
		if err != nil {
			// Means that OS type & Arch doesn't support that package format
			continue
		}
		packageName := filepath.Join(r.cfg.BuildDir, fmt.Sprintf("%s-%s-%s", binaryName, r.cfg.AgentVersion, suffix))
		build := common.Build{
			Version:    r.cfg.ReleaseVersion,
			Type:       b.OS.Type,
			Arch:       arch,
			Path:       packageName,
			SHA512Path: packageName + ".sha512",
		}

		builds = append(builds, build)
	}
	return builds
}

// prepare prepares for the runner to run.
//
// Creates the SSH keys to use, creates the archive of the repo and pulls the latest container for OGC.
func (r *Runner) prepare(ctx context.Context) (ssh.AuthMethod, string, error) {
	wd, err := WorkDir()
	if err != nil {
		return nil, "", err
	}
	cacheDir := filepath.Join(wd, r.cfg.StateDir)
	_, err = os.Stat(cacheDir)
	if errors.Is(err, os.ErrNotExist) {
		err = os.Mkdir(cacheDir, 0755)
		if err != nil {
			return nil, "", fmt.Errorf("failed to create %q: %w", cacheDir, err)
		}
	} else if err != nil {
		// unknown error
		return nil, "", err
	}

	var auth ssh.AuthMethod
	var repoArchive string
	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		a, err := r.createSSHKey(cacheDir)
		if err != nil {
			return err
		}
		auth = a
		return nil
	})
	g.Go(func() error {
		repo, err := r.createRepoArchive(gCtx, r.cfg.RepoDir, cacheDir)
		if err != nil {
			return err
		}
		repoArchive = repo
		return nil
	})
	err = g.Wait()
	if err != nil {
		return nil, "", err
	}
	return auth, repoArchive, err
}

// createSSHKey creates the required SSH keys
func (r *Runner) createSSHKey(dir string) (ssh.AuthMethod, error) {
	privateKey := filepath.Join(dir, "id_rsa")
	_, priErr := os.Stat(privateKey)
	publicKey := filepath.Join(dir, "id_rsa.pub")
	_, pubErr := os.Stat(publicKey)
	var signer ssh.Signer
	if errors.Is(priErr, os.ErrNotExist) || errors.Is(pubErr, os.ErrNotExist) {
		// either is missing (re-create)
		r.logger.Logf("Create SSH keys to use for SSH")
		_ = os.Remove(privateKey)
		_ = os.Remove(publicKey)
		pri, err := tssh.NewPrivateKey()
		if err != nil {
			return nil, fmt.Errorf("failed to create ssh private key: %w", err)
		}
		pubBytes, err := tssh.NewPublicKey(&pri.PublicKey)
		if err != nil {
			return nil, fmt.Errorf("failed to create ssh public key: %w", err)
		}
		priBytes := tssh.EncodeToPEM(pri)
		err = os.WriteFile(privateKey, priBytes, 0600)
		if err != nil {
			return nil, fmt.Errorf("failed to write ssh private key: %w", err)
		}
		err = os.WriteFile(publicKey, pubBytes, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to write ssh public key: %w", err)
		}
		signer, err = ssh.ParsePrivateKey(priBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse ssh private key: %w", err)
		}
	} else if priErr != nil {
		// unknown error
		return nil, priErr
	} else if pubErr != nil {
		// unknown error
		return nil, pubErr
	} else {
		// read from existing private key
		priBytes, err := os.ReadFile(privateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to read ssh private key %s: %w", privateKey, err)
		}
		signer, err = ssh.ParsePrivateKey(priBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse ssh private key: %w", err)
		}
	}
	return ssh.PublicKeys(signer), nil
}

func (r *Runner) createRepoArchive(ctx context.Context, repoDir string, dir string) (string, error) {
	zipPath := filepath.Join(dir, "agent-repo.zip")
	_ = os.Remove(zipPath) // start fresh
	r.logger.Logf("Creating zip archive of repo to send to remote hosts")
	err := createRepoZipArchive(ctx, repoDir, zipPath)
	if err != nil {
		return "", fmt.Errorf("failed to create zip archive of repo: %w", err)
	}
	return zipPath, nil
}

// startStacks starts the stacks required for the tests to run
func (r *Runner) startStacks(ctx context.Context) error {
	var versions []string
	batchToVersion := make(map[string]string)
	for _, lb := range r.batches {
		if !lb.Skip && lb.Batch.Stack != nil {
			if lb.Batch.Stack.Version == "" {
				// no version defined on the stack; set it to the defined stack version
				lb.Batch.Stack.Version = r.cfg.StackVersion
			}
			if !slices.Contains(versions, lb.Batch.Stack.Version) {
				versions = append(versions, lb.Batch.Stack.Version)
			}
			batchToVersion[lb.ID] = lb.Batch.Stack.Version
		}
	}

	var requests []stackReq
	for _, version := range versions {
		id := strings.Replace(version, ".", "", -1)
		requests = append(requests, stackReq{
			request: common.StackRequest{ID: id, Version: version},
			stack:   r.findStack(id),
		})
	}

	reportResult := func(version string, stack common.Stack, err error) {
		r.batchToStackMx.Lock()
		defer r.batchToStackMx.Unlock()
		res := stackRes{
			stack: stack,
			err:   err,
		}
		for batchID, batchVersion := range batchToVersion {
			if batchVersion == version {
				r.batchToStack[batchID] = res
				ch, ok := r.batchToStackCh[batchID]
				if ok {
					ch <- res
				}
			}
		}
	}

	// start goroutines to provision the needed stacks
	for _, request := range requests {
		go func(ctx context.Context, req stackReq) {
			var err error
			var stack common.Stack
			if req.stack != nil {
				stack = *req.stack
			} else {
				stack, err = r.sp.Create(ctx, req.request)
				if err != nil {
					reportResult(req.request.Version, stack, err)
					return
				}
				err = r.addOrUpdateStack(stack)
				if err != nil {
					reportResult(stack.Version, stack, err)
					return
				}
			}

			if stack.Ready {
				reportResult(stack.Version, stack, nil)
				return
			}

			stack, err = r.sp.WaitForReady(ctx, stack)
			if err != nil {
				reportResult(stack.Version, stack, err)
				return
			}

			err = r.addOrUpdateStack(stack)
			if err != nil {
				reportResult(stack.Version, stack, err)
				return
			}

			reportResult(stack.Version, stack, nil)
		}(ctx, request)
	}

	return nil
}

func (r *Runner) getStackForBatchID(id string) (common.Stack, error) {
	r.batchToStackMx.Lock()
	res, ok := r.batchToStack[id]
	if ok {
		r.batchToStackMx.Unlock()
		return res.stack, res.err
	}
	_, ok = r.batchToStackCh[id]
	if ok {
		return common.Stack{}, fmt.Errorf("getStackForBatchID called twice; this is not allowed")
	}
	ch := make(chan stackRes, 1)
	r.batchToStackCh[id] = ch
	r.batchToStackMx.Unlock()

	// 12 minutes is because the stack should have been ready after 10 minutes or returned an error
	// this only exists to ensure that if that code is not blocking that this doesn't block forever
	t := time.NewTimer(12 * time.Minute)
	defer t.Stop()
	select {
	case <-t.C:
		return common.Stack{}, fmt.Errorf("failed waiting for a response after 12 minutes")
	case res = <-ch:
		return res.stack, res.err
	}
}

func (r *Runner) findInstance(id string) (StateInstance, bool) {
	r.stateMx.Lock()
	defer r.stateMx.Unlock()
	for _, existing := range r.state.Instances {
		if existing.Same(StateInstance{
			Instance: common.Instance{ID: id, Provisioner: r.ip.Name()}}) {
			return existing, true
		}
	}
	return StateInstance{}, false
}

func (r *Runner) addOrUpdateInstance(instance StateInstance) error {
	r.stateMx.Lock()
	defer r.stateMx.Unlock()

	state := r.state
	found := false
	for idx, existing := range state.Instances {
		if existing.Same(instance) {
			state.Instances[idx] = instance
			found = true
			break
		}
	}
	if !found {
		state.Instances = append(state.Instances, instance)
	}
	r.state = state
	return r.writeState()
}

func (r *Runner) findStack(id string) *common.Stack {
	r.stateMx.Lock()
	defer r.stateMx.Unlock()
	for _, existing := range r.state.Stacks {
		if existing.Same(common.Stack{ID: id, Provisioner: r.sp.Name()}) {
			return &existing
		}
	}
	return nil
}

func (r *Runner) addOrUpdateStack(stack common.Stack) error {
	r.stateMx.Lock()
	defer r.stateMx.Unlock()

	state := r.state
	found := false
	for idx, existing := range state.Stacks {
		if existing.Same(stack) {
			state.Stacks[idx] = stack
			found = true
			break
		}
	}
	if !found {
		state.Stacks = append(state.Stacks, stack)
	}
	r.state = state
	return r.writeState()
}

func (r *Runner) loadState() error {
	data, err := os.ReadFile(r.getStatePath())
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to read state file %s: %w", r.getStatePath(), err)
	}
	var state State
	err = yaml.Unmarshal(data, &state)
	if err != nil {
		return fmt.Errorf("failed unmarshal state file %s: %w", r.getStatePath(), err)
	}
	r.state = state
	return nil
}

func (r *Runner) writeState() error {
	data, err := yaml.Marshal(&r.state)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}
	err = os.WriteFile(r.getStatePath(), data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write state file %s: %w", r.getStatePath(), err)
	}
	return nil
}

func (r *Runner) getStatePath() string {
	return filepath.Join(r.cfg.StateDir, "state.yml")
}

func (r *Runner) mergeResults(results map[string]common.OSRunnerResult) (Result, error) {
	var rawOutput bytes.Buffer
	var jsonOutput bytes.Buffer
	var suites JUnitTestSuites
	for id, res := range results {
		for _, pkg := range res.Packages {
			err := mergePackageResult(pkg, id, false, &rawOutput, &jsonOutput, &suites)
			if err != nil {
				return Result{}, err
			}
		}
		for _, pkg := range res.SudoPackages {
			err := mergePackageResult(pkg, id, true, &rawOutput, &jsonOutput, &suites)
			if err != nil {
				return Result{}, err
			}
		}
	}
	var junitBytes bytes.Buffer
	err := writeJUnit(&junitBytes, suites)
	if err != nil {
		return Result{}, fmt.Errorf("failed to marshal junit: %w", err)
	}

	var complete Result
	for _, suite := range suites.Suites {
		complete.Tests += suite.Tests
		complete.Failures += suite.Failures
	}
	complete.Output = rawOutput.Bytes()
	complete.JSONOutput = jsonOutput.Bytes()
	complete.XMLOutput = junitBytes.Bytes()
	return complete, nil
}

// Same returns true if other is the same instance as this one.
// Two instances are considered the same if their provider and ID are the same.
func (s StateInstance) Same(other StateInstance) bool {
	return s.Provisioner == other.Provisioner &&
		s.ID == other.ID
}

func mergePackageResult(pkg common.OSRunnerPackageResult, batchName string, sudo bool, rawOutput io.Writer, jsonOutput io.Writer, suites *JUnitTestSuites) error {
	suffix := ""
	sudoStr := "false"
	if sudo {
		suffix = "(sudo)"
		sudoStr = "true"
	}
	if pkg.Output != nil {
		rawLogger := &runnerLogger{writer: rawOutput, timestamp: false}
		pkgWriter := common.NewPrefixOutput(rawLogger, fmt.Sprintf("%s(%s)%s: ", pkg.Name, batchName, suffix))
		_, err := pkgWriter.Write(pkg.Output)
		if err != nil {
			return fmt.Errorf("failed to write raw output from %s %s: %w", batchName, pkg.Name, err)
		}
	}
	if pkg.JSONOutput != nil {
		jsonSuffix, err := suffixJSONResults(pkg.JSONOutput, fmt.Sprintf("(%s)%s", batchName, suffix))
		if err != nil {
			return fmt.Errorf("failed to suffix json output from %s %s: %w", batchName, pkg.Name, err)
		}
		_, err = jsonOutput.Write(jsonSuffix)
		if err != nil {
			return fmt.Errorf("failed to write json output from %s %s: %w", batchName, pkg.Name, err)
		}
	}
	if pkg.XMLOutput != nil {
		pkgSuites, err := parseJUnit(pkg.XMLOutput)
		if err != nil {
			return fmt.Errorf("failed to parse junit from %s %s: %w", batchName, pkg.Name, err)
		}
		for _, pkgSuite := range pkgSuites.Suites {
			// append the batch information to the suite name
			pkgSuite.Name = fmt.Sprintf("%s(%s)%s", pkgSuite.Name, batchName, suffix)
			pkgSuite.Properties = append(pkgSuite.Properties, JUnitProperty{
				Name:  "batch",
				Value: batchName,
			}, JUnitProperty{
				Name:  "sudo",
				Value: sudoStr,
			})
			suites.Suites = append(suites.Suites, pkgSuite)
		}
	}
	return nil
}

func findBatchByID(id string, batches []common.OSBatch) (common.OSBatch, bool) {
	for _, batch := range batches {
		if batch.ID == id {
			return batch, true
		}
	}
	return common.OSBatch{}, false
}

type runnerLogger struct {
	writer    io.Writer
	timestamp bool
}

func (l *runnerLogger) Logf(format string, args ...any) {
	if l.timestamp {
		_, _ = fmt.Fprintf(l.writer, "[%s] >>> %s\n", time.Now().Format(time.StampMilli), fmt.Sprintf(format, args...))
	} else {
		_, _ = fmt.Fprintf(l.writer, ">>> %s\n", fmt.Sprintf(format, args...))
	}
}

type batchLogger struct {
	wrapped common.Logger
	prefix  string
}

func filterSupportedOS(batches []common.OSBatch, provisioner common.InstanceProvisioner) []common.OSBatch {
	var filtered []common.OSBatch
	for _, batch := range batches {
		if ok := provisioner.Supported(batch.OS.OS); ok {
			filtered = append(filtered, batch)
		}
	}
	return filtered
}

func (b *batchLogger) Logf(format string, args ...any) {
	b.wrapped.Logf("(%s) %s", b.prefix, fmt.Sprintf(format, args...))
}

type stackRes struct {
	stack common.Stack
	err   error
}

type stackReq struct {
	request common.StackRequest
	stack   *common.Stack
}
