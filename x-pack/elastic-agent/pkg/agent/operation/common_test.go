// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package operation

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"go.elastic.co/apm"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/info"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/configuration"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/stateresolver"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/download"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/install"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/uninstall"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/app"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/monitoring/noop"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/process"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/retry"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/server"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/status"
)

var downloadPath = getAbsPath("tests/downloads")
var installPath = getAbsPath("tests/scripts")

func getTestOperator(t *testing.T, downloadPath string, installPath string, p *app.Descriptor) *Operator {
	operatorCfg := &configuration.SettingsConfig{
		RetryConfig: &retry.Config{
			Enabled:      true,
			RetriesCount: 2,
			Delay:        3 * time.Second,
			MaxDelay:     10 * time.Second,
		},
		ProcessConfig: &process.Config{
			FailureTimeout: 1, // restart instantly
		},
		DownloadConfig: &artifact.Config{
			TargetDirectory: downloadPath,
			InstallPath:     installPath,
		},
		LoggingConfig: logger.DefaultLoggingConfig(),
	}

	l := getLogger()
	agentInfo, _ := info.NewAgentInfo(true)

	fetcher := &DummyDownloader{}
	verifier := &DummyVerifier{}
	installer := &DummyInstallerChecker{}
	uninstaller := &DummyUninstaller{}

	stateResolver, err := stateresolver.NewStateResolver(l)
	if err != nil {
		t.Fatal(err)
	}
	srv, err := server.New(l, "localhost:0", &ApplicationStatusHandler{}, apm.DefaultTracer)
	if err != nil {
		t.Fatal(err)
	}
	err = srv.Start()
	if err != nil {
		t.Fatal(err)
	}

	operator, err := NewOperator(context.Background(), l, agentInfo, "p1", operatorCfg, fetcher, verifier, installer, uninstaller, stateResolver, srv, nil, noop.NewMonitor(), status.NewController(l))
	if err != nil {
		t.Fatal(err)
	}

	operator.config.DownloadConfig.OperatingSystem = "darwin"
	operator.config.DownloadConfig.Architecture = "64"

	// make the download path so the `operation_verify` can ensure the path exists
	downloadConfig := operator.config.DownloadConfig
	fullPath, err := artifact.GetArtifactPath(p.Spec(), p.Version(), downloadConfig.OS(), downloadConfig.Arch(), downloadConfig.TargetDirectory)
	if err != nil {
		t.Fatal(err)
	}
	createFile(t, fullPath)

	return operator
}

func getLogger() *logger.Logger {
	loggerCfg := logger.DefaultLoggingConfig()
	loggerCfg.Level = logp.ErrorLevel
	l, _ := logger.NewFromConfig("", loggerCfg, false)
	return l
}

func getProgram(binary, version string) *app.Descriptor {
	spec := program.SupportedMap[binary]
	downloadCfg := &artifact.Config{
		InstallPath:     installPath,
		OperatingSystem: "darwin",
		Architecture:    "64",
	}
	return app.NewDescriptor(spec, version, downloadCfg, nil)
}

func getAbsPath(path string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), path)
}

func createFile(t *testing.T, path string) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		file, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
	}
}

func waitFor(t *testing.T, check func() error) {
	started := time.Now()
	for {
		err := check()
		if err == nil {
			return
		}
		if time.Since(started) >= 15*time.Second {
			t.Fatalf("check timed out after 15 second: %s", err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

type DummyDownloader struct{}

func (*DummyDownloader) Download(_ context.Context, _ program.Spec, _ string) (string, error) {
	return "", nil
}

var _ download.Downloader = &DummyDownloader{}

type DummyVerifier struct{}

func (*DummyVerifier) Verify(_ program.Spec, _ string, _ bool) (bool, error) {
	return true, nil
}

var _ download.Verifier = &DummyVerifier{}

type DummyInstallerChecker struct{}

func (*DummyInstallerChecker) Check(_ context.Context, _ program.Spec, _, _ string) error {
	return nil
}

func (*DummyInstallerChecker) Install(_ context.Context, _ program.Spec, _, _ string) error {
	return nil
}

var _ install.InstallerChecker = &DummyInstallerChecker{}

type DummyUninstaller struct{}

func (*DummyUninstaller) Uninstall(_ context.Context, _ program.Spec, _, _ string) error {
	return nil
}

var _ uninstall.Uninstaller = &DummyUninstaller{}
