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

package dashboards

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/kibana"
	"github.com/elastic/beats/libbeat/logp"
)

var importAPI = "/api/kibana/dashboards/import"

type KibanaLoader struct {
	client       *kibana.Client
	config       *Config
	version      common.Version
	hostname     string
	msgOutputter MessageOutputter
}

// NewKibanaLoader creates a new loader to load Kibana files
func NewKibanaLoader(ctx context.Context, cfg *common.Config, dashboardsConfig *Config, hostname string, msgOutputter MessageOutputter) (*KibanaLoader, error) {

	if cfg == nil || !cfg.Enabled() {
		return nil, fmt.Errorf("Kibana is not configured or enabled")
	}

	client, err := getKibanaClient(ctx, cfg, dashboardsConfig.Retry, 0)
	if err != nil {
		return nil, fmt.Errorf("Error creating Kibana client: %v", err)
	}

	loader := KibanaLoader{
		client:       client,
		config:       dashboardsConfig,
		version:      client.GetVersion(),
		hostname:     hostname,
		msgOutputter: msgOutputter,
	}

	version := client.GetVersion()
	loader.statusMsg("Initialize the Kibana %s loader", version.String())

	return &loader, nil
}

func getKibanaClient(ctx context.Context, cfg *common.Config, retryCfg *Retry, retryAttempt uint) (*kibana.Client, error) {
	client, err := kibana.NewKibanaClient(cfg)
	if err != nil {
		if retryCfg.Enabled && (retryCfg.Maximum == 0 || retryCfg.Maximum > retryAttempt) {
			select {
			case <-ctx.Done():
				return nil, err
			case <-time.After(retryCfg.Interval):
				return getKibanaClient(ctx, cfg, retryCfg, retryAttempt+1)
			}
		}
		return nil, fmt.Errorf("Error creating Kibana client: %v", err)
	}
	return client, nil
}

// ImportIndexFile imports an index pattern from a file
func (loader KibanaLoader) ImportIndexFile(file string) error {
	// read json file
	reader, err := ioutil.ReadFile(file)
	if err != nil {
		return fmt.Errorf("fail to read index-pattern from file %s: %v", file, err)
	}

	var indexContent common.MapStr
	err = json.Unmarshal(reader, &indexContent)
	if err != nil {
		return fmt.Errorf("fail to unmarshal the index content from file %s: %v", file, err)
	}

	return loader.ImportIndex(indexContent)
}

// ImportIndex imports the passed index pattern to Kibana
func (loader KibanaLoader) ImportIndex(pattern common.MapStr) error {
	params := url.Values{}
	params.Set("force", "true") //overwrite the existing dashboards

	indexContent := ReplaceIndexInIndexPattern(loader.config.Index, pattern)
	return loader.client.ImportJSON(importAPI, params, indexContent)
}

// ImportDashboard imports the dashboard file
func (loader KibanaLoader) ImportDashboard(file string) error {
	params := url.Values{}
	params.Set("force", "true")            //overwrite the existing dashboards
	params.Add("exclude", "index-pattern") //don't import the index pattern from the dashboards

	// read json file
	reader, err := ioutil.ReadFile(file)
	if err != nil {
		return fmt.Errorf("fail to read dashboard from file %s: %v", file, err)
	}
	var content common.MapStr
	err = json.Unmarshal(reader, &content)
	if err != nil {
		return fmt.Errorf("fail to unmarshal the dashboard content from file %s: %v", file, err)
	}

	content = ReplaceIndexInDashboardObject(loader.config.Index, content)

	content, err = ReplaceStringInDashboard("CHANGEME_HOSTNAME", loader.hostname, content)
	if err != nil {
		return fmt.Errorf("fail to replace the hostname in dashboard %s: %v", file, err)
	}

	return loader.client.ImportJSON(importAPI, params, content)
}

func (loader KibanaLoader) Close() error {
	return loader.client.Close()
}

func (loader KibanaLoader) statusMsg(msg string, a ...interface{}) {
	if loader.msgOutputter != nil {
		loader.msgOutputter(msg, a...)
	} else {
		logp.Debug("dashboards", msg, a...)
	}
}
