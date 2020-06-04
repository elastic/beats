// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/plugin/state"

	"gopkg.in/yaml.v2"

	"github.com/elastic/elastic-agent-client/v7/pkg/proto"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/plugin/process"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/server"
)

const (
	configurationFlag     = "-c"
	configFileTempl       = "%s.yml" // providing beat id
	configFilePermissions = 0644     // writable only by owner
)

// Start starts the application with a specified config.
func (a *Application) Start(ctx context.Context, t Taggable, cfg map[string]interface{}) (err error) {
	defer func() {
		if err != nil {
			// inject App metadata
			err = errors.New(err, errors.M(errors.MetaKeyAppName, a.name), errors.M(errors.MetaKeyAppName, a.id))
		}
	}()
	a.appLock.Lock()
	defer a.appLock.Unlock()

	cfgStr, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	a.startContext = ctx
	a.tag = t

	// Failed applications can be started again.
	if a.srvState != nil {
		status, _ := a.srvState.Status()
		if status != proto.StateObserved_FAILED {
			return nil
		}
		a.srvState.SetStatus(proto.StateObserved_STARTING, "Starting")
		a.srvState.UpdateConfig(string(cfgStr))
	} else {
		a.srvState, err = a.srv.Register(a, string(cfgStr))
		if err != nil {
			return err
		}
	}
	if a.state.Status != state.Stopped {
		// restarting as it was previously in a different state
		a.state.Status = state.Restarting
		a.state.Message = "Restarting"
	} else {
		a.state.Status = state.Starting
		a.state.Message = "Starting"
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

	if err := a.monitor.Prepare(a.name, a.pipelineID, a.uid, a.gid); err != nil {
		return err
	}

	if a.limiter != nil {
		a.limiter.Add()
	}

	spec := a.spec.Spec()
	spec.Args = injectLogLevel(a.logLevel, spec.Args)

	// use separate file
	isSidecar := IsSidecar(t)
	spec.Args = a.monitor.EnrichArgs(a.name, a.pipelineID, spec.Args, isSidecar)

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
		spec.Args...)
	if err != nil {
		return err
	}

	err = a.srvState.WriteConnInfo(a.state.ProcessInfo.Stdin)
	if err != nil {
		return err
	}
	err = a.state.ProcessInfo.Stdin.Close()
	if err != nil {
		return err
	}

	// setup watcher
	a.watch(ctx, t, a.state.ProcessInfo, cfg)

	return nil
}

func injectLogLevel(logLevel string, args []string) []string {
	var level string
	// Translate to level beat understands
	switch logLevel {
	case "trace":
		level = "debug"
	case "info":
		level = "info"
	case "debug":
		level = "debug"
	case "error":
		level = "error"
	}

	if args == nil || level == "" {
		return args
	}

	return append(args, "-E", "logging.level="+level)
}

func injectDataPath(args []string, pipelineID, id string) []string {
	dataPath := filepath.Join(paths.Data(), "run", pipelineID, id)
	return append(args, "-E", "path.data="+dataPath)
}

func updateSpecConfig(spec *ProcessSpec, configPath string) error {
	// check if config is already provided
	configIndex := -1
	for i, v := range spec.Args {
		if v == configurationFlag {
			configIndex = i
			break
		}
	}

	if configIndex != -1 {
		// -c provided
		if len(spec.Args) == configIndex+1 {
			// -c is last argument, appending
			spec.Args = append(spec.Args, configPath)
		}
		spec.Args[configIndex+1] = configPath
		return nil
	}

	spec.Args = append(spec.Args, configurationFlag, configPath)
	return nil
}

func changeOwner(path string, uid, gid int) error {
	if runtime.GOOS == "windows" {
		// on windows it always returns the syscall.EWINDOWS error, wrapped in *PathError
		return nil
	}

	return os.Chown(path, uid, gid)
}
