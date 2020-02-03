// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux darwin

package operation

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	operatorCfg "github.com/elastic/beats/x-pack/agent/pkg/agent/operation/config"
	"github.com/elastic/beats/x-pack/agent/pkg/agent/program"
	"github.com/elastic/beats/x-pack/agent/pkg/agent/stateresolver"
	"github.com/elastic/beats/x-pack/agent/pkg/artifact"
	"github.com/elastic/beats/x-pack/agent/pkg/artifact/download"
	"github.com/elastic/beats/x-pack/agent/pkg/artifact/install"
	"github.com/elastic/beats/x-pack/agent/pkg/config"
	"github.com/elastic/beats/x-pack/agent/pkg/core/logger"
	"github.com/elastic/beats/x-pack/agent/pkg/core/plugin/app"
	"github.com/elastic/beats/x-pack/agent/pkg/core/plugin/app/monitoring"
	"github.com/elastic/beats/x-pack/agent/pkg/core/plugin/process"
	"github.com/elastic/beats/x-pack/agent/pkg/core/plugin/retry"
	"github.com/elastic/beats/x-pack/agent/pkg/core/plugin/state"
)

var installPath = "tests/scripts"

func TestMain(m *testing.M) {
	// init supported with test cases
	shortSpec := program.Spec{
		Name:         "short",
		Cmd:          "/bin/echo",
		Configurable: "file",
		Args:         []string{"123"},
	}
	longSpec := program.Spec{
		Name:         "long",
		Cmd:          "/bin/sh",
		Configurable: "file",
		Args:         []string{"-c", "echo 123; sleep 100"},
	}
	configurableSpec := program.Spec{
		Name:         "configurable",
		Cmd:          "configurable",
		Configurable: "file",
		Args:         []string{},
	}
	configByFileSpec := program.Spec{
		Name:         "configurablebyfile",
		Cmd:          "configurablebyfile",
		Configurable: "file",
		Args:         []string{},
	}

	program.Supported = append(program.Supported, shortSpec, longSpec, configurableSpec, configByFileSpec)
}

func TestNotSupported(t *testing.T) {
	p := getProgram("notsupported", "1.0")

	operator, _ := getTestOperator(t, "tests/scripts")
	err := operator.start(p, nil)
	if err == nil {
		t.Fatal("was expecting error but got none")
	}
}

func TestShortRun(t *testing.T) {
	p := getProgram("short", "1.0")

	operator, _ := getTestOperator(t, "tests/scripts")
	if err := operator.start(p, nil); err != nil {
		t.Fatal(err)
	}

	// let the watcher kick in
	<-time.After(1 * time.Second)

	items := operator.State()
	if len(items) == 1 && items[p.ID()].Status == state.Running {
		t.Fatalf("Process reattach info not stopped %#v, %+v", items, items[p.ID()].Status)
	}

	os.Remove(filepath.Join(operator.config.DownloadConfig.InstallPath, "short--1.0.yml"))
}

func TestShortRunInvalid(t *testing.T) {
	p := getProgram("bumblebee", "")
	operator, _ := getTestOperator(t, "/bin")
	if err := operator.start(p, nil); err == nil {
		t.Fatal(err)
	}

	// let the watcher kick in
	<-time.After(1 * time.Second)

	items := operator.State()
	if len(items) == 1 && items[p.ID()].Status == state.Running {
		t.Fatalf("Process reattach info not stopped %#v, %+v", items, items[p.ID()].Status)
	}
}

func TestLongRunWithStop(t *testing.T) {
	p := getProgram("long", "1.0")

	operator, _ := getTestOperator(t, "tests/scripts")
	if err := operator.start(p, nil); err != nil {
		t.Fatal(err)
	}

	// wait for watcher so we know it was now cancelled immediately
	<-time.After(1 * time.Second)

	items := operator.State()
	item0, ok := items[p.ID()]
	if !ok || item0.Status != state.Running {
		t.Fatalf("Process not running %#v", items)
	}

	pid := item0.ProcessInfo.PID

	// stop the process
	if err := operator.stop(p); err != nil {
		t.Fatalf("Failed to stop process with PID %d: %v", pid, err)
	}

	// let the watcher kick in
	<-time.After(1 * time.Second)

	// check state updated
	items = operator.State()
	item1, ok := items[p.ID()]
	if !ok || item1.Status == state.Running {
		t.Fatalf("Process state says running after Stop %#v", items)
	}

	// check process stopped
	proc, err := os.FindProcess(pid)
	if err != nil && proc != nil {
		t.Fatal("Process found")
	}
}

