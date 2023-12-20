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

package add_kubernetes_metadata

import (
	"fmt"
	"time"

	"github.com/elastic/elastic-agent-autodiscover/kubernetes"
	"github.com/elastic/elastic-agent-autodiscover/kubernetes/metadata"
	"github.com/elastic/elastic-agent-libs/config"
)

type kubeAnnotatorConfig struct {
	KubeConfig        string                       `config:"kube_config"`
	KubeClientOptions kubernetes.KubeClientOptions `config:"kube_client_options"`
	Node              string                       `config:"node"`
	Scope             string                       `config:"scope"`
	Namespace         string                       `config:"namespace"`
	SyncPeriod        time.Duration                `config:"sync_period"`
	// Annotations are kept after pod is removed, until they haven't been accessed
	// for a full `cleanup_timeout`:
	CleanupTimeout  time.Duration `config:"cleanup_timeout"`
	Indexers        PluginConfig  `config:"indexers"`
	Matchers        PluginConfig  `config:"matchers"`
	DefaultMatchers Enabled       `config:"default_matchers"`
	DefaultIndexers Enabled       `config:"default_indexers"`

	AddResourceMetadata *metadata.AddResourceMetadataConfig `config:"add_resource_metadata"`
}

type Enabled struct {
	Enabled bool `config:"enabled"`
}

type PluginConfig []map[string]config.C

func (k *kubeAnnotatorConfig) InitDefaults() {
	k.SyncPeriod = 10 * time.Minute
	k.CleanupTimeout = 60 * time.Second
	k.DefaultMatchers = Enabled{true}
	k.DefaultIndexers = Enabled{true}
	k.Scope = "node"
	k.AddResourceMetadata = metadata.GetDefaultResourceMetadataConfig()
}

func (k *kubeAnnotatorConfig) Validate() error {
	if k.Scope != "node" && k.Scope != "cluster" {
		return fmt.Errorf("invalid scope %s, valid values include `cluster`, `node`", k.Scope)
	}

	if k.Scope == "cluster" {
		k.Node = ""
	}

	// Checks below were added to warn the users early on and avoid initialising the processor in case the `logs_path`
	// matcher config is not valid: supported paths defined as a `logs_path` configuration setting are strictly defined
	// if `resource_type` is set
	for _, matcher := range k.Matchers {
		if matcherCfg, ok := matcher["logs_path"]; ok {
			if matcherCfg.HasField("resource_type") {
				logsPathMatcher := struct {
					LogsPath     string `config:"logs_path"`
					ResourceType string `config:"resource_type"`
				}{}

				err := matcherCfg.Unpack(&logsPathMatcher)
				if err != nil {
					return fmt.Errorf("fail to unpack the `logs_path` matcher configuration: %w", err)
				}
				if logsPathMatcher.LogsPath == "" {
					return fmt.Errorf("invalid logs_path matcher configuration: when resource_type is defined, logs_path must be set as well")
				}
				if logsPathMatcher.ResourceType != "pod" && logsPathMatcher.ResourceType != "container" {
					return fmt.Errorf("invalid resource_type %s, valid values include `pod`, `container`", logsPathMatcher.ResourceType)
				}
				if logsPathMatcher.ResourceType == "pod" && !(logsPathMatcher.LogsPath == "/var/lib/kubelet/pods/" || logsPathMatcher.LogsPath == "/var/log/pods/") {
					return fmt.Errorf("invalid logs_path defined for resource_type: %s, valid values include `/var/lib/kubelet/pods/`, `/var/log/pods/`", logsPathMatcher.ResourceType)
				}
				if logsPathMatcher.ResourceType == "container" && logsPathMatcher.LogsPath != "/var/log/containers/" {
					return fmt.Errorf("invalid logs_path defined for resource_type: %s, valid value is `/var/log/containers/`", logsPathMatcher.ResourceType)
				}
			}

		}
	}

	return nil
}
