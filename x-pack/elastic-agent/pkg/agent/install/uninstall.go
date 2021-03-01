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
	// directory. So cmd.exe is spawned that sleeps for 2 seconds (using ping, recommend way from
	// from Windows) then rmdir is performed.
	rmdir := exec.Command(
		filepath.Join(os.Getenv("windir"), "system32", "cmd.exe"),
		"/C", "ping", "-n", "2", "127.0.0.1", "&&", "rmdir", "/s", "/q", path)
	_ = rmdir.Start()

}

func uninstallPrograms(ctx context.Context, cfgFile string) error {
	log, err := logger.NewWithLogpLevel("", logp.ErrorLevel)
	if err != nil {
		return err
	}

	cfg, err := operations.LoadFullAgentConfig(cfgFile)
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
	ast, err := transpiler.NewAST(mm)
	if err != nil {
		return nil, errors.New("failed to create a ast from config", err)
	}

	agentInfo, err := info.NewAgentInfo()
	if err != nil {
		return nil, errors.New("failed to get an agent info", err)
	}

	ppMap, err := program.Programs(agentInfo, ast)

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
	ast, err := transpiler.NewAST(cfgMap)
	if err != nil {
		return nil, err
	}

	// apply dynamic inputs
	inputs, ok := transpiler.Lookup(ast, "inputs")
	if ok {
		varsArray := make([]*transpiler.Vars, 0, 0)
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

		renderedInputs, err := renderInputs(inputs, varsArray)
		if err != nil {
			return nil, err
		}
		err = transpiler.Insert(ast, renderedInputs, "inputs")
		if err != nil {
			return nil, err
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
	return config.NewConfigFrom(finalConfig)
}

// Dynamic inputs section
// TODO(michal): move to shared code during refactoring
func renderInputs(inputs transpiler.Node, varsArray []*transpiler.Vars) (transpiler.Node, error) {
	l, ok := inputs.Value().(*transpiler.List)
	if !ok {
		return nil, fmt.Errorf("inputs must be an array")
	}
	nodes := []*transpiler.Dict{}
	nodesMap := map[string]*transpiler.Dict{}
	for _, vars := range varsArray {
		for _, node := range l.Value().([]transpiler.Node) {
			dict, ok := node.Clone().(*transpiler.Dict)
			if !ok {
				continue
			}
			n, err := dict.Apply(vars)
			if err == transpiler.ErrNoMatch {
				// has a variable that didn't exist, so we ignore it
				continue
			}
			if err != nil {
				// another error that needs to be reported
				return nil, err
			}
			if n == nil {
				// condition removed it
				continue
			}
			dict = n.(*transpiler.Dict)
			hash := string(dict.Hash())
			_, exists := nodesMap[hash]
			if !exists {
				nodesMap[hash] = dict
				nodes = append(nodes, dict)
			}
		}
	}
	nInputs := []transpiler.Node{}
	for _, node := range nodes {
		nInputs = append(nInputs, promoteProcessors(node))
	}
	return transpiler.NewList(nInputs), nil
}

func promoteProcessors(dict *transpiler.Dict) *transpiler.Dict {
	p := dict.Processors()
	if p == nil {
		return dict
	}
	var currentList *transpiler.List
	current, ok := dict.Find("processors")
	if ok {
		currentList, ok = current.Value().(*transpiler.List)
		if !ok {
			return dict
		}
	}
	ast, _ := transpiler.NewAST(map[string]interface{}{
		"processors": p,
	})
	procs, _ := transpiler.Lookup(ast, "processors")
	nodes := nodesFromList(procs.Value().(*transpiler.List))
	if ok && currentList != nil {
		nodes = append(nodes, nodesFromList(currentList)...)
	}
	dictNodes := dict.Value().([]transpiler.Node)
	set := false
	for i, node := range dictNodes {
		switch n := node.(type) {
		case *transpiler.Key:
			if n.Name() == "processors" {
				dictNodes[i] = transpiler.NewKey("processors", transpiler.NewList(nodes))
				set = true
			}
		}
		if set {
			break
		}
	}
	if !set {
		dictNodes = append(dictNodes, transpiler.NewKey("processors", transpiler.NewList(nodes)))
	}
	return transpiler.NewDict(dictNodes)
}

func nodesFromList(list *transpiler.List) []transpiler.Node {
	return list.Value().([]transpiler.Node)
}