func TestLongRunWithCrash(t *testing.T) {
	p := getProgram("long", "1.0")

	operator, _ := getTestOperator(t, "tests/scripts")
	if err := operator.start(p, nil); err != nil {
		t.Fatal(err)
	}

	// wait for watcher so we know it was now cancelled immediately
	<-time.After(1 * time.Second)

	items := operator.State()
	item0, ok := items[p.ID()]
	if !ok || item0.Status != state.Running {
		t.Fatalf("Process not running %#v", items)
	}

	// crash the process
	pid := item0.ProcessInfo.PID
	proc, err := os.FindProcess(pid)
	if err != nil {
		t.Fatalf("Failed to get process with PID %d: %v", pid, err)
	}
	if err := proc.Kill(); err != nil {
		t.Fatalf("Failed to kill process with PID %d: %v", pid, err)
	}

	// let the watcher kick in
	<-time.After(3 * time.Second)

	// check process restarted
	items = operator.State()
	item1, ok := items[p.ID()]
	if !ok || item1.Status != state.Running {
		t.Fatalf("Process not present after restart %#v", items)
	}

	newPid := item1.ProcessInfo.PID
	if pid == newPid {
		t.Fatalf("Process not restarted, still with the same PID %d", pid)
	}

	// stop restarted process
	if err := operator.stop(p); err != nil {
		t.Fatalf("Failed to stop restarted process %d: %v", newPid, err)
	}
}

func TestTwoProcesses(t *testing.T) {
	p := getProgram("long", "1.0")

	operator, _ := getTestOperator(t, "tests/scripts")
	if err := operator.start(p, nil); err != nil {
		t.Fatal(err)
	}

	// wait for watcher so we know it was now cancelled immediately
	<-time.After(1 * time.Second)

	items := operator.State()
	item0, ok := items[p.ID()]
	if !ok || item0.Status != state.Running {
		t.Fatalf("Process not running %#v", items)
	}

	// start the same process again
	if err := operator.start(p, nil); err != nil {
		t.Fatal(err)
	}

	// let the watcher kick in
	<-time.After(1 * time.Second)

	items = operator.State()
	item1, ok := items[p.ID()]
	if !ok || item0.Status != state.Running {
		t.Fatalf("Process not running %#v", items)
	}

	if item0.ProcessInfo.PID != item1.ProcessInfo.PID {
		t.Fatal("Process got updated, expected the same")
	}

	// check process restarted
	operator.stop(p)
}

func TestConfigurableRun(t *testing.T) {
	p := getProgram("configurable", "1.0")

	spec := p.Spec()
	if s, err := os.Stat(spec.BinaryPath); err != nil || s == nil {
		t.Fatalf("binary not available %s", spec.BinaryPath)
	} else {
		t.Logf("found file %v", spec.BinaryPath)
	}

	operator, _ := getTestOperator(t, installPath)
	if err := operator.start(p, nil); err != nil {
		t.Fatal(err)
	}

	// wait for watcher so we know it was now cancelled immediately
	<-time.After(1 * time.Second)

	items := operator.State()
	item0, ok := items[p.ID()]
	if !ok || item0.Status != state.Running {
		t.Fatalf("Process not running %#v", items)
	}

	pid := item0.ProcessInfo.PID

	// check it is still running
	<-time.After(2 * time.Second)

	items = operator.State()
	item1, ok := items[p.ID()]
	if !ok || item1.Status != state.Running {
		t.Fatalf("Process stopped running %#v", items)
	}

	newPID := item1.ProcessInfo.PID
	if pid != newPID {
		t.Fatalf("Process crashed in between first pid: '%v' second pid: '%v'", pid, newPID)
	}

	// try to configure
	cfg := make(map[string]interface{})
	tstFilePath := filepath.Join(os.TempDir(), fmt.Sprintf("tmp%d", rand.Uint32()))
	cfg["TestFile"] = tstFilePath
	if err := operator.pushConfig(p, cfg); err != nil {
		t.Fatalf("failed to config: %v", err)
	}

	if s, err := os.Stat(tstFilePath); err != nil || s == nil {
		t.Fatalf("failed to create a file using Config call %s", tstFilePath)
	}

	// stop the process
	if err := operator.stop(p); err != nil {
		t.Fatalf("Failed to stop process with PID %d: %v", pid, err)
	}

	// let the watcher kick in
	<-time.After(1 * time.Second)

	// check reattach collection cleaned up
	items = operator.State()
	item2, ok := items[p.ID()]
	if !ok || item2.Status == state.Running {
		t.Fatalf("Process still running after stop %#v", items)
	}

	// check process stopped
	proc, err := os.FindProcess(pid)
	if err != nil && proc != nil {
		t.Fatal("Process found")
	}
}

