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
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	beatversion "github.com/elastic/beats/v7/libbeat/version"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/kibana"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/version"
)

var (
	// We started using Saved Objects API in 7.15. But to help integration
	// developers migrate their dashboards we are more lenient.
	minimumRequiredVersionSavedObjects = version.MustNew("7.14.0")

	// the base path of the saved objects API
	// On serverless, you must add an x-elastic-internal-header to reach this API
	importAPI = "/api/saved_objects/_import"
)

// KibanaLoader loads Kibana files
type KibanaLoader struct {
	client       *kibana.Client
	config       *Config
	version      version.V
	hostname     string
	msgOutputter MessageOutputter
	logger       *logp.Logger

	loadedAssets map[string]bool
}

// NewKibanaLoader creates a new loader to load Kibana files
func NewKibanaLoader(ctx context.Context, cfg *config.C, dashboardsConfig *Config, msgOutputter MessageOutputter, beatInfo beat.Info) (*KibanaLoader, error) {
	if cfg == nil || !cfg.Enabled() {
		return nil, fmt.Errorf("kibana is not configured or enabled")
	}

	client, err := getKibanaClient(ctx, cfg, dashboardsConfig.Retry, 0, beatInfo.Beat)
	if err != nil {
		return nil, fmt.Errorf("Error creating Kibana client: %w", err)
	}

	loader := KibanaLoader{
		client:       client,
		config:       dashboardsConfig,
		version:      client.GetVersion(),
		hostname:     beatInfo.Hostname,
		msgOutputter: msgOutputter,
		logger:       beatInfo.Logger.Named("dashboards"),
		loadedAssets: make(map[string]bool, 0),
	}

	version := client.GetVersion()
	loader.statusMsg("Initialize the Kibana %s loader", version.String())

	return &loader, nil
}

func getKibanaClient(ctx context.Context, cfg *config.C, retryCfg *Retry, retryAttempt uint, beatname string) (*kibana.Client, error) {
	client, err := kibana.NewKibanaClient(cfg, beatname, beatversion.GetDefaultVersion(), beatversion.Commit(), beatversion.BuildTime().String())
	if err != nil {
		if retryCfg.Enabled && (retryCfg.Maximum == 0 || retryCfg.Maximum > retryAttempt) {
			select {
			case <-ctx.Done():
				return nil, err
			case <-time.After(retryCfg.Interval):
				return getKibanaClient(ctx, cfg, retryCfg, retryAttempt+1, beatname)
			}
		}
		return nil, fmt.Errorf("Error creating Kibana client: %w", err)
	}
	return client, nil
}

// ImportIndexFile imports an index pattern from a file
func (loader KibanaLoader) ImportIndexFile(file string) error {
	if loader.version.LessThan(minimumRequiredVersionSavedObjects) {
		return fmt.Errorf("Kibana version must be at least %s", minimumRequiredVersionSavedObjects.String()) //nolint:staticcheck //Keep old behavior
	}

	loader.statusMsg("Importing index file from %s", file)

	// read json file
	reader, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("fail to read index-pattern from file %s: %w", file, err)
	}

	var indexContent mapstr.M
	err = json.Unmarshal(reader, &indexContent)
	if err != nil {
		return fmt.Errorf("fail to unmarshal the index content from file %s: %w", file, err)
	}

	return loader.ImportIndex(indexContent)
}

// ImportIndex imports the passed index pattern to Kibana
func (loader KibanaLoader) ImportIndex(pattern mapstr.M) error {
	if loader.version.LessThan(minimumRequiredVersionSavedObjects) {
		return fmt.Errorf("kibana version must be at least %s", minimumRequiredVersionSavedObjects.String())
	}

	var errs []error

	params := url.Values{}
	params.Set("overwrite", "true")

	if err := ReplaceIndexInIndexPattern(loader.config.Index, pattern); err != nil {
		errs = append(errs, fmt.Errorf("error setting index '%s' in index pattern: %w", loader.config.Index, err))
	}

	err := loader.client.ImportMultiPartFormFile(importAPI, params, "index-template.ndjson", pattern.String())
	if err != nil {
		errs = append(errs, fmt.Errorf("error loading index pattern: %w", err))
	}
	return errors.Join(errs...)
}

// ImportDashboard imports the dashboard file
func (loader KibanaLoader) ImportDashboard(file string) error {
	if loader.version.LessThan(minimumRequiredVersionSavedObjects) {
		return fmt.Errorf("Kibana version must be at least %s", minimumRequiredVersionSavedObjects.String()) //nolint:staticcheck //Keep old behavior
	}

	loader.statusMsg("Importing dashboard from %s", file)

	params := url.Values{}
	params.Set("overwrite", "true")

	// read json file
	content, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("fail to read dashboard from file %s: %w", file, err)
	}

	content = loader.formatDashboardAssets(content)

	dashboardWithReferences, err := loader.addReferences(file, content)
	if err != nil {
		return fmt.Errorf("error getting references of dashboard: %w", err)
	}

	if err := loader.client.ImportMultiPartFormFile(importAPI, params, correctExtension(file), dashboardWithReferences); err != nil {
		return fmt.Errorf("error dashboard asset: %w", err)
	}

	loader.loadedAssets[file] = true
	return nil
}

type dashboardObj struct {
	References []dashboardReference `json:"references"`
}
type dashboardReference struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

func (loader KibanaLoader) addReferences(path string, dashboard []byte) (string, error) {
	var d dashboardObj
	err := json.Unmarshal(dashboard, &d)
	if err != nil {
		return "", fmt.Errorf("failed to parse dashboard references: %w", err)
	}

	base := filepath.Dir(path)
	var result string
	for _, ref := range d.References {
		if ref.Type == "index-pattern" {
			continue
		}
		referencePath := filepath.Join(base, "..", ref.Type, ref.ID+".json")
		if _, ok := loader.loadedAssets[referencePath]; ok {
			continue
		}
		refContents, err := os.ReadFile(referencePath)
		if err != nil {
			return "", fmt.Errorf("fail to read referenced asset from file %s: %w", referencePath, err)
		}
		refContents = loader.formatDashboardAssets(refContents)
		refContentsWithReferences, err := loader.addReferences(referencePath, refContents)
		if err != nil {
			return "", fmt.Errorf("failed to get references of %s: %w", referencePath, err)
		}

		result += refContentsWithReferences
		loader.loadedAssets[referencePath] = true
	}

	var res mapstr.M
	err = json.Unmarshal(dashboard, &res)
	if err != nil {
		return "", fmt.Errorf("failed to convert asset: %w", err)
	}
	result += res.String() + "\n"

	return result, nil
}

func (loader KibanaLoader) formatDashboardAssets(content []byte) []byte {
	content = ReplaceIndexInDashboardObject(loader.config.Index, content, loader.logger)
	content = EncodeJSONObjects(content, loader.logger)

	replacements := loader.config.StringReplacements
	if replacements == nil {
		replacements = make(map[string]string)
	}
	replacements["CHANGEME_HOSTNAME"] = loader.hostname
	for needle, replacement := range replacements {
		content = ReplaceStringInDashboard(needle, replacement, content)
	}

	return content
}

func correctExtension(file string) string {
	return filepath.Base(file[:len(file)-len("json")]) + "ndjson"
}

// Close closes the client
func (loader KibanaLoader) Close() error {
	return loader.client.Close()
}

func (loader KibanaLoader) statusMsg(msg string, a ...interface{}) {
	if loader.msgOutputter != nil {
		loader.msgOutputter(msg, a...)
	} else {
		loader.logger.Debugf(msg, a...)
	}
}
