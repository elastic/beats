// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package process

import (
	"context"
	"io"
	"path/filepath"

	"gopkg.in/yaml.v2"

	"github.com/elastic/elastic-agent-client/v7/pkg/proto"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/app"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/process"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/server"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/state"
)

// Start starts the application with a specified config.
func (a *Application) Start(ctx context.Context, t app.Taggable, cfg map[string]interface{}) error {
	a.appLock.Lock()
	defer a.appLock.Unlock()

	return a.start(ctx, t, cfg, false)
}

// Start starts the application without grabbing the lock.
func (a *Application) start(ctx context.Context, t app.Taggable, cfg map[string]interface{}, isRestart bool) (err error) {
	defer func() {
		if err != nil {
			// inject App metadata
			err = errors.New(err, errors.M(errors.MetaKeyAppName, a.name), errors.M(errors.MetaKeyAppName, a.id))
		}
	}()

	// starting only if it's not running
	// or if it is, then only in case it's a restart and this call initiates from restart call
	if a.Started() && a.state.Status != state.Restarting {
		if a.state.ProcessInfo == nil {
			// already started if not stopped or crashed
			return nil
		}

		// in case app reported status it might still be running and failure timer
		// in progress. Stop timer and stop failing process
		a.stopFailedTimer()
		a.stopWatcher(a.state.ProcessInfo)

		// kill the process
		_ = a.state.ProcessInfo.Process.Kill()
		a.state.ProcessInfo = nil
	}

	if a.state.Status == state.Restarting && !isRestart {
		return nil
	}

	cfgStr, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	a.startContext = ctx
	a.tag = t
	srvState := a.srvState

	// Failed applications can be started again.
	if srvState != nil {
		a.setState(state.Starting, "Starting", nil)
		srvState.SetStatus(proto.StateObserved_STARTING, a.state.Message, a.state.Payload)
		srvState.UpdateConfig(srvState.Config())
	} else {
		a.srvState, err = a.srv.Register(a, string(cfgStr))
		if err != nil {
			return err
		}
		// Set input types from the spec
		a.srvState.SetInputTypes(a.desc.Spec().ActionInputTypes)
	}

	if a.state.Status != state.Stopped {
		// restarting as it was previously in a different state
		a.setState(state.Restarting, "Restarting", nil)
	} else if a.state.Status != state.Restarting {
		// keep restarting state otherwise it's starting
		a.setState(state.Starting, "Starting", nil)
	}

	defer func() {
		if err != nil {
			if a.srvState != nil {
				a.srvState.Destroy()
				a.srvState = nil
			}
			if a.state.ProcessInfo != nil {
				_ = a.state.ProcessInfo.Process.Kill()
				a.state.ProcessInfo = nil
			}
		}
	}()

	if err := a.monitor.Prepare(a.desc.Spec(), a.pipelineID, a.uid, a.gid); err != nil {
		return err
	}

	if a.limiter != nil {
		a.limiter.Add()
	}

	spec := a.desc.ProcessSpec()
	spec.Args = injectLogLevel(a.logLevel, spec.Args)

	// use separate file
	isSidecar := app.IsSidecar(t)
	spec.Args = a.monitor.EnrichArgs(a.desc.Spec(), a.pipelineID, spec.Args, isSidecar)

	// specify beat name to avoid data lock conflicts
	// as for https://github.com/elastic/beats/v7/pull/14030 more than one instance
	// of the beat with same data path fails to start
	spec.Args = injectDataPath(spec.Args, a.pipelineID, a.id)

	a.state.ProcessInfo, err = process.Start(
		a.logger,
		spec.BinaryPath,
		a.processConfig,
		a.uid,
		a.gid,
		spec.Args)
	if err != nil {
		return err
	}

	// write connect info to stdin
	go a.writeToStdin(a.srvState, a.state.ProcessInfo.Stdin)

	// create closer for watcher, used to terminate watcher without
	// side effect of restarting process during shutdown
	cancelCtx, cancel := context.WithCancel(ctx)
	a.watchClosers[a.state.ProcessInfo.PID] = cancel
	// setup watcher
	a.watch(cancelCtx, t, a.state.ProcessInfo, cfg)

	return nil
}

func (a *Application) writeToStdin(as *server.ApplicationState, wc io.WriteCloser) {
	err := as.WriteConnInfo(wc)
	if err != nil {
		a.logger.Errorf("failed writing connection info to spawned application: %s", err)
	}
	_ = wc.Close()
}

func injectLogLevel(logLevel string, args []string) []string {
	var level string
	// Translate to level beat understands
	switch logLevel {
	case "info":
		level = "info"
	case "debug":
		level = "debug"
	case "warning":
		level = "warning"
	case "error":
		level = "error"
	}

	if args == nil || level == "" {
		return args
	}

	return append(args, "-E", "logging.level="+level)
}

func injectDataPath(args []string, pipelineID, id string) []string {
	dataPath := filepath.Join(paths.Home(), "run", pipelineID, id)
	return append(args, "-E", "path.data="+dataPath)
}
