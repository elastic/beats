// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,386 linux,amd64

package tracing

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"
	"golang.org/x/sys/unix"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/go-sysinfo/providers/linux"
)

type Engine struct {
	log                       Logger
	traceFS                   *TraceFS
	traceFSpath               *string
	autoMount                 bool
	vars                      common.MapStr
	transforms                []ProbeTransform
	resolveSymbols            map[string][]string
	installed                 []Probe
	probes                    []ProbeDef
	guesses                   []Guesser
	perfChannelConf           []PerfChannelConf
	groupName, effectiveGroup string
	perfChannel               *PerfChannel
}

var kernelVersion string

func init() {
	var err error
	if kernelVersion, err = linux.KernelVersion(); err != nil {
		logp.Err("Failed fetching Linux kernel version: %v", err)
	}
}

// New creates a new engine.
// groupName must be a unique group name for kprobes.
// cfg is a list of configuration options to setup the kprobes tracing engine.
func New(groupName string, cfg ...ConfigFn) (*Engine, error) {
	eng := &Engine{
		log:            logp.L(), // TODO: Default logger?
		autoMount:      true,
		vars:           make(common.MapStr),
		groupName:      groupName,
		effectiveGroup: fmt.Sprintf("%s%d", groupName, os.Getpid()),
	}
	eng.perfChannelConf = append(eng.perfChannelConf, WithTimestamp())
	eng.transforms = append(eng.transforms,
		withGroup(eng.effectiveGroup),
		withTemplates(eng.vars))
	eng.vars.Update(archVariables)
	for _, act := range cfg {
		if err := act(eng); err != nil {
			return nil, err
		}
	}
	if len(eng.probes) == 0 {
		return nil, errors.New("no probes registered. Need at least one probe to monitor")
	}
	return eng, nil
}

// Vars gives access to the variables used as templates for probes.
func (e *Engine) Vars() common.MapStr {
	return e.vars
}

func (e *Engine) installProbe(pdef ProbeDef) (format ProbeFormat, decoder Decoder, err error) {
	for _, d := range e.transforms {
		pdef = d(pdef)
	}
	if pdef.Decoder == nil {
		return format, decoder, errors.New("nil decoder in probe definition")
	}
	if err = e.traceFS.AddKProbe(pdef.Probe); err != nil {
		return format, decoder, errors.Wrapf(err, "failed installing probe '%s'", pdef.Probe.String())
	}
	e.installed = append(e.installed, pdef.Probe)
	if format, err = e.traceFS.LoadProbeFormat(pdef.Probe); err != nil {
		return format, decoder, errors.Wrap(err, "failed to load probe format")
	}
	if decoder, err = pdef.Decoder(format); err != nil {
		return format, decoder, errors.Wrap(err, "failed to create decoder")
	}
	return
}

func (e *Engine) uninstallProbes() error {
	var errs multierror.Errors
	for _, probe := range e.installed {
		if err := e.traceFS.RemoveKProbe(probe); err != nil {
			errs = append(errs, err)
		}
	}
	e.installed = nil
	return errs.Err()
}

func (e *Engine) uninstallIf(condition ProbeCondition) error {
	kprobes, err := e.traceFS.ListKProbes()
	if err != nil {
		return errors.Wrap(err, "failed to list installed kprobes")
	}
	var errs multierror.Errors
	for _, probe := range kprobes {
		if condition(probe) {
			if err := e.traceFS.RemoveKProbe(probe); err != nil {
				errs = append(errs, errors.Wrapf(err, "unable to remove kprobe '%s'", probe.String()))
			}
		}
	}
	return errs.Err()
}

