// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package beatreceiver

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/api"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cmd/instance"
	"github.com/elastic/beats/v7/libbeat/version"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	metricreport "github.com/elastic/elastic-agent-system-metrics/report"

	"go.uber.org/zap"
)

// BaseReceiver holds common configurations for beatreceivers.
type BeatReceiver struct {
	HttpConf *config.C
	Beat     *instance.Beat
	Beater   beat.Beater
	Logger   *zap.Logger
}

// BeatReceiver.Stop() starts the beat receiver.
func (b *BeatReceiver) Start() error {
	if err := b.startMonitoring(); err != nil {
		return fmt.Errorf("could not start the HTTP server for the monitoring API: %w", err)
	}
	if err := b.Beater.Run(&b.Beat.Beat); err != nil {
		return fmt.Errorf("beat receiver run error: %w", err)
	}
	return nil
}

// BeatReceiver.Stop() stops beat receiver.
func (b *BeatReceiver) Shutdown() error {
	b.Beater.Stop()
	if err := b.stopMonitoring(); err != nil {
		return fmt.Errorf("error stopping monitoring server: %w", err)
	}
	return nil
}

func (b *BeatReceiver) startMonitoring() error {
	if b.HttpConf == nil || !b.HttpConf.Enabled() {
		return nil
	}
	var err error

	b.Beat.RegisterMetrics()

	statsReg := b.Beat.Info.Monitoring.StatsRegistry

	// stats.beat
	processReg := statsReg.GetRegistry("beat")
	if processReg == nil {
		processReg = statsReg.NewRegistry("beat")
	}

	// stats.system
	systemReg := statsReg.GetRegistry("system")
	if systemReg == nil {
		systemReg = statsReg.NewRegistry("system")
	}

	err = metricreport.SetupMetrics(logp.NewLogger("metrics"), b.Beat.Info.Beat, version.GetDefaultVersion(), metricreport.WithProcessRegistry(processReg), metricreport.WithSystemRegistry(systemReg))
	if err != nil {
		return err
	}
	b.Beat.API, err = api.NewWithDefaultRoutes(logp.NewLogger("metrics.http"), b.HttpConf, api.RegistryLookupFunc(b.Beat.Info.Monitoring.Namespace))
	if err != nil {
		return err
	}
	b.Beat.API.Start()

	return nil
}

func (b *BeatReceiver) stopMonitoring() error {
	if b.Beat.API != nil {
		return b.Beat.API.Stop()
	}
	return nil
}
