package event

import (
	"errors"
	"time"
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

func defaultKubernetesEventsConfig() kubeEventsConfig {
	return kubeEventsConfig{
		InCluster:  true,
		SyncPeriod: 1 * time.Second,
	}
}

func (c kubeEventsConfig) Validate() error {
	if !c.InCluster && c.KubeConfig == "" {
		return errors.New("`kube_config` path can't be empty when in_cluster is set to false")
	}
	return nil
}
