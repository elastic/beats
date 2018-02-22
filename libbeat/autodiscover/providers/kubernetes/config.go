package kubernetes

import (
	"time"

	"github.com/elastic/beats/libbeat/autodiscover/template"
)

// Config for kubernetes autodiscover provider
type Config struct {
	InCluster      bool          `config:"in_cluster"`
	KubeConfig     string        `config:"kube_config"`
	Host           string        `config:"host"`
	Namespace      string        `config:"namespace"`
	SyncPeriod     time.Duration `config:"sync_period"`
	CleanupTimeout time.Duration `config:"cleanup_timeout"`

	IncludeLabels      []string `config:"include_labels"`
	ExcludeLabels      []string `config:"exclude_labels"`
	IncludeAnnotations []string `config:"include_annotations"`

	Templates template.MapperSettings `config:"templates"`
}

func defaultConfig() *Config {
	return &Config{
		InCluster:      true,
		SyncPeriod:     1 * time.Second,
		CleanupTimeout: 60 * time.Second,
	}
}
