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
	"os/exec"
	"strings"
	"time"

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
}

type Enabled struct {
	Enabled bool `config:"enabled"`
}

type PluginConfig []map[string]common.Config

func trimAfter(value string, a string) string {
	// Get substring after a string.
	pos := strings.LastIndex(value, a)
	if pos == -1 {
		return ""
	}
	adjustedPos := pos + len(a)
	if adjustedPos >= len(value) {
		return ""
	}
	return value[adjustedPos:len(value)]
}

func findDaemonsConfigFlag() string {
	// Search in processes and search for config in flags
	psRes, _ := exec.Command("bash", "-c", "ps aux | grep -- -kubeconfig=").Output()
	psResStr := string(psRes)
	trimmedPsResStr := trimAfter(psResStr, "--kubeconfig=")
	arr := strings.Split(trimmedPsResStr, " ")
	kubeConfigPath := arr[0]
	return kubeConfigPath
}

func getSystemKubeConfig() string {
	homeKubeConfig := os.Getenv("HOME") + "/.kube/config"
	if _, err := os.Stat(homeKubeConfig); !os.IsNotExist(err) {
		return homeKubeConfig
	}
	envKubeConfig := os.Getenv("KUBECONFIG")
	if _, err := os.Stat(envKubeConfig); !os.IsNotExist(err) {
		return envKubeConfig
	}
	kubeConfigPath := findDaemonsConfigFlag()
	if _, err := os.Stat(kubeConfigPath); !os.IsNotExist(err) {
		return kubeConfigPath
	}
	kubeletConfig := "/etc/kubernetes/kubelet.conf"
	if _, err := os.Stat(kubeletConfig); !os.IsNotExist(err) {
		return kubeletConfig
	}
	return homeKubeConfig
}

func defaultKubernetesAnnotatorConfig() kubeAnnotatorConfig {
	return kubeAnnotatorConfig{
		KubeConfig:      getSystemKubeConfig(),
		SyncPeriod:      10 * time.Minute,
		CleanupTimeout:  60 * time.Second,
		DefaultMatchers: Enabled{true},
		DefaultIndexers: Enabled{true},
	}
}
