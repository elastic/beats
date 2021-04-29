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

package diagnostics

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/elastic/beats/v7/libbeat/cmd/instance"
	"github.com/elastic/beats/v7/libbeat/logp"
)

var (
	logName = "diagnostics"
)

func GetInfo(beat *instance.Beat, config map[string]interface{}) {
	ctx, cancel := context.WithCancel(context.Background())

	log := logp.NewLogger(logName)
	diag := Diagnostics{
		DiagStart: time.Now(),
		Context:   ctx,
		Metrics:   Metrics{},
		Logger:    log,
		Beat: Beat{
			Info:       beat.Info,
			ConfigPath: config["path"].(map[string]interface{})["config"].(string),
			LogPath:    config["path"].(map[string]interface{})["logs"].(string),
			ModulePath: strings.TrimSuffix(config["filebeat"].(map[string]interface{})["config"].(map[string]interface{})["modules"].(map[string]interface{})["path"].(string), "/*.yml"),
		},
		Docker: Docker{
			IsContainer: false,
		},
	}
	foldername := createFiles(&diag)
	diag.DiagFolder = foldername
	getBeatInfo(&diag)
	copyBeatConfig(&diag)
	copyModuleConfig(&diag)
	getHostInfo(&diag)

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt)

	go func() {
		select {
		case <-shutdown:
			cancel()
		case <-ctx.Done():
		}
		<-shutdown
		os.Exit(2)
	}()
	var interval = time.Duration(10) * time.Second
	ticker := time.NewTicker(time.Duration(interval))
	defer func() {
		signal.Stop(shutdown)
		defer ticker.Stop()
		cancel()
	}()
	log.Info("Starting collection of Metrics")
	for {
		select {
		case <-ticker.C:
			getMetrics(diag)
		case <-ctx.Done():
			log.Info("Shutting Down")
			copyBeatLogs(&diag)
			return
		}
	}
}

func GetMonitor() {
	return
}

func GetProfile() {
	return
}
