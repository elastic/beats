package events

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
)

type kubeEventsConfig struct {
	InCluster  bool          `config:"in_cluster"`
	KubeConfig string        `config:"kube_config"`
	Namespace  string        `config:"namespace"`
	SyncPeriod time.Duration `config:"sync_period"`
}

type Enabled struct {
	Enabled bool `config:"enabled"`
}

type PluginConfig []map[string]common.Config

func defaultKuberentesEventsConfig() kubeEventsConfig {
	return kubeEventsConfig{
		InCluster:  true,
		SyncPeriod: 1 * time.Second,
	}
}
