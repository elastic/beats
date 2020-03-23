// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package operation

import (
	"context"
	"testing"
	"time"

	operatorCfg "github.com/elastic/beats/v7/x-pack/agent/pkg/agent/operation/config"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/agent/stateresolver"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/artifact"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/artifact/download"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/artifact/install"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/core/plugin/app"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/core/plugin/app/monitoring"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/core/plugin/process"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/core/plugin/retry"
)

var installPath = "tests/scripts"

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