func TestConfigurableByFileRun(t *testing.T) {
	cfg := make(map[string]interface{})
	cfg["TestFile"] = "tstFilePath"
	downloadCfg := &artifact.Config{
		InstallPath:     installPath,
		OperatingSystem: "darwin",
	}

	p := app.NewDescriptor("configurablebyfile", "1.0", downloadCfg, nil)
	installPath := "tests/scripts"
	spec := p.Spec()
	if s, err := os.Stat(spec.BinaryPath); err != nil || s == nil {
		t.Fatalf("binary not available %s", spec.BinaryPath)
	} else {
		t.Logf("found file %v", spec.BinaryPath)
	}

	operator, _ := getTestOperator(t, installPath)
	if err := operator.start(p, cfg); err != nil {
		t.Fatal(err)
	}

	// wait for watcher so we know it was now cancelled immediately
	<-time.After(1 * time.Second)

	items := operator.State()
	item0, ok := items[p.ID()]
	if !ok || item0.Status != state.Running {
		t.Fatalf("Process not running %#v", items)
	}

	// check it is still running
	<-time.After(2 * time.Second)

	items = operator.State()
	item1, ok := items[p.ID()]
	if !ok || item1.Status != state.Running {
		t.Fatalf("Process not running anymore %#v", items)
	}

	if item0.ProcessInfo.PID != item1.ProcessInfo.PID {
		t.Fatalf("Process crashed in between first pid: '%v' second pid: '%v'", item0.ProcessInfo.PID, item1.ProcessInfo.PID)
	}

	// stop the process
	if err := operator.stop(p); err != nil {
		t.Fatalf("Failed to stop process with PID %d: %v", item1.ProcessInfo.PID, err)
	}

	// let the watcher kick in
	<-time.After(1 * time.Second)

	// check reattach collection cleaned up
	items = operator.State()
	item2, ok := items[p.ID()]
	if !ok || item2.Status == state.Running {
		t.Fatalf("Process still running after stop %#v", items)
	}

	// check process stopped
	proc, err := os.FindProcess(item1.ProcessInfo.PID)
	if err != nil && proc != nil {
		t.Fatal("Process found")
	}
}

func getTestOperator(t *testing.T, installPath string) (*Operator, *operatorCfg.Config) {
	operatorConfig := &operatorCfg.Config{
		RetryConfig: &retry.Config{
			Enabled:      true,
			RetriesCount: 2,
			Delay:        3 * time.Second,
			MaxDelay:     10 * time.Second,
		},
		ProcessConfig: &process.Config{},
		DownloadConfig: &artifact.Config{
			InstallPath: installPath,
		},
		MonitoringConfig: &monitoring.Config{
			MonitorMetrics: false,
		},
	}

	cfg, err := config.NewConfigFrom(operatorConfig)
	if err != nil {
		t.Fatal(err)
	}

	l := getLogger()

	fetcher := &DummyDownloader{}
	installer := &DummyInstaller{}

	stateResolver, err := stateresolver.NewStateResolver(l)
	if err != nil {
		t.Fatal(err)
	}

	operator, err := NewOperator(context.Background(), l, "p1", cfg, fetcher, installer, stateResolver, nil)
	if err != nil {
		t.Fatal(err)
	}

	operator.config.DownloadConfig.OperatingSystem = "darwin"
	operator.config.DownloadConfig.Architecture = "32"

	return operator, operatorConfig
}

func getLogger() *logger.Logger {
	l, _ := logger.New()
	return l
}

func getProgram(binary, version string) *app.Descriptor {
	downloadCfg := &artifact.Config{
		InstallPath:     installPath,
		OperatingSystem: "darwin",
	}
	return app.NewDescriptor(binary, version, downloadCfg, nil)
}

type TestConfig struct {
	TestFile string
}

type DummyDownloader struct {
}

func (*DummyDownloader) Download(_ context.Context, p, v string) (string, error) {
	return "", nil
}

var _ download.Downloader = &DummyDownloader{}

type DummyInstaller struct {
}

func (*DummyInstaller) Install(p, v, _ string) error {
	return nil
}

var _ install.Installer = &DummyInstaller{}
