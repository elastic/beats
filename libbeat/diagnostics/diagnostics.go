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
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/elastic/beats/v7/libbeat/cmd/instance"
)

// Initializes a new instace of Diagnostics
func NewDiag(beat *instance.Beat, config map[string]interface{}) Diagnostics {
	ctx, cancel := context.WithCancel(context.Background())
	diag := Diagnostics{
		DiagStart: time.Now(),
		Metrics:   Metrics{},
		Type:      "",
		Interval:  "",
		Duration:  "",
		API: API{
			Client:      nil,
			NpipeClient: "",
			Protocol:    "",
			Host:        "",
		},
		Context:    ctx,
		CancelFunc: cancel,
		Beat: Beat{
			Info:       beat.Info,
			State:      State{},
			ConfigPath: config["path"].(map[string]interface{})["config"].(string),
			LogPath:    config["path"].(map[string]interface{})["logs"].(string),
			ModulePath: strings.TrimSuffix(config["filebeat"].(map[string]interface{})["config"].(map[string]interface{})["modules"].(map[string]interface{})["path"].(string), "/*.yml"),
		},
		// TODO, Currently does nothing, as docker tasks has been removed currently, might remove later, currently a placeholder
		Docker: Docker{
			IsContainer: false,
		},
	}
	return diag
}

// Runs all tasks depending on diagnostic type (info, monitoring or profile)
func (d *Diagnostics) Run() {
	// HTTP, unix socket or npipe client should only be created if the user has not disabled it through arguments
	if !d.LogOnly {
		d.createClient()
	}
	d.createFolderAndFiles()

	d.runInfoTasks()
	if d.Type == "monitor" || d.Type == "profile" {
		d.runMonitorTasks()
	}
	if d.Type == "profile" {
		d.runProfileTasks()
	}

	// Tasks that should run for all diagnostic types, and needs to run last
	d.createManifest()
	d.copyBeatLogs()
}

// Collects beat and enabled module configuration files, and optionally metadata from API.
func (d *Diagnostics) runInfoTasks() {
	d.copyBeatConfig()
	d.copyModuleConfig()
	if !d.LogOnly {
		d.getBeatInfo()
	}
}

// Collects beat metrics from HTTP, Unix socket or npipe API from a running beat instance.
// Need to move routine and ctx outside of function so profiling could use it as well.
func (d *Diagnostics) runMonitorTasks() {
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt)

	interval, _ := time.ParseDuration(d.Interval)
	duration, _ := time.ParseDuration(d.Duration)
	ticker := time.NewTicker(interval)
	timer := time.NewTimer(duration)
	defer func() {
		signal.Stop(shutdown)
		defer ticker.Stop()
		defer timer.Stop()
		d.CancelFunc()
	}()

	fmt.Fprintf(os.Stdout, "starting collection of Metrics for with interval: %s and duration: %s, Press CTRL+C to stop\n", interval, duration)
	for {
		select {
		case <-shutdown:
			d.CancelFunc()
		case <-ticker.C:
			d.getMetrics()
		case <-d.Context.Done():
			fmt.Fprintf(os.Stdout, "process cancelled, shutting Down\n")
			d.copyBeatLogs()
			os.Exit(1)
		case <-timer.C:
			fmt.Fprintf(os.Stdout, "duration finished, shutting Down\n")
			d.copyBeatLogs()
			os.Exit(1)
		}
	}
}

// TODO If I want to run profiling and metric collection at the same time, the metric collection needs to go into
// its own goroutine.
func (d *Diagnostics) runProfileTasks() {
	return
}

// Creates an instance of the intended client, depending on protocol choosen by user.
func (d *Diagnostics) createClient() {
	if d.API.Protocol == "npipe" {
		fmt.Fprintf(os.Stderr, "Npipe is currently not supported\n")
		os.Exit(1)
	}
	d.API.Client = &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, d.API.Protocol, d.API.Host)
			},
		},
	}
}

// TODO, does it really need a decoder?
func (d *Diagnostics) apiRequest(url string) map[string]interface{} {
	body := make(map[string]interface{})
	req, _ := http.NewRequest("GET", url, nil)
	res, err := d.API.Client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to call beats api: %s\n", err)
		return nil
	}
	defer res.Body.Close()
	json.NewDecoder(res.Body).Decode(&body)
	return body
}
