// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux darwin

package operation

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/urso/ecslog"

	"github.com/elastic/fleet/x-pack/pkg/artifact"
	"github.com/elastic/fleet/x-pack/pkg/artifact/download"
	"github.com/elastic/fleet/x-pack/pkg/artifact/install"
	"github.com/elastic/fleet/x-pack/pkg/bus"
	"github.com/elastic/fleet/x-pack/pkg/bus/topic"
	"github.com/elastic/fleet/x-pack/pkg/config"
	"github.com/elastic/fleet/x-pack/pkg/core/logger"
	"github.com/elastic/fleet/x-pack/pkg/core/plugin/clientvault"
	"github.com/elastic/fleet/x-pack/pkg/core/plugin/process"
	"github.com/elastic/fleet/x-pack/pkg/core/plugin/process/watcher"
	rconfig "github.com/elastic/fleet/x-pack/pkg/core/remoteconfig/grpc"
)

func TestShortRun(t *testing.T) {
	p := getProgram("short", "1.0")

	operator, operatorConfig := getTestOperator(t, "tests/scripts")
	if err := operator.Start(p); err != nil {
		t.Fatal(err)
	}

	// let the watcher kick in
	<-time.After(1 * time.Second)

	reattachInfo := newReattachCollection(operatorConfig)
	items, err := reattachInfo.items()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) > 0 {
		t.Fatalf("Process reattach info not removed %#v", items)
	}

	os.Remove(operatorConfig.ReattachCollectionPath)
}

func TestShortRunInvalid(t *testing.T) {
	p := getProgram("bumblebee", "")
	operator, operatorConfig := getTestOperator(t, "/bin")
	if err := operator.Start(p); err == nil {
		t.Fatal(err)
	}

	// let the watcher kick in
	<-time.After(1 * time.Second)

	reattachInfo := newReattachCollection(operatorConfig)
	items, err := reattachInfo.items()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) > 0 {
		t.Fatalf("Process reattach info not removed %#v", items)
	}

	os.Remove(operatorConfig.ReattachCollectionPath)
}

func TestLongRunWithStop(t *testing.T) {
	p := getProgram("long", "1.0")

	operator, operatorConfig := getTestOperator(t, "tests/scripts")
	if err := operator.Start(p); err != nil {
		t.Fatal(err)
	}

	// wait for watcher so we know it was now cancelled immediately
	<-time.After(1 * time.Second)

	reattachInfo := newReattachCollection(operatorConfig)
	items, err := reattachInfo.items()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("Process not running %#v", items)
	}

	pid := items[0].PID

	// stop the process
	if err := operator.Stop(p); err != nil {
		t.Fatalf("Failed to stop process with PID %d: %v", pid, err)
	}

	// let the watcher kick in
	<-time.After(1 * time.Second)

	// check reattach collection cleaned up
	items, err = reattachInfo.items()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) > 0 {
		t.Fatalf("Process still in reattach collection %#v", items)
	}

	// check process stopped
	proc, err := os.FindProcess(pid)
	if err != nil && proc != nil {
		t.Fatal("Process found")
	}

	os.Remove(operatorConfig.ReattachCollectionPath)
}

func TestLongRunWithCrash(t *testing.T) {
	p := getProgram("long", "1.0")

	operator, operatorConfig := getTestOperator(t, "tests/scripts")
	if err := operator.Start(p); err != nil {
		t.Fatal(err)
	}

	// wait for watcher so we know it was now cancelled immediately
	<-time.After(1 * time.Second)

	reattachInfo := newReattachCollection(operatorConfig)
	items, err := reattachInfo.items()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("Process not running %#v", items)
	}

	pid := items[0].PID

	// crash the process
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
	items, err = reattachInfo.items()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("Process not in reattach collection after kill %#v", items)
	}

	newPid := items[0].PID
	if pid == newPid {
		t.Fatalf("Process not restarted, still with the same PID %d", pid)
	}

	// check process restarted
	if err := operator.Stop(p); err != nil {
		t.Fatalf("Failed to stop restarted process %d: %v", newPid, err)
	}

	os.Remove(operatorConfig.ReattachCollectionPath)
}

func TestTwoProcesses(t *testing.T) {
	p := getProgram("long", "1.0")

	operator, operatorConfig := getTestOperator(t, "tests/scripts")
	if err := operator.Start(p); err != nil {
		t.Fatal(err)
	}

	// wait for watcher so we know it was now cancelled immediately
	<-time.After(1 * time.Second)

	reattachInfo := newReattachCollection(operatorConfig)
	items, err := reattachInfo.items()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("Process not running %#v", items)
	}

	pid := items[0].PID

	// start the same process again
	if err := operator.Start(p); err != nil {
		t.Fatal(err)
	}

	// let the watcher kick in
	<-time.After(1 * time.Second)

	items, err = reattachInfo.items()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) > 1 {
		t.Fatal("Two processes running. Expected 1")
	}

	if items[0].PID != pid {
		t.Fatal("Process got updated, expected the same")
	}

	// check process restarted
	operator.Stop(p)

	os.Remove(operatorConfig.ReattachCollectionPath)
}

