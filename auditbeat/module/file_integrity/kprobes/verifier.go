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
	"io/fs"
	"os"
	"strings"
	"time"

	"github.com/elastic/beats/v7/auditbeat/tracing"

	tkbtf "github.com/elastic/tk-btf"
)

//go:embed embed
var embedBTFFolder embed.FS

func getVerifiedProbes(ctx context.Context, timeout time.Duration) (map[tracing.Probe]tracing.AllocateFn, executor, error) {
	fExec := newFixedThreadExecutor(ctx)

	probeMgr, err := newProbeManager(fExec)
	if err != nil {
		return nil, nil, err
	}

	specs, err := loadAllSpecs()
	if err != nil {
		return nil, nil, err
	}

	var allErr error
	for len(specs) > 0 {

		s := specs[0]
		if !probeMgr.shouldBuild(s) {
			specs = specs[1:]
			continue
		}

		probes, err := probeMgr.build(s)
		if err != nil {
			allErr = errors.Join(allErr, err)
			specs = specs[1:]
			continue
		}

		if err := verify(ctx, fExec, probes, timeout); err != nil {
			if probeMgr.onErr(err) {
				continue
			}
			allErr = errors.Join(allErr, err)
			specs = specs[1:]
			continue
		}

		return probes, fExec, nil
	}

	fExec.Close()
	return nil, nil, errors.Join(allErr, errors.New("could not validate probes"))
}

func loadAllSpecs() ([]*tkbtf.Spec, error) {
	var specs []*tkbtf.Spec

	spec, err := tkbtf.NewSpecFromKernel()
	if err != nil {
		if !errors.Is(err, tkbtf.ErrSpecKernelNotSupported) {
			return nil, err
		}
	} else {
		specs = append(specs, spec)
	}

	embeddedSpecs, err := loadEmbeddedSpecs()
	if err != nil {
		return nil, err
	}
	specs = append(specs, embeddedSpecs...)
	return specs, nil
}

func loadEmbeddedSpecs() ([]*tkbtf.Spec, error) {
	var specs []*tkbtf.Spec
	err := fs.WalkDir(embedBTFFolder, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".btf") {
			return nil
		}

		embedFileBytes, err := embedBTFFolder.ReadFile(path)
		if err != nil {
			return err
		}

		embedSpec, err := tkbtf.NewSpecFromReader(bytes.NewReader(embedFileBytes), nil)
		if err != nil {
			return err
		}

		specs = append(specs, embedSpec)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return specs, nil
}

func verify(ctx context.Context, exec executor, probes map[tracing.Probe]tracing.AllocateFn, timeout time.Duration) error {
	basePath, err := os.MkdirTemp("", "verifier")
	if err != nil {
		return err
	}

	defer os.RemoveAll(basePath)

	verifier, err := newEventsVerifier(basePath)
	if err != nil {
		return err
	}

	pChannel, err := newPerfChannel(probes, 4, 512, exec.GetTID())
	if err != nil {
		return err
	}

	m, err := newMonitor(ctx, true, pChannel, exec)
	if err != nil {
		return err
	}

	defer m.Close()

	// start the monitor
	if err := m.Start(); err != nil {
		return err
	}

	// spaw goroutine to send events to verifier to be verified
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
				return
			case <-cancel:
				return
			}
		}
	}()

	// add verify base path to monitor
	if err := m.Add(basePath); err != nil {
		return err
	}

	// invoke verifier event generation from our executor
	if err := exec.Run(verifier.GenerateEvents); err != nil {
		return err
	}

	// wait for either no new events arriving for timeout duration or
	// ctx to be cancelled
	select {
	case err = <-retC:
		if err != nil {
			return err
		}
	case <-ctx.Done():
		return ctx.Err()
	}

	// check that all events have been verified
	if err := verifier.Verified(); err != nil {
		return err
	}

	return nil
}
