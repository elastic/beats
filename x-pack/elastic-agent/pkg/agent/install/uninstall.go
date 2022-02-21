// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package install

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/kardianos/service"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/info"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/transpiler"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/uninstall"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/capabilities"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/composable"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config/operations"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/app"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/status"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/release"
)

const (
	inputsKey  = "inputs"
	outputsKey = "outputs"
)

// Uninstall uninstalls persistently Elastic Agent on the system.
func Uninstall(cfgFile string) error {
	// uninstall the current service
	svc, err := newService()
	if err != nil {
		return err
	}
	status, _ := svc.Status()
	if status == service.StatusRunning {
		err := svc.Stop()
		if err != nil {
			return errors.New(
				err,
				fmt.Sprintf("failed to stop service (%s)", paths.ServiceName),
				errors.M("service", paths.ServiceName))
		}
		status = service.StatusStopped
	}
	_ = svc.Uninstall()

	if err := uninstallPrograms(context.Background(), cfgFile); err != nil {
		return err
	}

	// remove, if present on platform
	if paths.ShellWrapperPath != "" {
		err = os.Remove(paths.ShellWrapperPath)
		if !os.IsNotExist(err) && err != nil {
			return errors.New(
				err,
				fmt.Sprintf("failed to remove shell wrapper (%s)", paths.ShellWrapperPath),
				errors.M("destination", paths.ShellWrapperPath))
		}
	}

	// remove existing directory
	err = os.RemoveAll(paths.InstallPath)
	if err != nil {
		if runtime.GOOS == "windows" {
			// possible to fail on Windows, because elastic-agent.exe is running from
			// this directory.
			return nil
		}
		return errors.New(
			err,
			fmt.Sprintf("failed to remove installation directory (%s)", paths.InstallPath),
			errors.M("directory", paths.InstallPath))
	}

	return nil
}

// RemovePath helps with removal path where there is a probability
// of running into self which might prevent removal.
// Removal will be initiated 2 seconds after a call.
func RemovePath(path string) error {
	cleanupErr := os.RemoveAll(path)
	if cleanupErr != nil && isBlockingOnSelf(cleanupErr) {
		delayedRemoval(path)
	}

	return cleanupErr
}

func isBlockingOnSelf(err error) bool {
	// cannot remove self, this is expected on windows
	// fails with  remove {path}}\elastic-agent.exe: Access is denied
	return runtime.GOOS == "windows" &&
		err != nil &&
		strings.Contains(err.Error(), "elastic-agent.exe") &&
		strings.Contains(err.Error(), "Access is denied")
}

func delayedRemoval(path string) {
	// The installation path will still exists because we are executing from that
	// directory. So cmd.exe is spawned that sleeps for 2 seconds (using ping, recommend way
	// from Windows) then rmdir is performed.
	rmdir := exec.Command(
		filepath.Join(os.Getenv("windir"), "system32", "cmd.exe"),
		"/C", "ping", "-n", "2", "127.0.0.1", "&&", "rmdir", "/s", "/q", path)
	_ = rmdir.Start()
}

func uninstallPrograms(ctx context.Context, cfgFile string) error {
	log, err := logger.NewWithLogpLevel("", logp.ErrorLevel, false)
	if err != nil {
		return err
	}

	cfg, err := operations.LoadFullAgentConfig(cfgFile, false)
	if err != nil {
		return err
	}

	cfg, err = applyDynamics(ctx, log, cfg)
	if err != nil {
		return err
	}

	pp, err := programsFromConfig(cfg)
	if err != nil {
		return err
	}

	// nothing to remove
	if len(pp) == 0 {
		return nil
	}

	uninstaller, err := uninstall.NewUninstaller()
	if err != nil {
		return err
	}

	currentVersion := release.Version()
	if release.Snapshot() {
		currentVersion = fmt.Sprintf("%s-SNAPSHOT", currentVersion)
	}
	artifactConfig := artifact.DefaultConfig()

	for _, p := range pp {
		descriptor := app.NewDescriptor(p.Spec, currentVersion, artifactConfig, nil)
		if err := uninstaller.Uninstall(ctx, p.Spec, currentVersion, descriptor.Directory()); err != nil {
			fmt.Printf("failed to uninstall '%s': %v\n", p.Spec.Name, err)
		}
	}

	return nil
}

func programsFromConfig(cfg *config.Config) ([]program.Program, error) {
	mm, err := cfg.ToMapStr()
	if err != nil {
		return nil, errors.New("failed to create a map from config", err)
	}

	// if no input is defined nothing to remove
	if _, found := mm[inputsKey]; !found {
		return nil, nil
	}

	// if no output is defined nothing to remove
	if _, found := mm[outputsKey]; !found {
		return nil, nil
	}

	ast, err := transpiler.NewAST(mm)
	if err != nil {
		return nil, errors.New("failed to create a ast from config", err)
	}

	agentInfo, err := info.NewAgentInfo(false)
	if err != nil {
		return nil, errors.New("failed to get an agent info", err)
	}

	ppMap, err := program.Programs(agentInfo, ast)
	if err != nil {
		return nil, errors.New("failed to get programs from config", err)
	}

	var pp []program.Program
	check := make(map[string]bool)

	for _, v := range ppMap {
		for _, p := range v {
			if _, found := check[p.Spec.Cmd]; found {
				continue
			}

			pp = append(pp, p)
			check[p.Spec.Cmd] = true
		}
	}

	return pp, nil
}

func applyDynamics(ctx context.Context, log *logger.Logger, cfg *config.Config) (*config.Config, error) {
	cfgMap, err := cfg.ToMapStr()
	if err != nil {
		return nil, err
	}

	ast, err := transpiler.NewAST(cfgMap)
	if err != nil {
		return nil, err
	}

	// apply dynamic inputs
	inputs, ok := transpiler.Lookup(ast, "inputs")
	if ok {
		varsArray := make([]*transpiler.Vars, 0)
		var wg sync.WaitGroup
		wg.Add(1)
		varsCallback := func(vv []*transpiler.Vars) {
			varsArray = vv
			wg.Done()
		}

		ctrl, err := composable.New(log, cfg)
		if err != nil {
			return nil, err
		}
		ctrl.Run(ctx, varsCallback)
		wg.Wait()

		renderedInputs, err := transpiler.RenderInputs(inputs, varsArray)
		if err != nil {
			return nil, err
		}
		err = transpiler.Insert(ast, renderedInputs, "inputs")
		if err != nil {
			return nil, errors.New("inserting rendered inputs failed", err)
		}
	}

	// apply caps
	caps, err := capabilities.Load(paths.AgentCapabilitiesPath(), log, status.NewController(log))
	if err != nil {
		return nil, err
	}

	astIface, err := caps.Apply(ast)
	if err != nil {
		return nil, err
	}

	newAst, ok := astIface.(*transpiler.AST)
	if ok {
		ast = newAst
	}

	finalConfig, err := newAst.Map()
	if err != nil {
		return nil, err
	}

	return config.NewConfigFrom(finalConfig)
}