// Setup configures the engine. Must be called before any other method in Engine.
func (e *Engine) Setup() (err error) {
	e.log.Infof("Setting up kprobes for '%s' for kernel %s", e.effectiveGroup, kernelVersion)

	if err = e.openTraceFS(); err != nil {
		return errors.Wrap(err, "unable to access tracefs")
	}
	if err = e.uninstallIf(probeFromTerminatedProcess(e.groupName)); err != nil {
		e.log.Debugf("Errors removing probes from terminated processes: %v", err)
	}
	if err = e.uninstallIf(conflictingProbe(e.effectiveGroup)); err != nil {
		return errors.Wrapf(err, "unable to delete existing KProbes for group %s", e.effectiveGroup)
	}

	functions, err := loadTracingFunctions(e.traceFS)
	if err != nil {
		e.log.Debugf("Can't load available_tracing_functions, using alternative. err=%v", err)
	}

	for varName, alternatives := range e.resolveSymbols {
		if exists, _ := e.vars.HasKey(varName); exists {
			return fmt.Errorf("variable %s overwrites existing key", varName)
		}
		found := false
		var selected string
		for _, selected = range alternatives {
			if found = e.isKernelFunctionAvailable(selected, functions); found {
				break
			}
		}
		if !found {
			return fmt.Errorf("none of the required functions for %s is found. One of %v is required", varName, alternatives)
		}
		e.log.Debugf("Selected kernel function %s for %s", selected, varName)
		e.vars[varName] = selected
	}

	//
	// Make sure all the required kernel functions are available
	//
	for _, probeDef := range e.probes {
		probeDef = probeDef.ApplyTemplate(e.vars)
		name := probeDef.Probe.Address
		if !e.isKernelFunctionAvailable(name, functions) {
			return fmt.Errorf("required function '%s' is not available for tracing in the current kernel (%s)", name, kernelVersion)
		}
	}

	if len(e.guesses) > 0 {
		if err = e.guessAll(e.guesses,
			GuessContext{
				Log:     e.log,
				Vars:    e.vars,
				Timeout: time.Second * 30, // TODO: config
			}); err != nil {
			return errors.Wrap(err, "unable to guess one or more required parameters")
		}
	}
	names := make([]string, 0, len(e.vars))
	for name := range e.vars {
		names = append(names, name)
	}
	sort.Strings(names)
	e.log.Debugf("%d template variables in use:", len(e.vars))
	for _, key := range names {
		e.log.Debugf("  %s = %v", key, e.vars[key])
	}

	//
	// Create perf channel
	//
	e.perfChannel, err = NewPerfChannel(e.perfChannelConf...)
	if err != nil {
		return errors.Wrapf(err, "unable to create perf channel")
	}

	//
	// Register Kprobes
	//
	for _, probeDef := range e.probes {
		format, decoder, err := e.installProbe(probeDef)
		if err != nil {
			return errors.Wrapf(err, "unable to register probe %s", probeDef.Probe.String())
		}
		if err = e.perfChannel.MonitorProbe(format, decoder); err != nil {
			return errors.Wrapf(err, "unable to monitor probe %s", probeDef.Probe.String())
		}
	}
	return nil
}

// C returns a read-only channel for the tracing events. The types returned depend on the decoders
// associated with each probe installed.
func (e *Engine) C() <-chan interface{} {
	return e.perfChannel.C()
}

// ErrC is a read-only channel that notifies of any asynchronous error during the operation of the tracing channel.
func (e *Engine) ErrC() <-chan error {
	return e.perfChannel.ErrC()
}

// LostC is a read-only channel that notifies of any batch of lost tracing events between the kernel and userspace.
func (e *Engine) LostC() <-chan uint64 {
	return e.perfChannel.LostC()
}

// Start activates all kprobes and enables receiving tracing events.
func (e *Engine) Start() error {
	return e.perfChannel.Run()
}

// Stop terminates the collection of tracing event. No other methods on Engine can be called after Stop.
func (e *Engine) Stop() error {
	if e.perfChannel != nil {
		if err := e.perfChannel.Close(); err != nil {
			e.log.Warnf("Failed to close perf channel on exit: %v", err)
		}
	}
	if err := e.uninstallProbes(); err != nil {
		e.log.Warnf("Failed to remove KProbes on exit: %v", err)
	}
	// TODO: UMount
	return nil
}

func isRunningProcess(pid int) bool {
	return unix.Kill(pid, 0) == nil
}

func probeFromTerminatedProcess(groupName string) ProbeCondition {
	return func(probe Probe) bool {
		if strings.HasPrefix(probe.Group, groupName) {
			if pid, err := strconv.Atoi(probe.Group[len(groupName):]); err == nil && !isRunningProcess(pid) {
				return true
			}
		}
		return false
	}
}

func conflictingProbe(group string) ProbeCondition {
	return func(probe Probe) bool {
		return probe.Group == group
	}
}

// withGroup sets a custom group to probes before they are installed.
func withGroup(name string) ProbeTransform {
	return func(probe ProbeDef) ProbeDef {
		probe.Probe.Group = name
		return probe
	}
}

// withTemplates expands templates in probes before they are installed.
func withTemplates(vars common.MapStr) ProbeTransform {
	return func(probe ProbeDef) ProbeDef {
		return probe.ApplyTemplate(vars)
	}
}
