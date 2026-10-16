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

//go:build linux

package kprobes

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"time"

	"github.com/elastic/beats/v7/auditbeat/tracing"
	"github.com/elastic/elastic-agent-libs/logp"

	tkbtf "github.com/elastic/tk-btf"
)

//go:embed embed
var embedBTFFolder embed.FS

func getVerifiedProbes(ctx context.Context, timeout time.Duration, logger *logp.Logger) (map[tracing.Probe]tracing.AllocateFn, executor, error) {
	fExec := newFixedThreadExecutor(ctx)

	probeMgr, err := newProbeManager(fExec)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating probe manager: %w", err)
	}

	specs, err := loadAllSpecs()
	if err != nil {
		return nil, nil, fmt.Errorf("error loading BTF specs: %w", err)
	}

	var allErr error
	attempt := 0
	total := len(specs)
	logger.Infof("DEBUG loaded %d BTF specs (kernel spec first if present; 6 are embedded)", total)
	for len(specs) > 0 {

		s := specs[0]
		idx := total - len(specs)
		if !probeMgr.shouldBuild(s) {
			logger.Infof("DEBUG spec %d/%d skipped by shouldBuild", idx, total)
			specs = specs[1:]
			continue
		}

		probes, err := probeMgr.build(s)
		if err != nil {
			logger.Infof("DEBUG spec %d/%d build failed: %v", idx, total, err)
			allErr = errors.Join(allErr, err)
			specs = specs[1:]
			continue
		}

		attempt++
		logger.Infof("DEBUG verify attempt %d: spec %d/%d, %d specs remaining", attempt, idx, total, len(specs))
		if err := verify(ctx, fExec, probes, timeout, logger); err != nil {
			retry := probeMgr.onErr(err)
			logger.Infof("DEBUG verify attempt %d failed: retry=%t err=%v", attempt, retry, err)
			if retry {
				continue
			}
			logger.Infof("DEBUG spec %d/%d abandoned after verify failure", idx, total)
			allErr = errors.Join(allErr, err)
			specs = specs[1:]
			continue
		}

		logger.Infof("DEBUG spec %d/%d verified", idx, total)
		return probes, fExec, nil
	}

	fExec.Close()
	logger.Infof("DEBUG no spec validated; accumulated errors: %v", allErr)
	return nil, nil, errors.Join(allErr, errors.New("could not validate probes"))
}

func loadAllSpecs() ([]*tkbtf.Spec, error) {
	var specs []*tkbtf.Spec

	spec, err := tkbtf.NewSpecFromKernel()
	if err != nil {
		if !errors.Is(err, tkbtf.ErrSpecKernelNotSupported) {
			return nil, fmt.Errorf("error reading BTF from kernel: %w", err)
		}
	} else {
		specs = append(specs, spec)
	}

	embeddedSpecs, err := loadEmbeddedSpecs()
	if err != nil {
		return nil, fmt.Errorf("error loading embedded BTF specs: %w", err)
	}
	specs = append(specs, embeddedSpecs...)
	return specs, nil
}

func loadEmbeddedSpecs() ([]*tkbtf.Spec, error) {
	var specs []*tkbtf.Spec
	err := fs.WalkDir(embedBTFFolder, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("error while walking embedded BTF folder: %w", err)
		}

		if !strings.HasSuffix(path, ".btf") {
			return nil
		}

		embedFileBytes, err := embedBTFFolder.ReadFile(path)
		if err != nil {
			return fmt.Errorf("error reading embedded BTF file %s: %w", path, err)
		}

		embedSpec, err := tkbtf.NewSpecFromReader(bytes.NewReader(embedFileBytes), nil)
		if err != nil {
			return fmt.Errorf("error creating spec from embedded file %s: %w", path, err)
		}

		specs = append(specs, embedSpec)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return specs, nil
}

func verify(ctx context.Context, exec executor, probes map[tracing.Probe]tracing.AllocateFn, timeout time.Duration, logger *logp.Logger) error {
	basePath, err := os.MkdirTemp("", "verifier")
	if err != nil {
		return fmt.Errorf("error creating temp directory: %w", err)
	}

	defer os.RemoveAll(basePath)

	verifier, err := newEventsVerifier(basePath)
	if err != nil {
		return fmt.Errorf("error creating verifier: %w", err)
	}

	pChannel, err := newPerfChannel(probes, 4, 512, exec.GetTID())
	if err != nil {
		return fmt.Errorf("error creating perf buffer channels: %w", err)
	}

	m, err := newMonitor(ctx, true, pChannel, exec, logger)
	if err != nil {
		if closeErr := pChannel.Close(); closeErr != nil {
			logger.Warnf("error closing perf channel after monitor creation failure: %v", closeErr)
		}
		return fmt.Errorf("error creating events monitor: %w", err)
	}

	defer func() {
		if closeErr := m.Close(); closeErr != nil {
			logger.Warnf("error closing verification monitor: %v", closeErr)
		}
	}()

	// start the monitor
	if err := m.Start(); err != nil {
		return fmt.Errorf("error starting events monitor: %w", err)
	}

	// spawn goroutine to send events to verifier to be verified
	cancel := make(chan struct{})
	defer close(cancel)

	retC := make(chan error)

	go func() {
		defer close(retC)
		for {
			select {
			case runErr := <-m.ErrorChannel():
				retC <- runErr
				return

			case ev, ok := <-m.EventChannel():
				if !ok {
					retC <- errors.New("monitor closed unexpectedly")
					return
				}

				if err := verifier.validateEvent(ev.Path, ev.PID, ev.Op); err != nil {
					retC <- err
					return
				}
				continue
			case <-time.After(timeout):
				logger.Error("verify function timed out. Kernel may be unsupported or kprobes may have a bug")
				return
			case <-cancel:
				logger.Error("verify function got context canceled")
				return
			}
		}
	}()

	// add verify base path to monitor
	if err := m.Add(basePath); err != nil {
		return fmt.Errorf("error adding path %s to monitor: %w", basePath, err)
	}

	// invoke verifier event generation from our executor
	if err := exec.Run(verifier.GenerateEvents); err != nil {
		return fmt.Errorf("error running event generator: %w", err)
	}

	// wait for either no new events arriving for timeout duration or
	// ctx to be cancelled
	select {
	case err = <-retC:
		if err != nil {
			return fmt.Errorf("error while verifying events: %w", err)
		}
	case <-ctx.Done():
		return ctx.Err()
	}

	// check that all events have been verified
	if err := verifier.Verified(); err != nil {
		return fmt.Errorf("events could not be verified: %w", err)
	}

	return nil
}
