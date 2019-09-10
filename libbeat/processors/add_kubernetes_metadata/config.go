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
	"os"
	"time"

	"k8s.io/client-go/rest"

	"github.com/elastic/beats/libbeat/common"
)

type kubeAnnotatorConfig struct {
	KubeConfig string        `config:"kube_config"`
	Host       string        `config:"host"`
	Namespace  string        `config:"namespace"`
	SyncPeriod time.Duration `config:"sync_period"`
	// Annotations are kept after pod is removed, until they haven't been accessed
	// for a full `cleanup_timeout`:
	CleanupTimeout  time.Duration `config:"cleanup_timeout"`
	Indexers        PluginConfig  `config:"indexers"`
	Matchers        PluginConfig  `config:"matchers"`
	DefaultMatchers Enabled       `config:"default_matchers"`
	DefaultIndexers Enabled       `config:"default_indexers"`
	InCluster       bool
}

type Enabled struct {
	Enabled bool `config:"enabled"`
}

type PluginConfig []map[string]common.Config

func getSystemKubeConfig() string {
	envKubeConfig := os.Getenv("KUBECONFIG")
	if _, err := os.Stat(envKubeConfig); !os.IsNotExist(err) {
		return envKubeConfig
	}
	homeKubeConfig := os.Getenv("HOME") + "/.kube/config"
	if _, err := os.Stat(homeKubeConfig); !os.IsNotExist(err) {
		return homeKubeConfig
	}
	return homeKubeConfig
}

func inCluster() bool {
	_, err := rest.InClusterConfig()
	if err == nil {
		return true
	}
	return false
}

func defaultKubernetesAnnotatorConfig() kubeAnnotatorConfig {
	return kubeAnnotatorConfig{
		KubeConfig:      getSystemKubeConfig(),
		SyncPeriod:      10 * time.Minute,
		CleanupTimeout:  60 * time.Second,
		DefaultMatchers: Enabled{true},
		DefaultIndexers: Enabled{true},
		InCluster:       inCluster(),
	}
}
