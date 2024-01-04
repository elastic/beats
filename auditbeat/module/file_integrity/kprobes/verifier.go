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

	"github.com/elastic/beats/v7/auditbeat/module/file_integrity/kprobes/tracing"

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
	tfs, err := tracing.NewTraceFS()
	if err != nil {
		return err
	}

	basePath, err := os.MkdirTemp("", "verifier")
	if err != nil {
		return err
	}

	defer func() {
		_ = os.RemoveAll(basePath)
	}()

	p, err := newPathMonitor(ctx, exec, 0, true)
	if err != nil {
		return err
	}

	defer func() {
		_ = p.Close()
	}()

	// create a perf channel that monitors events only for this tid
	channel, err := tracing.NewPerfChannel(
		tracing.WithTimestamp(),
		tracing.WithRingSizeExponent(4),
		tracing.WithBufferSize(512),
		tracing.WithTID(exec.GetTID()),
		tracing.WithPollTimeout(100*time.Millisecond),
	)
	if err != nil {
		return err
	}

	defer func() {
		_ = channel.Close()
		for probe := range probes {
			_ = tfs.RemoveKProbe(probe)
		}
	}()

	// install probes through tracefs and add them to the perf channel
	for tracingProbe, allocFunc := range probes {
		if err := tfs.AddKProbe(tracingProbe); err != nil {
			return err
		}

		desc, err := tfs.LoadProbeFormat(tracingProbe)
		if err != nil {
			return err
		}

		decoder, err := tracing.NewStructDecoder(desc, allocFunc)
		if err != nil {
			return err
		}

		if err := channel.MonitorProbe(desc, decoder); err != nil {
			return err
		}
	}

	// start the perf channel
	if err := channel.Run(); err != nil {
		return err
	}

	verifier, err := newEventsVerifier(basePath)
	if err != nil {
		return err
	}

	eProc := newEventProcessor(p, verifier, true)

	retC := make(chan error)
	go func() {
		defer close(retC)
		for {
			select {
			case <-channel.LostC():
				retC <- errors.New("event loss in perf channel")
				return

			case err := <-channel.ErrC():
				retC <- err
				return

			case err := <-p.ErrC():
				retC <- err
				return

			case e, ok := <-channel.C():
				if !ok {
					err = errors.New("perf channel closed unexpectedly")
					return
				}

				switch eWithType := e.(type) {
				case *ProbeEvent:
					if err := eProc.process(ctx, eWithType); err != nil {
						retC <- err
						return
					}
					continue
				default:
					retC <- errors.New("unexpected event type")
					return
				}
			case <-time.After(timeout):
				return
			}
		}
	}()

	if err := p.AddPathToMonitor(ctx, basePath); err != nil {
		return err
	}

	if err := exec.Run(verifier.GenerateEvents); err != nil {
		return err
	}

	select {
	case err = <-retC:
		if err != nil {
			return err
		}
	}

	if err := verifier.Verified(); err != nil {
		return err
	}

	return nil
}
