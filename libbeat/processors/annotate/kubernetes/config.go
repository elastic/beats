package kubernetes

import (
	"github.com/elastic/beats/libbeat/common"
	"time"
)

type kubeAnnotatorConfig struct {
	InCluster       bool          `config:"in_cluster"`
	KubeConfig      string        `config:"kube_config"`
	Host            string        `config:"host"`
	Namespace       string        `config:"namespace"`
	SyncPeriod      time.Duration `config:"sync_period"`
	Indexers        PluginConfig  `config:"indexers"`
	Matchers        PluginConfig  `config:"matchers"`
	DefaultMatchers Enabled       `config:"default_matchers"`
	DefaultIndexers Enabled       `config:"default_indexers"`
}

type Enabled struct {
	Enabled bool `config:"enabled"`
}

type PluginConfig []map[string]common.Config

func defaultKuberentesAnnotatorConfig() kubeAnnotatorConfig {
	return kubeAnnotatorConfig{
		InCluster:       true,
		SyncPeriod:      1 * time.Second,
		Namespace:       "kube-system",
		DefaultMatchers: Enabled{true},
		DefaultIndexers: Enabled{true},
	}
}
