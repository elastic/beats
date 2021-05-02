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

func NewDiag(beat *instance.Beat, config map[string]interface{}) Diagnostics {
	ctx, cancel := context.WithCancel(context.Background())
	diag := Diagnostics{
		DiagStart: time.Now(),
		Metrics:   Metrics{},
		HTTP: HTTP{
			Client:   nil,
			Protocol: "",
			Host:     "",
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
		Docker: Docker{
			IsContainer: false,
		},
	}
	foldername := diag.createFolderAndFiles()
	diag.DiagFolder = foldername
	return diag
}

func (d *Diagnostics) GetInfo() {
	d.copyBeatConfig()
	d.copyModuleConfig()
	d.getBeatInfo()
	d.copyBeatLogs()
	d.createManifest()
}

func (d *Diagnostics) GetMonitor() {
	d.copyBeatConfig()
	d.copyModuleConfig()
	d.getBeatInfo()
	d.createManifest()

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

	fmt.Fprintf(os.Stdout, "Starting collection of Metrics, Press CTRL+C to stop\n")
	for {
		select {
		case <-shutdown:
			d.CancelFunc()
		case <-ticker.C:
			d.getMetrics()
		case <-d.Context.Done():
			fmt.Fprintf(os.Stdout, "Shutting Down\n")
			d.copyBeatLogs()
			os.Exit(1)
		case <-timer.C:
			fmt.Fprintf(os.Stdout, "Shutting Down\n")
			d.copyBeatLogs()
			os.Exit(1)
		}
	}
}

func (d *Diagnostics) GetProfile() {
	return
}

func (d *Diagnostics) CreateHTTPclient() *http.Client {
	c := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, d.HTTP.Protocol, d.HTTP.Host)
			},
		},
	}
	return c
}

func (d *Diagnostics) apiRequest(url string) map[string]interface{} {
	body := make(map[string]interface{})
	req, _ := http.NewRequest("GET", url, nil)
	res, err := d.HTTP.Client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to call beats api: %s\n", err)
		return nil
	}
	defer res.Body.Close()
	json.NewDecoder(res.Body).Decode(&body)
	return body
}