func TestConfigurableRun(t *testing.T) {
	p := getProgram("configurable", "1.0")
	installPath := "tests/scripts"
	downloadCfg := &artifact.Config{
		InstallPath:     installPath,
		OperatingSystem: "darwin",
	}
	spec, err := p.Spec(downloadCfg)
	if err != nil {
		t.Fatalf("spec not loaded %v", err)
	}

	if s, err := os.Stat(spec.BinaryPath); err != nil || s == nil {
		t.Fatalf("binary not available %s", spec.BinaryPath)
	} else {
		t.Logf("found file %v", spec.BinaryPath)
	}

	operator, operatorConfig := getTestOperator(t, installPath)
	if err := operator.Start(p); err != nil {
		t.Fatal(err)
	}

	// wait for watcher so we know it was now cancelled immediately
	<-time.After(1 * time.Second)

	reattachInfo := newReattachCollection(operatorConfig)
	items, err := reattachInfo.items()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("Process not running %#v", items)
	}

	pid := items[0].PID

	// check it is still running
	<-time.After(2 * time.Second)

	items, err = reattachInfo.items()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("Process not running %#v", items)
	}

	newPID := items[0].PID
	if pid != newPID {
		t.Fatalf("Process crashed in between first pid: '%v' second pid: '%v'", pid, newPID)
	}

	// try to configure
	cfg := p.Config()
	tstFilePath := filepath.Join(os.TempDir(), fmt.Sprintf("tmp%d", rand.Uint32()))
	cfg["TestFile"] = tstFilePath
	if err := operator.PushConfig(p); err != nil {
		t.Fatalf("failed to config: %v", err)
	}

	if s, err := os.Stat(tstFilePath); err != nil || s == nil {
		t.Fatalf("failed to create a file using Config call %s", tstFilePath)
	}

	// stop the process
	if err := operator.Stop(p); err != nil {
		t.Fatalf("Failed to stop process with PID %d: %v", pid, err)
	}

	// let the watcher kick in
	<-time.After(1 * time.Second)

	// check reattach collection cleaned up
	items, err = reattachInfo.items()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) > 0 {
		t.Fatalf("Process still in reattach collection %#v", items)
	}

	// check process stopped
	proc, err := os.FindProcess(pid)
	if err != nil && proc != nil {
		t.Fatal("Process found")
	}

	os.Remove(operatorConfig.ReattachCollectionPath)
}

func TestConfigurableByFileRun(t *testing.T) {
	cfg := make(map[string]interface{})
	cfg["TestFile"] = "tstFilePath"

	p := NewProgram("configurablebyfile", "1.0", cfg, nil)
	installPath := "tests/scripts"
	downloadCfg := &artifact.Config{
		InstallPath:     installPath,
		OperatingSystem: "darwin",
	}
	spec, err := p.Spec(downloadCfg)
	if err != nil {
		t.Fatalf("spec not loaded %v", err)
	}

	if s, err := os.Stat(spec.BinaryPath); err != nil || s == nil {
		t.Fatalf("binary not available %s", spec.BinaryPath)
	} else {
		t.Logf("found file %v", spec.BinaryPath)
	}

	operator, operatorConfig := getTestOperator(t, installPath)
	if err := operator.Start(p); err != nil {
		t.Fatal(err)
	}

	// wait for watcher so we know it was now cancelled immediately
	<-time.After(1 * time.Second)

	reattachInfo := newReattachCollection(operatorConfig)
	items, err := reattachInfo.items()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("Process not running %#v", items)
	}

	pid := items[0].PID

	// check it is still running
	<-time.After(2 * time.Second)

	items, err = reattachInfo.items()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("Process not running %#v", items)
	}

	newPID := items[0].PID
	if pid != newPID {
		t.Fatalf("Process crashed in between first pid: '%v' second pid: '%v'", pid, newPID)
	}

	// stop the process
	if err := operator.Stop(p); err != nil {
		t.Fatalf("Failed to stop process with PID %d: %v", pid, err)
	}

	// let the watcher kick in
	<-time.After(1 * time.Second)

	// check reattach collection cleaned up
	items, err = reattachInfo.items()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) > 0 {
		t.Fatalf("Process still in reattach collection %#v", items)
	}

	// check process stopped
	proc, err := os.FindProcess(pid)
	if err != nil && proc != nil {
		t.Fatal("Process found")
	}

	os.Remove(operatorConfig.ReattachCollectionPath)
}

func getTestOperator(t *testing.T, installPath string) (*Operator, *Config) {
	file, err := ioutil.TempFile("", "reattach")

	if err != nil {
		t.Fatal(err)
	}

	reattachPath := file.Name()
	file.Close()
	os.Remove(reattachPath)

	operatorConfig := &Config{
		ReattachCollectionPath: reattachPath,
		ProcessConfig:          &process.Config{},
		DownloadConfig: &artifact.Config{
			InstallPath:     installPath,
			OperatingSystem: "darwin",
		},
	}

	cfg, err := config.NewConfigFrom(operatorConfig)
	if err != nil {
		t.Fatal(err)
	}

	factory := rconfig.NewConnFactory(1*time.Second, 10*time.Second)
	cv, err := clientvault.NewClientVault(factory)
	if err != nil {
		t.Fatal(err)
	}

	l := getLogger()
	w := watcher.NewProcessWatcher(l)

	eb, err := bus.NewEventBus(l)
	if err != nil {
		t.Fatal(err)
	}

	err = eb.CreateTopic(topic.StateChanges)
	if err != nil {
		t.Fatal(err)
	}

	fetcher := &DummyDownloader{}
	installer := &DummyInstaller{}

	operator, err := NewOperator(l, cv, w, cfg, eb, true, fetcher, installer)
	if err != nil {
		t.Fatal(err)
	}

	return operator, operatorConfig
}

func getLogger() *ecslog.Logger {
	l, _ := logger.New()
	return l
}

func getProgram(binary, version string) Program {
	return NewProgram(binary, version, nil, nil)
}

type TestConfig struct {
	TestFile string
}

type DummyDownloader struct {
}

func (*DummyDownloader) Download(p, v string) (string, error) {
	return "", nil
}

var _ download.Downloader = &DummyDownloader{}

type DummyInstaller struct {
}

func (*DummyInstaller) Install(p, v, _ string) error {
	return nil
}

var _ install.Installer = &DummyInstaller{}
